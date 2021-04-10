package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/atotto/clipboard"
	rogo "github.com/go-vgo/robotgo"
)

//go:generate versioner -bno build_no VERSION version.go

var (
	serv            string
	tlsCert, tlsKey = "cert.pem", "key.pem"
	authCreds       string
	inLimit         = 256
	fLog            string

	log    = qbsllm.New(qbsllm.Lnormal, "gamcro", nil, nil)
	logCfg = qbsllm.NewConfig(log)

	//go:embed ui.html
	ui []byte
	//go:embed banner.txt
	banner string
)

func auth(h http.HandlerFunc) http.HandlerFunc {
	return func(wr http.ResponseWriter, rq *http.Request) {
		if authCreds != "" {
			user, pass, ok := rq.BasicAuth()
			if !ok {
				wr.Header().Set("WWW-Authenticate", `Basic realm="EDPC Event Receiver"`)
				http.Error(wr, "Unauthorized", http.StatusUnauthorized)
				return
			} else if ba := user + ":" + pass; ba != authCreds {
				log.Warna("Failed basic auth with `user` and `password` from `client`",
					user,
					pass,
					rq.RemoteAddr)
				s := time.Duration(1000 + rand.Intn(2000))
				time.Sleep(s * time.Millisecond)
				http.Error(wr, "Forbidden", http.StatusForbidden)
				return
			}
		}
		h(wr, rq)
	}
}

func showBanner() {
	os.Stdout.WriteString(banner)
	fmt.Printf("v%d.%d.%d [%s #%d]\n", Major, Minor, Patch, Quality, BuildNo)
}

func main() {
	showBanner()
	flag.StringVar(&serv, "addr", ":9420", "server address")
	flag.StringVar(&tlsCert, "cert", tlsCert, "TLS cert file to use for HTTPS")
	flag.StringVar(&tlsKey, "key", tlsKey, "TLS key file to use for HTTPS")
	flag.StringVar(&authCreds, "auth", "", "basic auth")
	flag.IntVar(&inLimit, "lim", inLimit, "Limit the length of input")
	flag.StringVar(&fLog, "log", "", "set log verbosity")
	flag.Parse()
	c4hgol.SetLevel(logCfg, fLog, nil)

	http.HandleFunc("/", handleUI)
	//	http.HandleFunc("/gamcro", handleMacro)
	http.HandleFunc("/type-str", auth(handleTypeStr))
	http.HandleFunc("/clip", auth(handleClipStr))

	if err := ensureCreds(); err != nil {
		log.Fatale(err)
	}
	if err := ensureTLSCert(tlsCert, tlsKey); err != nil {
		log.Fatale(err)
	}
	log.Infoa("Load TLS `certificate`", tlsCert)
	log.Infoa("Load TLS `key`", tlsKey)
	log.Infof("Runninig gamcro HTTPS server on %s", serv)
	log.Fatale(http.ListenAndServeTLS(serv, tlsCert, tlsKey, nil))
}

func handleUI(wr http.ResponseWriter, rq *http.Request) {
	wr.Write(ui)
}

func rqBody(rq *http.Request) ([]byte, error) {
	var rd io.Reader = rq.Body
	if inLimit > 0 {
		rd = io.LimitReader(rd, int64(inLimit))
	}
	return io.ReadAll(rd)
}

func handleTypeStr(wr http.ResponseWriter, rq *http.Request) {
	body, err := rqBody(rq)
	if err == nil && len(body) > 0 {
		s := string(body)
		log.Debuga("`type-str`", s)
		rogo.TypeStr(s)
	} else if err != nil {
		log.Errora("Read body failed with `err`", err)
	}
}

func handleClipStr(wr http.ResponseWriter, rq *http.Request) {
	body, err := rqBody(rq)
	if err == nil && len(body) > 0 {
		s := string(body)
		log.Debuga("`cpip`", s)
		if err = clipboard.WriteAll(s); err != nil {
			log.Errore(err)
		}
	} else if err != nil {
		log.Errora("Read body failed with `err`", err)
	}
}
