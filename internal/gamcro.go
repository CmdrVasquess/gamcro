package internal

import (
	"bytes"
	"crypto/tls"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"

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
	var keyPEMBlock []byte
	err = cryptReadFile(g.TLSKey, g.Passphr, func(rd io.Reader) error {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, rd); err != nil {
			return err
		}
		keyPEMBlock = buf.Bytes()
		return nil
	})
	if err != nil {
		return err
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
