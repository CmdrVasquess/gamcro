package internal

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/gorilla/mux"
)

var (
	log    = qbsllm.New(qbsllm.Lnormal, AppName, nil, nil)
	LogCfg = qbsllm.NewConfig(log)

	//go:embed webui
	webfs embed.FS
)

const (
	AppName    = "gamcro"
	NewDirPerm = 0750
)

type Gamcro struct {
	SrvAddr         string
	Passphr         []byte
	TLSCert, TLSKey string
	ClientAuth      AuthCreds
	singleClient    string
	MultiClient     bool
	ClientNet       string
	TxtLimit        int
}

func (g *Gamcro) Run(qrFlag bool) error {
	if g.TxtLimit <= 0 {
		g.TxtLimit = 256
	}
	webRoutes := mux.NewRouter()
	webRoutes.HandleFunc("/", handleUI)
	if staticDir, err := fs.Sub(webfs, "webui"); err != nil {
		return err
	} else {
		staticHdlr := http.FileServer(http.FS(staticDir))
		webRoutes.PathPrefix("/s/").Handler(g.auth(
			http.StripPrefix("/s/", staticHdlr).ServeHTTP,
		))
	}
	g.apiRoutes(webRoutes)
	if err := ensureTLSCert(g.TLSCert, g.TLSKey); err != nil {
		return err
	}
	log.Debuga("Load TLS `certificate`", g.TLSCert)
	log.Debuga("Load TLS `key`", g.TLSKey)
	log.Debugf("Runninig gamcro HTTPS server on %s", g.SrvAddr)
	return http.ListenAndServeTLS(g.SrvAddr, g.TLSCert, g.TLSKey, webRoutes)
}

func handleUI(wr http.ResponseWriter, rq *http.Request) {
	if rq.Method != http.MethodGet {
		http.Error(wr, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if rq.URL.Path != "/" {
		http.Error(wr, "not found", http.StatusNotFound)
	}
	http.Redirect(wr, rq, "/s/index.html", http.StatusSeeOther)
}

func (g *Gamcro) ConnectHint() string {
	if g.SrvAddr == "" {
		return ""
	}
	var svcurl string
	if g.SrvAddr[0] != ':' {
		svcurl = fmt.Sprintf("https://%s/", g.SrvAddr)
	} else {
		// With UDP we do NOT connect or send to that address (AFAIK)
		conn, _ := net.Dial("udp", "8.8.8.8:80")
		defer conn.Close()
		addr := conn.LocalAddr().(*net.UDPAddr)
		svcurl = fmt.Sprintf("https://%s%s/", addr.IP, g.SrvAddr)
	}
	return svcurl
}
