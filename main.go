package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"git.fractalqb.de/fractalqb/c4hgol"
	"git.fractalqb.de/fractalqb/pack/ospath"
	"git.fractalqb.de/fractalqb/qbsllm"
	"github.com/CmdrVasquess/gamcro/internal"
	"github.com/skip2/go-qrcode"
	"golang.org/x/term"
)

var (
	gamcro internal.Gamcro

	paths = ospath.NewApp(ospath.ExeDir(), internal.AppName)
	//go:embed banner.txt
	banner []byte
	log    = qbsllm.New(qbsllm.Lnormal, "tgamcro", nil, nil)
	logCfg = c4hgol.Config(qbsllm.NewConfig(log), internal.LogCfg)
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
	c4hgol.SetLevel(logCfg, *fLog, nil)
	gamcro.Passphr = readPassphrase()
	if err := ensureCreds(*authFlag, &gamcro.ClientAuth); err != nil {
		log.Fatale(err)
	}
	if hint := gamcro.ConnectHint(); hint != "" {
		log.Infof("Use %s to connect your browser to the Web UI", hint)
		if *fQR {
			qr, err := qrcode.New(hint, qrcode.Low)
			if err != nil {
				log.Errore(err)
			} else {
				art := qr.ToString(false)
				fmt.Print(art)
			}
		}
	}
	log.Infof("Authenticate to realm \"Gamcro: %s\"", internal.CurrentRealmKey)
	log.Fatale(gamcro.Run(*fQR))
}

func ensureCreds(flag string, cauth *internal.AuthCreds) (err error) {
	colonIdx := strings.IndexByte(flag, ':')
	switch {
	case flag == "":
		authFile := paths.LocalData(internal.DefaultCredsFile)
		if _, err = os.Stat(authFile); err == nil {
			log.Infoa("HTTP basic auth configuration `file` detected", authFile)
			log.Infoa("You can use -auth flag for different settings", authFile)
			err := cauth.ReadFile(authFile)
			if err == nil {
				return nil
			}
			log.Warne(err)
		}
		err = userInputCreds(paths, cauth)
	case flag == ":":
		err = userInputCreds(paths, cauth)
	case colonIdx >= 0:
		if colonIdx < len(flag)-1 {
			me := filepath.Base(os.Args[0])
			log.Warns("It is not secure to set passwords on the command line!")
			log.Infof("Better use '%s -auth <filename>' with restricted access to <filename>", me)
		}
	default:
		err = cauth.ReadFile(flag)
	}
	return err
	// if strings.HasSuffix(flag, ":") {
	// 	log.Warns("Cannot accept empty password")
	// 	passwd := makeRandStr(passwdChars, 7)
	// 	log.Infof("Using one-time password \"%s\"", passwd)
	// 	cfg.clientAuth.set(flag[:len(flag)-1], passwd)
	// }
}

func readPassphrase() []byte {
	fmt.Print("Enter passphrase (empty skips security): ")
	res, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatale(err)
	}
	fmt.Println()
	return res
}

func userInputCreds(paths ospath.AppPaths, cauth *internal.AuthCreds) (err error) {
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
	cauth.Set(usr, string(pass1))
	authFile := paths.LocalData(internal.DefaultCredsFile)
	fmt.Printf("\nSave user:password to '%s' (y/N)?", authFile)
	var answer string
	if _, err = fmt.Scan(&answer); err != nil {
		return err
	}
	if l := strings.ToLower(answer); l == "y" || l == "yes" {
		if _, err := ospath.ProvideDir(internal.NewDirPerm, authFile); err != nil {
			return err
		}
		err = cauth.WriteFile(authFile)
		return err
	}
	fmt.Println()
	return nil
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
