package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/gorilla/mux"
)

//go:generate versioner -bno build_no VERSION version.go

var (
	srvAddr         string
	tlsCert, tlsKey = "cert.pem", "key.pem"
	authCreds       string
	inLimit         = 256
	fLog            string

	log    = qbsllm.New(qbsllm.Lnormal, "gamcro", nil, nil)
	logCfg = qbsllm.NewConfig(log)

	//go:embed assets
	assets embed.FS
)

func asset(p string) string {
	return path.Join("assets", p)
}

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
	rd, err := assets.Open(asset("banner.txt"))
	if err != nil {
		log.Fatale(err)
	}
	defer rd.Close()
	io.Copy(os.Stdout, rd)
	fmt.Printf("v%d.%d.%d [%s #%d]\n", Major, Minor, Patch, Quality, BuildNo)
}

type dh int

func (_ dh) ServeHTTP(wr http.ResponseWriter, rq *http.Request) {
	log.Infoa("rq `path`", rq.URL.Path)
}

func main() {
	showBanner()
	flag.StringVar(&srvAddr, "addr", ":9420", "server address")
	flag.StringVar(&tlsCert, "cert", tlsCert, "TLS cert file to use for HTTPS")
	flag.StringVar(&tlsKey, "key", tlsKey, "TLS key file to use for HTTPS")
	flag.StringVar(&authCreds, "auth", "", "basic auth")
	flag.IntVar(&inLimit, "lim", inLimit, "Limit the length of input")
	flag.StringVar(&fLog, "log", "", "set log verbosity")
	flag.Parse()
	c4hgol.SetLevel(logCfg, fLog, nil)

	webRoutes := mux.NewRouter()
	webRoutes.HandleFunc("/", handleUI)
	if staticDir, err := fs.Sub(assets, "assets"); err != nil {
		log.Fatale(err)
	} else {
		staticHdlr := http.FileServer(http.FS(staticDir))
		webRoutes.PathPrefix("/s/").Handler(http.StripPrefix("/", staticHdlr))
	}
	apiRoutes(webRoutes)

	if err := ensureCreds(); err != nil {
		log.Fatale(err)
	}
	if err := ensureTLSCert(tlsCert, tlsKey); err != nil {
		log.Fatale(err)
	}
	log.Infoa("Load TLS `certificate`", tlsCert)
	log.Infoa("Load TLS `key`", tlsKey)
	log.Infof("Runninig gamcro HTTPS server on %s", srvAddr)
	connectHint()
	log.Fatale(http.ListenAndServeTLS(srvAddr, tlsCert, tlsKey, webRoutes))
}

func handleUI(wr http.ResponseWriter, rq *http.Request) {
	if rq.Method != http.MethodGet {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if rq.URL.Path != "/" {
		http.Error(wr, "not found", http.StatusNotFound)
	}
	http.Redirect(wr, rq, "/s/ui.html", http.StatusSeeOther)
}

func connectHint() {
	if srvAddr == "" {
		return
	}
	if srvAddr[0] != ':' {
		log.Infof("Use https://%s/ to connect your browser to the Web UI", srvAddr)
		return
	}
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	log.Infof("Use https://%s%s/ to connect your browser to the Web UI",
		addr.IP,
		srvAddr,
	)
	// addrs, _ := net.InterfaceAddrs()
	// for _, addr := range addrs {
	// 	ipn, ok := addr.(*net.IPNet)
	// 	if !ok || ipn.IP.IsLoopback() {
	// 		continue
	// 	}
	// 	fmt.Printf("%s: %s\n", ipn.Network(), ipn)
	// }
}
