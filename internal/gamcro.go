package internal

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/gorilla/mux"
)

var (
	log    = qbsllm.New(qbsllm.Lnormal, AppName, nil, nil)
	LogCfg = c4hgol.Config(
		qbsllm.NewConfig(log),
		qbsllm.NewConfig(mlog),
	)

	//go:embed webui
	webfs embed.FS
)

const (
	AppName    = "gamcro"
	NewDirPerm = 0750

	DefaultTextsDir = "texts"
)

type Gamcro struct {
	SrvAddr         string
	Passphr         []byte `json:"-"`
	TLSCert, TLSKey string
	ClientAuth      AuthCreds
	singleClient    string
	MultiClient     bool
	ClientNet       string
	TxtLimit        int
	APIs            GamcroAPI
	TextsDir        string
}

func (g *Gamcro) Run() error {
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
	webRoutes.HandleFunc("/config", g.auth(g.handleConfig)).
		Methods(http.MethodGet)
	webRoutes.HandleFunc("/client/release", g.auth(g.releaseClient))
	g.apiRoutes(webRoutes)
	if err := ensureTLSCert(g.TLSCert, g.TLSKey, g.Passphr); err != nil {
		return err
	}
	log.Debuga("Load TLS `certificate`", g.TLSCert)
	log.Debuga("Load TLS `key`", g.TLSKey)
	log.Debugf("Runninig gamcro HTTPS server on %s", g.SrvAddr)
	return g.listenAndServeTLS(webRoutes)
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

// Inspred by https://gist.github.com/tjamet/c9a53127c9bec54f62ed94685de85875
func (g *Gamcro) listenAndServeTLS(handler http.Handler) error {
	certPEMBlock, err := os.ReadFile(g.TLSCert)
	if err != nil {
		return err
	}
	keyPEMBlock, err := cryptReadFile(g.TLSKey, g.Passphr)
	if err != nil {
		return fmt.Errorf("read %s: %s", g.TLSKey, err)
	}
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return err
	}
	addr := g.SrvAddr
	if addr == "" {
		addr = ":https"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	return server.ServeTLS(ln, "", "")
}

func (g *Gamcro) handleConfig(wr http.ResponseWriter, rq *http.Request) {
	cfg := struct {
		Version     string
		APIs        []string
		MultiClient bool
		MacroSet    string
		Macros      []string
	}{
		Version:     fmt.Sprintf("%d.%d.%d", Major, Minor, Patch),
		MultiClient: g.MultiClient,
		MacroSet:    currentMacros.name,
	}
	for i := GamcroAPI(1); i < GamcroAPI_end; i <<= 1 {
		if g.APIs.Active(i) {
			cfg.APIs = append(cfg.APIs, i.String())
		}
	}
	for _, m := range currentMacros.macros {
		cfg.Macros = append(cfg.Macros, m.name)
	}
	wr.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(wr)
	enc.Encode(&cfg)
}
