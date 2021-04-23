package main

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
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"git.fractalqb.de/fractalqb/pack/ospath"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

func init() {
	localNets = newLocalNetList()
}

func newTLSCert(cert, key, commonName string) (err error) {
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

	if _, err = ospath.ProvideDir(newDirPerm, cert); err != nil {
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
	if _, err = ospath.ProvideDir(newDirPerm, key); err != nil {
		return fmt.Errorf("create key-file '%s': %s", key, err)
	}
	wr, err = os.OpenFile(key, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create key-file '%s': %s", key, err)
	}
	defer wr.Close()
	err = pem.Encode(wr, block)
	if err != nil {
		return fmt.Errorf("write key-file '%s': %s", key, err)
	}
	err = wr.Close()
	if err != nil {
		return fmt.Errorf("close key-file '%s': %s", key, err)
	}

	return nil
}

func ensureTLSCert(cert, key string) error {
	_, certErr := os.Stat(cert)
	_, keyErr := os.Stat(key)
	if !os.IsNotExist(certErr) || !os.IsNotExist(keyErr) {
		return nil
	}
	return newTLSCert(cert, key, "JV:Gamcro")
}

const defaultCredsFile = "auth.txt"

func ensureCreds(flag string) (err error) {
	colonIdx := strings.IndexByte(flag, ':')
	switch {
	case flag == "":
		authFile := paths.LocalData(defaultCredsFile)
		if _, err = os.Stat(authFile); err == nil {
			log.Infoa("HTTP basic auth configuration `file` detected", authFile)
			log.Infoa("You can use -auth flag for different settings", authFile)
			err := cfg.clientAuth.readFile(authFile)
			if err == nil {
				return nil
			}
			log.Warne(err)
		}
		err = userInputCreds(&cfg.clientAuth)
	case flag == ":":
		err = userInputCreds(&cfg.clientAuth)
	case colonIdx >= 0:
		if colonIdx < len(flag)-1 {
			me := filepath.Base(os.Args[0])
			log.Warns("It is not secure to set passwords on the command line!")
			log.Infof("Better use '%s -auth <filename>' with restricted access to <filename>", me)
		}
	default:
		err = cfg.clientAuth.readFile(flag)
	}
	return err
	// if strings.HasSuffix(flag, ":") {
	// 	log.Warns("Cannot accept empty password")
	// 	passwd := makeRandStr(passwdChars, 7)
	// 	log.Infof("Using one-time password \"%s\"", passwd)
	// 	cfg.clientAuth.set(flag[:len(flag)-1], passwd)
	// }
}

func userInputCreds(creds *authCreds) (err error) {
	log.Infos("Need user and password for HTTP basic auth")
	var usr string
	fmt.Print("Enter HTTP basic auth user: ")
	if _, err = fmt.Scan(&usr); err != nil {
		return err
	}
	var pass1 []byte
	for {
		fmt.Print("Enter HTTP basic auth password: ")
		pass1, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		fmt.Print("\nRepeat password: ")
		pass2, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		if reflect.DeepEqual(pass1, pass2) {
			break
		}
		fmt.Println()
		log.Infos("Passwords missmatch")
	}
	cfg.clientAuth.set(usr, string(pass1))
	authFile := paths.LocalData(defaultCredsFile)
	fmt.Printf("\nSave user:password to '%s' (y/N)?", authFile)
	var answer string
	if _, err = fmt.Scan(&answer); err != nil {
		return err
	}
	if l := strings.ToLower(answer); l == "y" || l == "yes" {
		if _, err := ospath.ProvideDir(newDirPerm, authFile); err != nil {
			return err
		}
		err = cfg.clientAuth.writeFile(authFile)
		return err
	}
	fmt.Println()
	return nil
}

func cryptWrite(wr io.Writer, passwd []byte, do func(io.Writer) error) error {
	if len(passwd) == 0 {
		return do(wr)
	}
	awr, err := armor.Encode(wr, "MESSAGE", nil)
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

func checkClient(rq *http.Request) (e string, code int) {
	chost, _, err := net.SplitHostPort(rq.RemoteAddr)
	if err != nil {
		log.Errore(err)
		return "Internal Server Error", http.StatusInternalServerError
	}
	if cfg.clientNet != "all" {
		if cip := net.ParseIP(chost); cip == nil {
			log.Errora("Cannot parse `client host`", chost)
			return "Internal Server Error", http.StatusInternalServerError
		} else if !localNets.contains(cip) {
			log.Warna("Non-local `client` connect blocked", rq.RemoteAddr)
			return "Forbidden", http.StatusForbidden
		}
	}
	if !cfg.multiClient && cfg.singleClient != "" {
		if chost != cfg.singleClient {
			log.Warna("A 2nd `client` machine was blocked", rq.RemoteAddr)
			log.Infoa("Current `client`", cfg.singleClient)
			return "Forbidden", http.StatusForbidden
		}
	}
	return "", 0
}

const realmChars = "0123456789ABCDEFGHJKLMNPQRTUVW"
const passwdChars = "0123456789ABCDEFGHJKLMNPQRTUVWabcdefghjklmnpqrtuvw!#$%&+,.-/:;=_~"

func makeRandStr(chars string, strlen int) string {
	charNo := big.NewInt(int64(len(chars)))
	var sb strings.Builder
	for strlen > 0 {
		c, err := rand.Int(rand.Reader, charNo)
		if err != nil {
			log.Fatale(err)
		}
		sb.WriteByte(chars[c.Uint64()])
		strlen--
	}
	return sb.String()
}

type authCreds struct {
	user string
	salt []byte
	pass []byte
}

const (
	authIter   = 4096
	authKeyLen = 24
)

func (ac *authCreds) set(user, passwd string) (err error) {
	if ac.salt == nil {
		ac.salt = make([]byte, 12)
	}
	if _, err = rand.Read(ac.salt); err != nil {
		return err
	}
	ac.user = user
	ac.pass = pbkdf2.Key([]byte(passwd), ac.salt, authIter, authKeyLen, sha256.New)
	return nil
}

func (ac *authCreds) check(user, passwd string) bool {
	if ac.user != user {
		return false
	}
	if len(ac.salt) == 0 {
		return string(ac.pass) == passwd
	}
	h := pbkdf2.Key([]byte(passwd), ac.salt, authIter, authKeyLen, sha256.New)
	return subtle.ConstantTimeCompare(h, ac.pass) == 1
}

func (ac *authCreds) writeFile(name string) error {
	tmpf := name + "~"
	wr, err := os.Create(tmpf)
	if err != nil {
		return err
	}
	defer wr.Close()
	err = cryptWrite(wr, cfg.passphr, func(wr io.Writer) error {
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
		return err
	})
	if err := wr.Close(); err != nil {
		return err
	}
	return os.Rename(tmpf, name)
}

func (ac *authCreds) readFile(name string) error {
	log.Infoa("Read HTTP basic auth user:password from `file`", name)
	rd, err := os.Open(name)
	if err != nil {
		return err
	}
	defer rd.Close()
	err = cryptRead(rd, cfg.passphr, func(rd io.Reader) error {
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
			return ac.set(parts[0], parts[1])
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
	})
	return err
}

var (
	currentRealmKey = makeRandStr(realmChars, 6)
	basicRealm      = fmt.Sprintf(`Basic realm="Gamcro: %s"`, currentRealmKey)
)

func auth(h http.HandlerFunc) http.HandlerFunc {
	return func(wr http.ResponseWriter, rq *http.Request) {
		if emsg, code := checkClient(rq); code != 0 {
			http.Error(wr, emsg, code)
			return
		}
		user, pass, ok := rq.BasicAuth()
		if !ok {
			wr.Header().Set("WWW-Authenticate", basicRealm)
			http.Error(wr, "Unauthorized", http.StatusUnauthorized)
			return
		} else if !cfg.clientAuth.check(user, pass) {
			log.Warna("Failed basic auth with `user` and `password` from `client`",
				user,
				pass,
				rq.RemoteAddr)
			s := time.Duration(1000 + mrand.Intn(2000))
			time.Sleep(s * time.Millisecond)
			http.Error(wr, "Forbidden", http.StatusForbidden)
			return
		} else {
			log.Tracea("Authorized `client`", rq.RemoteAddr)
		}
		if cfg.singleClient == "" {
			h, _, err := net.SplitHostPort(rq.RemoteAddr)
			if err != nil {
				log.Errore(err)
				http.Error(wr, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			log.Infoa("Locked to single `client address`", h)
			cfg.singleClient = h
		}
		h(wr, rq)
	}
}
