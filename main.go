package main

import (
	"bufio"
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
	gamcro = internal.Gamcro{
		APIs: internal.TypeAPI |
			internal.ClipPostAPI |
			internal.ClipGetAPI |
			internal.SaveTexts,
	}

	paths = ospath.NewApp(ospath.ExeDir(), internal.AppName)
	//go:embed banner.txt
	banner []byte
	log    = qbsllm.New(qbsllm.Lnormal, internal.AppName, nil, nil)
	logCfg = c4hgol.Config(qbsllm.NewConfig(log), internal.LogCfg)
)

func showBanner() {
	os.Stdout.Write(banner)
	fmt.Printf("v%d.%d.%d [%s #%d; %s]\n",
		internal.Major, internal.Minor, internal.Patch,
		internal.Prerelease, internal.BuildNo,
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
	fApis := flag.String("apis", gamcro.APIs.FlagString(), docAPIsFlag())
	flag.StringVar(&gamcro.CORS, "cors", "", docCORSFlag)
	noPass := flag.Bool("no-passphrase", false, docNoPassFlag)
	fQR := flag.Bool("qr", false, docQRFlag)
	fLog := flag.String("log", "", c4hgol.LevelCfgDoc(nil))
	flag.Parse()
	c4hgol.SetLevel(logCfg, *fLog, nil)
	if !*noPass {
		gamcro.Passphr = readPassphrase(false)
	}
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
	gamcro.APIs = internal.ParseRoboAPISet(*fApis)
	gamcro.TextsDir = paths.LocalDataPath(internal.DefaultTextsDir)
	log.Infof("Authenticate to realm \"Gamcro: %s\"", internal.CurrentRealmKey)
	log.Fatale(gamcro.Run())
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

func readPassword(prompt, prompt2 string, allowEmpty bool) (res []byte) {
	for {
		fmt.Print(prompt)
		var err error
		res, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatale(err)
		}
		fmt.Println()
		if (allowEmpty && len(res) == 0) || prompt2 == "" {
			break
		}
		fmt.Print(prompt2)
		res2, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatale(err)
		}
		fmt.Println()
		if reflect.DeepEqual(res, res2) {
			break
		}
		fmt.Println("Mismatch detected, please try again")
	}
	return res
}

func readPassphrase(twice bool) []byte {
	var prompt2 string
	if twice {
		prompt2 = "Repeat passphrase: "
	}
	return readPassword(
		"Enter passphrase (empty skips security): ",
		prompt2,
		true,
	)
}

func userInputCreds(paths ospath.AppPaths, cauth *internal.AuthCreds) (err error) {
	log.Infos("Need user and password for HTTP basic auth")
	var usr string
	fmt.Print("Enter HTTP basic auth user: ")
	if _, err = fmt.Scan(&usr); err != nil {
		return err
	}
	pass1 := readPassword(
		"Enter HTTP basic auth password: ",
		"Repeat password: ",
		false,
	)
	cauth.Set(usr, string(pass1))
	authFile := paths.LocalData(internal.DefaultCredsFile)
	fmt.Printf("Save user:password to '%s' (y/N)? ", authFile)
	var answer string
	scn := bufio.NewScanner(os.Stdin)
	if scn.Scan() {
		answer = scn.Text()
	}
	if l := strings.ToLower(answer); l == "y" || l == "yes" {
		if _, err := ospath.ProvideDir(internal.NewDirPerm, authFile); err != nil {
			return err
		}
		err = cauth.WriteFile(authFile)
		return err
	}
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

	docNoPassFlag = `Skip the question to enter a passphrase. Note that this bypasses
the security for the generated X.509 priavte key. Using this option
may be useful to avoid interactive input. But it is not recommended
for normal use. Having a browser that accepts a certificate that can
be used by any dubious program on your machine is a serious risk.`

	docCORSFlag = `When not empty the value will be used for the HTTP 
Access-Control-Allow-Origin header in HTTP responses. Also the
Access-Control-Allow-Credentials header then is set true.`
)

func docAPIsFlag() string {
	var sb strings.Builder
	sb.WriteString("Selet the API operations that are offered by Gamcro. The APIs are:")
	for i := internal.GamcroAPI(1); i < internal.GamcroAPI_end; i <<= 1 {
		fmt.Fprintf(&sb, "\n - %s", i.String())
	}
	fmt.Fprintln(&sb)
	return sb.String()
}
