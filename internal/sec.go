package internal

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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

	"golang.org/x/crypto/argon2"

	"git.fractalqb.de/fractalqb/pack/ospath"
	"github.com/gofrs/uuid"
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
	err = cryptWriteFile(key, passphrase, pem.EncodeToMemory(block))
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

type CryptError struct {
	op  string
	err error
}

func (ce CryptError) Error() string {
	return fmt.Sprintf("crypt %s error: %s", ce.op, ce.err)
}

func cryptWriteFile(name string, passwd, data []byte) error {
	wr, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer wr.Close()
	return cryptWrite(wr, passwd, data)
}

const (
	cryptIOVersion   = 2
	cryptSaltSize    = 16
	cryptKDFMem      = 64 * 1024
	cryptKDFIter     = 3
	cryptKDFParallel = 2
	cryptKDFKeyLen   = 32
)

func mkKey(passwd, salt []byte) (key, nsalt []byte, err error) {
	if salt == nil {
		salt = make([]byte, cryptSaltSize)
		if _, err = rand.Read(salt); err != nil {
			return nil, nil, err
		}
	}
	key = argon2.IDKey(
		passwd,
		salt,
		cryptKDFIter,
		cryptKDFMem,
		cryptKDFParallel,
		cryptKDFKeyLen,
	)
	return key, salt, nil
}

func cryptWrite(wr io.Writer, passwd, data []byte) error {
	if len(passwd) == 0 {
		if _, err := wr.Write([]byte{0}); err != nil {
			return err
		}
		_, err := wr.Write(data)
		return err
	}
	key, salt, err := mkKey([]byte(passwd), nil)
	if err != nil {
		return CryptError{"write", err}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return CryptError{"write", err}
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return CryptError{"write", err}
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return CryptError{"write", err}
	}
	ciph := aesgcm.Seal(nil, nonce, data, nil)
	if _, err := wr.Write([]byte{cryptIOVersion}); err != nil {
		return CryptError{"write", err}
	}
	if _, err = wr.Write(salt); err != nil {
		return CryptError{"write", err}
	}
	if _, err = wr.Write(nonce); err != nil {
		return CryptError{"write", err}
	}
	if _, err = wr.Write(ciph); err != nil {
		return CryptError{"write", err}
	}
	return nil
}

func cryptReadFile(name string, passwd []byte) ([]byte, error) {
	rd, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return cryptRead(rd, passwd)
}

func cryptRead(rd io.Reader, passwd []byte) ([]byte, error) {
	if len(passwd) == 0 {
		var buf bytes.Buffer
		if _, err := io.CopyN(&buf, rd, 1); err != nil {
			return nil, err
		}
		if buf.Bytes()[0] != 0 {
			return nil, fmt.Errorf(
				"detected crypt IO version %d for cleartext read",
				buf.Bytes()[0],
			)
		}
		buf.Reset()
		if _, err := io.Copy(&buf, rd); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	var salt bytes.Buffer
	if _, err := io.CopyN(&salt, rd, 1); err != nil {
		return nil, CryptError{"read", err}
	}
	if salt.Bytes()[0] != cryptIOVersion {
		return nil, CryptError{
			"read",
			fmt.Errorf(
				"detected crypt IO version %d instead of %d",
				salt.Bytes()[0],
				cryptIOVersion,
			),
		}
	}
	salt.Reset()
	if _, err := io.CopyN(&salt, rd, cryptSaltSize); err != nil {
		return nil, CryptError{"read", err}
	}
	key, _, _ := mkKey([]byte(passwd), salt.Bytes())
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, CryptError{"read", err}
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, CryptError{"read", err}
	}
	var nonce bytes.Buffer
	if _, err := io.CopyN(&nonce, rd, int64(aesgcm.NonceSize())); err != nil {
		return nil, CryptError{"read", err}
	}
	var ciph bytes.Buffer
	if _, err := io.Copy(&ciph, rd); err != nil {
		return nil, CryptError{"read", err}
	}
	plaintext, err := aesgcm.Open(nil, nonce.Bytes(), ciph.Bytes(), nil)
	if err != nil {
		return nil, CryptError{"read", err}
	}
	return plaintext, nil
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
	key, salt, err := mkKey([]byte(passwd), nil)
	if err != nil {
		return err
	}
	ac.user = user
	ac.salt = salt
	ac.pass = key
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
	h, _, _ := mkKey([]byte(passwd), ac.salt)
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

func (g *Gamcro) releaseClient(wr http.ResponseWriter, rq *http.Request) {
	log.Infoa("Release `client`", g.singleClient)
	g.singleClient = ""
	wr.WriteHeader(http.StatusNoContent)
}

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
