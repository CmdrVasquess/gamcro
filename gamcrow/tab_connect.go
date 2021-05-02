package main

import (
	"net/url"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"
)

type ConnectTab struct {
	container.TabItem
}

func NewConnectTab() *ConnectTab {
	const urlstr = "https://192.168.0.2:9420/"
	url, _ := url.Parse(urlstr)
	qr, _ := qrcode.New(urlstr, qrcode.Medium)
	img := canvas.NewImageFromImage(qr.Image(320))
	img.FillMode = canvas.ImageFillOriginal
	res := &ConnectTab{
		TabItem: *container.NewTabItem("Connect Hint",
			container.NewVBox(
				widget.NewHyperlink(urlstr, url),
				img,
			),
		),
	}
	return res
}
