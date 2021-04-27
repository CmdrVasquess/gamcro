package internal

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"git.fractalqb.de/fractalqb/pack/ospath"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/pbkdf2"
)

func init() {
	localNets = newLocalNetList()
}

func newTLSCert(cert, key, commonName string, passphrase []byte) (err error) {
	log.Infoa("Create self signed `certificate` with `key` as `common name`",
		cert,
		key,
		commonName)
	pKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}
	validStart := time.Now()
	validTil := validStart.Add(10 * 365 * 24 * time.Hour) // ~ 10 years
	serNo, err := uuid.NewV4()
	if err != nil {
		return err
	}
	cerTmpl := x509.Certificate{
		SerialNumber:          new(big.Int).SetBytes(serNo.Bytes()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             validStart,
		NotAfter:              validTil,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	cerDer, err := x509.CreateCertificate(
		rand.Reader,
		&cerTmpl, &cerTmpl,
		pKey.Public(), pKey)
	if err != nil {
		return fmt.Errorf("create cert: %s", err)
	}

	if _, err = ospath.ProvideDir(NewDirPerm, cert); err != nil {
		return fmt.Errorf("create cert-file '%s': %s", cert, err)
	}
	wr, err := os.Create(cert)
	if err != nil {
		return fmt.Errorf("create cert-file '%s': %s", cert, err)
	}
	defer wr.Close()
	err = pem.Encode(wr, &pem.Block{Type: "CERTIFICATE", Bytes: cerDer})
	if err != nil {
		return fmt.Errorf("write cert to '%s': %s", cert, err)
	}
	err = wr.Close()
	if err != nil {
		return fmt.Errorf("close cert-file '%s': %s", cert, err)
	}

	ecpem, err := x509.MarshalECPrivateKey(pKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %s", err)
	}
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: ecpem}
	if _, err = ospath.ProvideDir(NewDirPerm, key); err != nil {
		return fmt.Errorf("create key-file '%s': %s", key, err)
	}
	err = cryptWriteFile(key, passphrase, func(wr io.Writer) error {
		return pem.Encode(wr, block)
	})
	if err != nil {
		fmt.Errorf("write key-file '%s': %s", key, err)
	}
	return err
}

func ensureTLSCert(cert, key string, passpharse []byte) error {
	_, certErr := os.Stat(cert)
	_, keyErr := os.Stat(key)
	if !os.IsNotExist(certErr) || !os.IsNotExist(keyErr) {
		return nil
	}
	return newTLSCert(cert, key, "JV:Gamcro", passpharse)
}

const DefaultCredsFile = "auth.txt"

func cryptWriteFile(name string, passwd []byte, do func(io.Writer) error) error {
	wr, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer wr.Close()
	return cryptWrite(wr, passwd, do)
}

func cryptWrite(wr io.Writer, passwd []byte, do func(io.Writer) error) error {
	if len(passwd) == 0 {
		return do(wr)
	}
	awr, err := armor.Encode(wr, "PGP MESSAGE", nil)
	if err != nil {
		return err
	}
	defer awr.Close()
	ewr, err := openpgp.SymmetricallyEncrypt(awr, passwd, nil, nil)
	if err != nil {
		return err
	}
	defer ewr.Close()
	return do(ewr)
}

func cryptReadFile(name string, passwd []byte, do func(io.Reader) error) error {
	rd, err := os.Open(name)
	if err != nil {
		return err
	}
	defer rd.Close()
	return cryptRead(rd, passwd, do)
}

func cryptRead(rd io.Reader, passwd []byte, do func(io.Reader) error) error {
	if len(passwd) == 0 {
		return do(rd)
	}
	ard, err := armor.Decode(rd)
	if err != nil {
		return err
	}
	md, err := openpgp.ReadMessage(ard.Body,
		nil,
		func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
			return passwd, nil
		},
		nil,
	)
	if err != nil {
		return err
	}
	return do(md.UnverifiedBody)
}

type localNetList []*net.IPNet

func newLocalNetList() (line localNetList) {
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok {
				line = append(line, ipnet)
			}
		}
	}
	return line
}

func (lnl localNetList) contains(ip net.IP) bool {
	for _, ln := range lnl {
		if ln.Contains(ip) {
			log.Tracea("`client` on `network`", ip, ln)
			return true
		}
	}
	return false
}

var localNets localNetList

func (g *Gamcro) checkClient(rq *http.Request) (e string, code int) {
	chost, _, err := net.SplitHostPort(rq.RemoteAddr)
	if err != nil {
		log.Errore(err)
		return "Internal Server Error", http.StatusInternalServerError
	}
	if g.ClientNet != "all" {
		if cip := net.ParseIP(chost); cip == nil {
			log.Errora("Cannot parse `client host`", chost)
			return "Internal Server Error", http.StatusInternalServerError
		} else if !localNets.contains(cip) {
			log.Warna("Non-local `client` connect blocked", rq.RemoteAddr)
			return "Forbidden", http.StatusForbidden
		}
	}
	if !g.MultiClient && g.singleClient != "" {
		if chost != g.singleClient {
			log.Warna("A 2nd `client` machine was blocked", rq.RemoteAddr)
			log.Infoa("Current `client`", g.singleClient)
			return "Forbidden", http.StatusForbidden
		}
	}
	return "", 0
}

const realmChars = "0123456789ABCDEFGHJKLMNPQRTUVW"
const passwdChars = "0123456789ABCDEFGHJKLMNPQRTUVWabcdefghjklmnpqrtuvw!#$%&+,.-/:;=_~"

func makeRandStr(chars string, strlen int) (string, error) {
	charNo := big.NewInt(int64(len(chars)))
	var sb strings.Builder
	for strlen > 0 {
		c, err := rand.Int(rand.Reader, charNo)
		if err != nil {
			return "", err
		}
		sb.WriteByte(chars[c.Uint64()])
		strlen--
	}
	return sb.String(), nil
}

func mustMakeRandStr(chars string, strlen int) string {
	res, err := makeRandStr(chars, strlen)
	if err != nil {
		panic(err)
	}
	return res
}

type AuthCreds struct {
	user string
	salt []byte
	pass []byte
	ctpw string
}

const (
	authIter   = 4096
	authKeyLen = 24
)

func (ac *AuthCreds) Set(user, passwd string) (err error) {
	if ac.salt == nil {
		ac.salt = make([]byte, 12)
	}
	if _, err = rand.Read(ac.salt); err != nil {
		return err
	}
	ac.user = user
	ac.pass = pbkdf2.Key([]byte(passwd), ac.salt, authIter, authKeyLen, sha256.New)
	ac.ctpw = ""
	return nil
}

func (ac *AuthCreds) check(user, passwd string) bool {
	if ac.user != user {
		return false
	}
	if ac.ctpw != "" {
		return passwd == ac.ctpw
	}
	if len(ac.salt) == 0 {
		return string(ac.pass) == passwd
	}
	h := pbkdf2.Key([]byte(passwd), ac.salt, authIter, authKeyLen, sha256.New)
	res := subtle.ConstantTimeCompare(h, ac.pass) == 1
	if res {
		ac.ctpw = passwd
	}
	return res
}

func (ac *AuthCreds) WriteFile(name string) error {
	tmpf := name + "~"
	wr, err := os.Create(tmpf)
	if err != nil {
		return err
	}
	defer wr.Close()
	if len(ac.pass) == 0 {
		_, err = fmt.Fprintf(wr, "%s:%s", ac.user, string(ac.pass))
		return err
	}
	if _, err = fmt.Fprintln(wr, ac.user); err != nil {
		return err
	}
	_, err = fmt.Fprintln(wr, base64.StdEncoding.EncodeToString(ac.salt))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(wr, base64.StdEncoding.EncodeToString(ac.pass))
	if err != nil {
		return err
	}
	if err := wr.Close(); err != nil {
		return err
	}
	return os.Rename(tmpf, name)
}

func (ac *AuthCreds) ReadFile(name string) error {
	log.Debuga("Read HTTP basic auth user:password from `file`", name)
	ac.ctpw = ""
	rd, err := os.Open(name)
	if err != nil {
		return err
	}
	defer rd.Close()
	scan := bufio.NewScanner(rd)
	if !scan.Scan() {
		return fmt.Errorf("auth file '%s' is empty", name)
	}
	line := scan.Text()
	if strings.IndexByte(line, ':') >= 0 {
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid cleartext credentials line in %s", name)
		}
		log.Warns("Creadential file in cleartext form.")
		log.Infos("Use Gamcro interactive input to create file with hashed password.")
		return ac.Set(parts[0], parts[1])
	}
	ac.user = line
	if !scan.Scan() {
		return fmt.Errorf("premature end of auth file %s", name)
	}
	ac.salt, err = base64.StdEncoding.DecodeString(scan.Text())
	if err != nil {
		return err
	}
	if !scan.Scan() {
		return fmt.Errorf("premature end of auth file %s", name)
	}
	ac.pass, err = base64.StdEncoding.DecodeString(scan.Text())
	return err
}

var (
	CurrentRealmKey = mustMakeRandStr(realmChars, 6)
	basicRealm      = fmt.Sprintf(`Basic realm="Gamcro: %s"`, CurrentRealmKey)
)

func (g *Gamcro) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(wr http.ResponseWriter, rq *http.Request) {
		if emsg, code := g.checkClient(rq); code != 0 {
			http.Error(wr, emsg, code)
			return
		}
		user, pass, ok := rq.BasicAuth()
		if !ok {
			wr.Header().Set("WWW-Authenticate", basicRealm)
			http.Error(wr, "Unauthorized", http.StatusUnauthorized)
			return
		} else if !g.ClientAuth.check(user, pass) {
			log.Warna("Failed basic auth for `user` from `client`",
				user,
				rq.RemoteAddr)
			s := time.Duration(1000 + mrand.Intn(2000))
			time.Sleep(s * time.Millisecond)
			http.Error(wr, "Forbidden", http.StatusForbidden)
			return
		} else {
			log.Tracea("Authorized `client`", rq.RemoteAddr)
		}
		if g.singleClient == "" {
			h, _, err := net.SplitHostPort(rq.RemoteAddr)
			if err != nil {
				log.Errore(err)
				http.Error(wr, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			log.Infoa("Locked to single `client address`", h)
			g.singleClient = h
		}
		h(wr, rq)
	}
}
