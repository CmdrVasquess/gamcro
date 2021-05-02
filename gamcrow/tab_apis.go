package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/CmdrVasquess/gamcro/internal"
)

type APIsTab struct {
	container.TabItem
	apis internal.GamcroAPI
}

func NewAPIsTab(prefs fyne.Preferences) *APIsTab {
	res := &APIsTab{
		apis: internal.ParseRoboAPISet(prefs.String(prefAPIs)),
	}
	var ls []fyne.CanvasObject
	for i := internal.GamcroAPI(1); i < internal.GamcroAPI_end; i <<= 1 {
		bit := i
		chk := widget.NewCheck(i.String(), func(f bool) {
			if f {
				res.apis |= bit
			} else {
				res.apis &= ^bit
			}
			prefs.SetString(prefAPIs, res.apis.FlagString())
		})
		chk.SetChecked(res.apis&i != 0)
		ls = append(ls, chk)
	}
	res.TabItem = *container.NewTabItem("APIs", container.NewVBox(ls...))
	return res
}
