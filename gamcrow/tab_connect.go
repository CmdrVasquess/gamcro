package main

import (
	"bytes"
	_ "embed"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"
)

//go:embed gamcrow.png
var logoPNG []byte

type ConnectTab struct {
	container.TabItem
	box   *fyne.Container
	top   fyne.CanvasObject
	img   *canvas.Image
	guide *widget.Label
}

func NewConnectTab() *ConnectTab {
	lb := widget.NewLabel(guide())
	img := canvas.NewImageFromReader(bytes.NewReader(logoPNG), "Logo")
	img.Resize(fyne.Size{320, 320})
	img.FillMode = canvas.ImageFillOriginal
	vbox := container.NewVBox(lb, img)
	res := &ConnectTab{
		TabItem: *container.NewTabItem("Connect Hint", vbox),
		box:     vbox,
		top:     lb,
		img:     img,
		guide:   lb,
	}
	return res
}

func (ctab *ConnectTab) setGuide(txt string) {
	ctab.guide.SetText(txt)
}

func (ctab *ConnectTab) setHint(urlstr string) {
	qr, _ := qrcode.New(urlstr, qrcode.Medium)
	img := canvas.NewImageFromImage(qr.Image(320))
	img.FillMode = canvas.ImageFillOriginal
	url, _ := url.Parse(urlstr)
	ctab.box.Remove(ctab.top)
	ctab.box.Remove(ctab.img)
	ctab.box.Add(widget.NewHyperlink(urlstr, url))
	ctab.box.Add(img)
	ctab.img = img
}
