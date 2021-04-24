package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"runtime"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/pack/ospath"
	"github.com/CmdrVasquess/gamcro/internal"
)

var (
	gamcro internal.Gamcro

	paths = ospath.NewApp(ospath.ExeDir(), internal.AppName)
	//go:embed banner.txt
	banner []byte
)

func showBanner() {
	os.Stdout.Write(banner)
	fmt.Printf("v%d.%d.%d [%s #%d; %s]\n",
		internal.Major, internal.Minor, internal.Patch,
		internal.Quality, internal.BuildNo,
		runtime.Version(),
	)
}

func main() {
	showBanner()
	flag.StringVar(&gamcro.SrvAddr, "addr", ":9420", docSrvAddrFlag)
	flag.StringVar(&gamcro.TLSCert, "cert", paths.LocalData("cert.pem"), docTlsCertFlag)
	flag.StringVar(&gamcro.TLSKey, "key", paths.LocalData("key.pem"), docTlsKeyFlag)
	authFlag := flag.String("auth", "", fmt.Sprintf(docAuthCredsFlag, internal.DefaultCredsFile))
	flag.IntVar(&gamcro.TxtLimit, "text-limit", 256, docTxtLimitFlag)
	flag.BoolVar(&gamcro.MultiClient, "multi-client", false, docMCltFlag)
	flag.StringVar(&gamcro.ClientNet, "clients", "local", docClientsFlag)
	fQR := flag.Bool("qr", false, docQRFlag)
	fLog := flag.String("log", "", c4hgol.LevelCfgDoc(nil))
	flag.Parse()
	c4hgol.SetLevel(internal.LogCfg, *fLog, nil)
	gamcro.Run(paths, *authFlag, *fQR)
}

const (
	docSrvAddrFlag = `This is the local address the API server and the UI server
listen to. Most people should be fine with the default. If
you just need to set a different port number <port> set
the flag value ":<port>". For more details read the description
of the address parameter in https://golang.org/pkg/net/#Listen
`

	docTlsCertFlag = `TLS certificate file to use for HTTPS.
If neither the certificate nor the key file exist, Gamcro will
generate them with an self-signed X.509 certificate.
`

	docTlsKeyFlag = `TLS key file to use for HTTPS.
If neither the certificate nor the key file exist, Gamcro will
generate them with an self-signed X.509 certificate.
`

	docAuthCredsFlag = `Access to the API server is protected by HTTP basic auth.
A single <user>:<password> pair will be used to check a user's
authorization. Use this flags to set the <user>:<password>
credentials. The current settings are determined like this:
 - When the 'auth' falg is empty, Gamcro checks for the file
   '%[1]s' in the same folder as the Gamcro executable.
   When present Gamcro reads <user>:<password> from the first
   text line of the file. Keep read access to that file as
   restrictive as possible.
 - When 'auth' flag is set to ":" Gamcro ignores the '%[1]s'
   file in the executables directory and reads <user> and
   <password> from the terminal.
 - Otherwise when 'auth' flag's value contains ':' Gamcro
   considers 'auth' flag to be <user>:<password> and uses is.
 - Else 'auth' flag is considered to be a filename and Gamcro
   will try to read <user>:<password> from the first text line
   of that file.`

	docTxtLimitFlag = `Limit the length of text input to API.`

	docMCltFlag = `Allow more than one client machine to send macros.`

	docClientsFlag = `Which API clients are allowed. If clients is not 'all' only
clients from the local network will be accepted.`

	docQRFlag = `Show connect URL as QR code`
)
