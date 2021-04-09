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
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/term"
)

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

func ensureCreds() (err error) {
	if authCreds == "" {
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
		authCreds = usr + ":" + string(pass1)
		log.Infos(authCreds)
	}
	if strings.IndexByte(authCreds, ':') >= 0 {
		log.Warns("It is not secure to set passwords on the command line!")
		me := filepath.Base(os.Args[0])
		log.Infof("Better use '%s -auth <filename>' with restricted access to <filename>", me)
		return nil
	}
	log.Infoa("Read HTTP basic auth user:password from `file`", authCreds)
	rd, err := os.Open(authCreds)
	if err != nil {
		return err
	}
	defer rd.Close()
	scan := bufio.NewScanner(rd)
	if !scan.Scan() {
		return fmt.Errorf("auth file '%s' is empty", authCreds)
	}
	authCreds = scan.Text()
	return nil
}
