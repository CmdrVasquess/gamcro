package internal

import (
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	"github.com/gorilla/mux"
)

func (g *Gamcro) apiRoutes(r *mux.Router) {
	r.HandleFunc("/keyboard/type", g.auth(g.handleKeyboardType)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
	r.HandleFunc("/clip", g.auth(g.handleClipStr)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
}

func (g *Gamcro) rqBodyRd(wr http.ResponseWriter, rq *http.Request) io.ReadCloser {
	var rd io.ReadCloser = rq.Body
	if g.TxtLimit > 0 {
		rd = http.MaxBytesReader(wr, rq.Body, int64(g.TxtLimit))
	}
	return rd
}

func (g *Gamcro) rqBody(wr http.ResponseWriter, rq *http.Request) ([]byte, error) {
	rd := g.rqBodyRd(wr, rq)
	defer rd.Close()
	return io.ReadAll(rd)
}

// TODO implicitly convert []byte to filtered str: avoid one copy?
func filterStr(s string, f func(rune) bool) string {
	var sb strings.Builder
	for _, r := range s {
		if f(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func cleanText(s string) string {
	return filterStr(s, unicode.IsGraphic)
}

func (g *Gamcro) handleKeyboardType(wr http.ResponseWriter, rq *http.Request) {
	body, err := g.rqBody(wr, rq)
	if err != nil {
		log.Errora("Read body failed with `err`", err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		return
	}
	txt := cleanText(string(body))
	log.Infoa("keyboard/type `text`", txt)
	robotgo.TypeStr(txt)
}

func (g *Gamcro) handleClipStr(wr http.ResponseWriter, rq *http.Request) {
	body, err := g.rqBody(wr, rq)
	if err == nil && len(body) > 0 {
		s := cleanText(string(body))
		log.Infoa("clip `text`", s)
		if err = clipboard.WriteAll(s); err != nil {
			log.Errore(err)
		}
	} else if err != nil {
		log.Errora("Read body failed with `err`", err)
	}
}
