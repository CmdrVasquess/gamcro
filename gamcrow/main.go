package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"git.fractalqb.de/fractalqb/pack/ospath"
	"github.com/CmdrVasquess/gamcro/internal"
)

var (
	paths  = ospath.NewApp(ospath.ExeDir(), internal.AppName)
	gamcro = internal.Gamcro{
		APIs: internal.TypeAPI | internal.ClipPostAPI,
	}
	defaultAuthFile = paths.LocalData(internal.DefaultCredsFile)

	gapp       fyne.App
	wapp       fyne.Window
	configTab  *ConfigTab
	apisTab    *APIsTab
	connectTab *ConnectTab
	mainBox    *fyne.Container
	startBtn   *widget.Button
	authFile   string
)

func hasAuthFile() string {
	_, err := os.Stat(defaultAuthFile)
	if os.IsNotExist(err) {
		log.Println("no auth file", defaultAuthFile)
		return ""
	}
	return defaultAuthFile
}

func guide() string {
	cfgValid := configTab.userEntry.Validate() == nil
	cfgValid = cfgValid && configTab.passEntry.Validate() == nil
	if authFile == "" && !cfgValid {
		return "Enter Web user and password on Config tab"
	}
	if len(gamcro.Passphr) == 0 {
		return "Enter Passphrase"
	}
	return "Press Start"
}

const (
	prefSrvAddr = "srv-addr"
	prefTextLim = "text-limit"
	prefClients = "clients"
	prefSCHost  = "single-client-host"
	prefAPIs    = "apis"
)

func initPrefs(prefs fyne.Preferences) {
	if prefs.String(prefSrvAddr) == "" {
		prefs.SetString(prefSrvAddr, ":9420")
	}
	if prefs.Int(prefTextLim) < 1 {
		prefs.SetInt(prefTextLim, 250)
	}
	if prefs.String(prefClients) == "" {
		prefs.SetString(prefClients, "local")
	}
	if prefs.String(prefAPIs) == "" {
		apis := internal.TypeAPI | internal.ClipPostAPI
		prefs.SetString(prefAPIs, apis.FlagString())
	}
}

func main() {
	gapp = app.NewWithID("de.fractalqb.jv.gamcro")
	prefs := gapp.Preferences()
	initPrefs(prefs)
	authFile = hasAuthFile()

	configTab = NewConfigTab(prefs, &gamcro)
	apisTab = NewAPIsTab(prefs)
	connectTab = NewConnectTab()

	tabs := container.NewAppTabs(
		&connectTab.TabItem,
		&configTab.TabItem,
		&apisTab.TabItem,
	)

	startBtn = widget.NewButton("Start", nil)
	startBtn.Disable()
	startBtn.OnTapped = startGamcro

	passEntry := widget.NewPasswordEntry()
	passEntry.Validator = func(s string) error {
		if s == "" {
			startBtn.Disable()
			return errors.New("Passphrase must not be empty")
		}
		startBtn.Enable()
		return nil
	}
	passEntry.OnChanged = func(s string) {
		gamcro.Passphr = []byte(s)
		connectTab.setGuide(guide())
	}
	passFrom := widget.NewForm(widget.NewFormItem("Passphrase", passEntry))

	mainBox = container.NewVBox(
		tabs,
		passFrom,
		startBtn,
	)

	wapp = gapp.NewWindow(fmt.Sprintf("GamcroW v%d.%d.%d",
		internal.Major,
		internal.Minor,
		internal.Patch,
	))
	wapp.SetContent(mainBox)
	wapp.ShowAndRun()
}

func startGamcro() {
	prefs := gapp.Preferences()
	startBtn.Disable()
	gamcro.SrvAddr = prefs.String(prefSrvAddr)
	if configTab.user != "" && configTab.passwd != "" {
		log.Println(defaultAuthFile)
		err := gamcro.ClientAuth.Set(configTab.user, configTab.passwd)
		if err == nil {
			gamcro.ClientAuth.WriteFile(defaultAuthFile)
		} else {
			log.Println(err)
		}
	} else {
		err := gamcro.ClientAuth.ReadFile(defaultAuthFile)
		if err != nil { // TODO what to do on error
			log.Println(err)
		}
	}
	gamcro.MultiClient = !prefs.Bool(prefSCHost)
	gamcro.ClientNet = prefs.String(prefClients)
	gamcro.TxtLimit = prefs.Int(prefTextLim)
	gamcro.TLSCert = paths.LocalData("cert.pem")
	gamcro.TLSKey = paths.LocalData("key.pem")
	gamcro.APIs = apisTab.apis
	gamcro.TextsDir = paths.LocalDataPath(internal.DefaultTextsDir)

	connectTab.setHint(gamcro.ConnectHint())
	mainBox.Remove(startBtn)
	mainBox.Add(widget.NewLabel(fmt.Sprintf(
		"Browser Login to Gamcro: %s",
		internal.CurrentRealmKey,
	)))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(&gamcro)
	go func() {
		err := gamcro.Run()
		log.Println(err)
		info := dialog.NewInformation("Error running Gamcro", err.Error(), wapp)
		info.SetOnClosed(func() {
			gapp.Quit()
		})
		wapp.Content().Refresh()
		info.Show()
	}()
}
