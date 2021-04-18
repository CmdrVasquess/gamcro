package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"runtime"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/pack/ospath"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/gorilla/mux"
)

//go:generate versioner -bno build_no VERSION version.go

const (
	appName    = "gamcro"
	newDirPerm = 0750
)

type Config struct {
	srvAddr         string
	passphr         []byte
	tlsCert, tlsKey string
	authCreds       string
	singleClient    string
	multiClient     bool
	clientNet       string
	txtLimit        int
}

var (
	cfg   = Config{txtLimit: 256}
	fLog  string
	paths = ospath.NewApp(ospath.ExeDir(), appName)

	log    = qbsllm.New(qbsllm.Lnormal, appName, nil, nil)
	logCfg = qbsllm.NewConfig(log)

	//go:embed banner.txt
	banner []byte
	//go:embed web-ui/dist
	webfs embed.FS
)

func showBanner() {
	os.Stdout.Write(banner)
	fmt.Printf("v%d.%d.%d [%s #%d; %s]\n",
		Major, Minor, Patch,
		Quality, BuildNo,
		runtime.Version(),
	)
}

func main() {
	showBanner()
	flag.StringVar(&cfg.srvAddr, "addr", ":9420", docSrvAddrFlag)
	flag.StringVar(&cfg.tlsCert, "cert", paths.LocalData("cert.pem"), docTlsCertFlag)
	flag.StringVar(&cfg.tlsKey, "key", paths.LocalData("key.pem"), docTlsKeyFlag)
	flag.StringVar(&cfg.authCreds, "auth", "",
		fmt.Sprintf(docAuthCredsFlag, defaultCredsFile))
	flag.IntVar(&cfg.txtLimit, "text-limit", cfg.txtLimit, docTxtLimitFlag)
	flag.BoolVar(&cfg.multiClient, "multi-client", false, docMCltFlag)
	flag.StringVar(&cfg.clientNet, "clients", "local", docClientsFlag)
	flag.StringVar(&fLog, "log", "", c4hgol.LevelCfgDoc(nil))
	flag.Parse()
	c4hgol.SetLevel(logCfg, fLog, nil)

	webRoutes := mux.NewRouter()
	webRoutes.HandleFunc("/", handleUI)
	if staticDir, err := fs.Sub(webfs, "web-ui/dist"); err != nil {
		log.Fatale(err)
	} else {
		staticHdlr := http.FileServer(http.FS(staticDir))
		webRoutes.PathPrefix("/s/").Handler(http.StripPrefix("/s/", staticHdlr))
	}
	apiRoutes(webRoutes)

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

	if err := ensureCreds(); err != nil {
		log.Fatale(err)
	}
	if err := ensureTLSCert(cfg.tlsCert, cfg.tlsKey); err != nil {
		log.Fatale(err)
	}
	log.Infoa("Load TLS `certificate`", cfg.tlsCert)
	log.Infoa("Load TLS `key`", cfg.tlsKey)
	log.Infof("Runninig gamcro HTTPS server on %s", cfg.srvAddr)
	connectHint()
	log.Fatale(http.ListenAndServeTLS(cfg.srvAddr, cfg.tlsCert, cfg.tlsKey, webRoutes))
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

func connectHint() {
	if cfg.srvAddr == "" {
		return
	}
	if cfg.srvAddr[0] != ':' {
		log.Infof("Use https://%s/ to connect your browser to the Web UI", cfg.srvAddr)
		return
	}
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	log.Infof("Use https://%s%s/ to connect your browser to the Web UI",
		addr.IP,
		cfg.srvAddr,
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
