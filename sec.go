package main

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
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

func ensureCreds() (err error) {
	switch {
	case cfg.authCreds == "":
		authFile := paths.LocalData(defaultCredsFile)
		if _, err := os.Stat(authFile); err == nil {
			log.Infoa("HTTP basic auth configuration `file` detected", authFile)
			log.Infoa("You can use -auth flag for different settings", authFile)
			creds, err := readCredsFile(authFile)
			if err == nil {
				cfg.authCreds = creds
				return nil
			}
			log.Warne(err)
		}
		return userInputCreds()
	case cfg.authCreds == ":":
		return userInputCreds()
	case strings.IndexByte(cfg.authCreds, ':') >= 0:
		log.Warns("It is not secure to set passwords on the command line!")
		me := filepath.Base(os.Args[0])
		log.Infof("Better use '%s -auth <filename>' with restricted access to <filename>", me)
		return nil
	default:
		creds, err := readCredsFile(cfg.authCreds)
		if err != nil {
			return err
		}
		cfg.authCreds = creds
		return nil
	}
}

func readCredsFile(name string) (string, error) {
	log.Infoa("Read HTTP basic auth user:password from `file`", name)
	rd, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer rd.Close()
	var res string
	err = cryptRead(rd, cfg.passphr, func(rd io.Reader) error {
		scan := bufio.NewScanner(rd)
		if !scan.Scan() {
			return fmt.Errorf("auth file '%s' is empty", name)
		}
		res = scan.Text()
		return nil
	})
	if err == nil {
		if strings.IndexByte(res, ':') < 0 {
			err = fmt.Errorf("invalid credentials in file '%s'", name)
		}
	} else if err == io.EOF {
		err = fmt.Errorf("failed to read credentials from '%s': %s", name, err)
	}
	return res, err
}

func userInputCreds() (err error) {
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
	cfg.authCreds = usr + ":" + string(pass1)
	if len(cfg.passphr) > 0 {
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
			wr, err := os.Create(authFile)
			if err != nil {
				return err
			}
			defer wr.Close()
			err = cryptWrite(wr, cfg.passphr, func(wr io.Writer) error {
				_, err = fmt.Fprintln(wr, cfg.authCreds)
				return err
			})
			return err
		}
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

func newLocalNetList() (res localNetList) {
	ifs, _ := net.Interfaces()
	for _, i := range ifs {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok {
				res = append(res, ipnet)
			}
		}
	}
	return res
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

func auth(h http.HandlerFunc) http.HandlerFunc {
	return func(wr http.ResponseWriter, rq *http.Request) {
		if emsg, code := checkClient(rq); code != 0 {
			http.Error(wr, emsg, code)
			return
		}
		user, pass, ok := rq.BasicAuth()
		if !ok {
			wr.Header().Set(
				"WWW-Authenticate",
				`Basic realm="Gamcro Client Authentication"`,
			)
			http.Error(wr, "Unauthorized", http.StatusUnauthorized)
			return
		} else if ba := user + ":" + pass; ba != cfg.authCreds {
			log.Warna("Failed basic auth with `user` and `password` from `client`",
				user,
				pass,
				rq.RemoteAddr)
			s := time.Duration(1000 + mrand.Intn(2000))
			time.Sleep(s * time.Millisecond)
			http.Error(wr, "Forbidden", http.StatusForbidden)
			return
		} else {
			log.Debuga("Authorized `client`", rq.RemoteAddr)
		}
		if cfg.singleClient == "" {
			h, _, err := net.SplitHostPort(rq.RemoteAddr)
			if err != nil {
				log.Errore(err)
				http.Error(wr, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			cfg.singleClient = h
		}
		h(wr, rq)
	}
}
