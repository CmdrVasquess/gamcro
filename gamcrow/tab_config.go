package main

import (
	"errors"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/CmdrVasquess/gamcro/internal"
)

type ConfigTab struct {
	container.TabItem
	user   string
	passwd string

	userEntry *widget.Entry
	passEntry *widget.Entry
}

func NewConfigTab(prefs fyne.Preferences, gamcro *internal.Gamcro) *ConfigTab {
	res := &ConfigTab{}

	res.userEntry = widget.NewEntry()
	res.userEntry.Validator = func(s string) error {
		if authFile != "" || s != "" {
			return nil
		}
		return errors.New("You must provide a username")
	}

	res.passEntry = widget.NewPasswordEntry()
	res.passEntry.Validator = func(s string) error {
		if res.userEntry.Text != "" && s == "" {
			return errors.New("Password must not be empty")
		}
		return nil
	}
	res.passEntry.OnChanged = func(s string) {
		res.passwd = s
		connectTab.setGuide(guide())
	}
	res.passEntry.Disable()

	res.userEntry.OnChanged = func(user string) {
		res.user = user
		if user == "" {
			res.passEntry.Disable()
		} else {
			res.passEntry.Enable()
		}
		res.passEntry.Validate()
		connectTab.setGuide(guide())
	}

	txtLimEntry := widget.NewEntry()
	txtLimEntry.Validator = func(s string) error {
		i, err := strconv.Atoi(s)
		if err != nil || i < 1 {
			return errors.New("Text limit must be an integer greater than zero.")
		}
		return nil
	}
	txtLimEntry.OnChanged = func(s string) {
		i, _ := strconv.Atoi(s)
		prefs.SetInt(prefTextLim, i)
	}
	txtLimEntry.SetText(strconv.Itoa(prefs.Int(prefTextLim)))

	clientsSelect := widget.NewSelect(
		[]string{"local", "all"}, nil,
	)
	clientsSelect.SetSelected(prefs.String(prefClients))
	clientsSelect.OnChanged = func(s string) { prefs.SetString(prefClients, s) }

	form := widget.NewForm(
		widget.NewFormItem("Web User", res.userEntry),
		widget.NewFormItem("Web Password", res.passEntry),
		widget.NewFormItem("Server Address", widget.NewEntryWithData(
			binding.BindPreferenceString(prefSrvAddr, prefs),
		)),
		widget.NewFormItem("Text Limit", txtLimEntry),
		widget.NewFormItem("Clients", clientsSelect),
	)

	schostCheck := widget.NewCheck("Single Client Hots", func(chk bool) {
		prefs.SetBool(prefSCHost, chk)
	})
	schostCheck.SetChecked(prefs.BoolWithFallback(prefSCHost, true))

	res.TabItem = *container.NewTabItem("Config",
		container.NewVBox(
			form,
			schostCheck,
		),
	)
	return res
}
