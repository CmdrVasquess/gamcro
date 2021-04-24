package internal

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"git.fractalqb.de/fractalqb/pack/ospath"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
)

var (
	log    = qbsllm.New(qbsllm.Lnormal, AppName, nil, nil)
	LogCfg = qbsllm.NewConfig(log)

	//go:embed webui
	webfs embed.FS
)

const (
	AppName    = "gamcro"
	newDirPerm = 0750
)

type Gamcro struct {
	SrvAddr         string
	Passphr         []byte
	TLSCert, TLSKey string
	clientAuth      authCreds
	singleClient    string
	MultiClient     bool
	ClientNet       string
	TxtLimit        int
}

func (g *Gamcro) Run(paths ospath.AppPaths, authFlag string, qrFlag bool) {
	if g.TxtLimit <= 0 {
		g.TxtLimit = 256
	}
	webRoutes := mux.NewRouter()
	webRoutes.HandleFunc("/", handleUI)
	if staticDir, err := fs.Sub(webfs, "webui"); err != nil {
		log.Fatale(err)
	} else {
		staticHdlr := http.FileServer(http.FS(staticDir))
		webRoutes.PathPrefix("/s/").Handler(g.auth(
			http.StripPrefix("/s/", staticHdlr).ServeHTTP,
		))
	}
	g.apiRoutes(webRoutes)

	// TODO elaborate encrypted storage
	//var err error
	// fmt.Print("Enter passphrase for file encryption (empty disables encryption): ")
	// passphr, err = term.ReadPassword(int(os.Stdin.Fd()))
	// fmt.Println()
	// if err != nil {
	// 	log.Fatale(err)
	// } else if len(passphr) == 0 {
	// 	log.Warns("Empty passphrase. File encryption diabled.")
	// } else {
	// 	log.Infos("File encryption enabled")
	// }

	if err := g.ensureCreds(authFlag, paths); err != nil {
		log.Fatale(err)
	}
	if err := ensureTLSCert(g.TLSCert, g.TLSKey); err != nil {
		log.Fatale(err)
	}
	log.Infoa("Load TLS `certificate`", g.TLSCert)
	log.Infoa("Load TLS `key`", g.TLSKey)
	log.Infof("Runninig gamcro HTTPS server on %s", g.SrvAddr)
	g.connectHint(qrFlag)
	log.Infof("Authenticate to realm \"Gamcro: %s\"", currentRealmKey)
	log.Fatale(http.ListenAndServeTLS(g.SrvAddr, g.TLSCert, g.TLSKey, webRoutes))
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

func (g *Gamcro) connectHint(qr bool) {
	if g.SrvAddr == "" {
		return
	}
	var svcurl string
	if g.SrvAddr[0] != ':' {
		svcurl = fmt.Sprintf("https://%s/", g.SrvAddr)
	} else {
		conn, _ := net.Dial("udp", "8.8.8.8:80")
		defer conn.Close()
		addr := conn.LocalAddr().(*net.UDPAddr)
		svcurl = fmt.Sprintf("https://%s%s/", addr.IP, g.SrvAddr)
	}
	log.Infof("Use %s to connect your browser to the Web UI", svcurl)
	if qr {
		qr, err := qrcode.New(svcurl, qrcode.Low)
		if err != nil {
			log.Errore(err)
		} else {
			art := qr.ToString(false)
			fmt.Print(art)
		}
	}
}
