package internal

import (
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-vgo/robotgo"
	"github.com/gorilla/mux"
)

type RoboAPI uint32

const (
	RoboType RoboAPI = (1 << iota)
	RoboTap
	RoboClip
)

func (r RoboAPI) Active(a RoboAPI) bool {
	return (r & a) == a
}

func (g *Gamcro) apiRoutes(r *mux.Router) {
	r.HandleFunc("/keyboard/type", g.auth(g.handleKeyboardType)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
	r.HandleFunc("/keyboard/tap/{key}", g.auth(g.handleKeyboardTap)).
		Methods(http.MethodPost)
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

func validQuery(rq *http.Request, r validation.MapRule) (map[string][]string, error) {
	qry := rq.URL.Query()
	err := validKbdTapQuery.Validate(qry)
	return qry, err
}

func (g *Gamcro) handleKeyboardType(wr http.ResponseWriter, rq *http.Request) {
	if !g.RoboAPIs.Active(RoboType) {
		wr.WriteHeader(http.StatusForbidden)
		return
	}
	body, err := g.rqBody(wr, rq)
	if err != nil {
		log.Errora("Read body failed with `err`", err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(body) > 0 {
		txt := cleanText(string(body))
		log.Infoa("keyboard/type `text`", txt)
		robotgo.TypeStr(txt)
	}
	wr.WriteHeader(http.StatusNoContent)
}

var validKbdTapQuery = validation.Map(
	validation.Key("arg", validation.Required).Optional(),
)

func (g *Gamcro) handleKeyboardTap(wr http.ResponseWriter, rq *http.Request) {
	if !g.RoboAPIs.Active(RoboTap) {
		wr.WriteHeader(http.StatusForbidden)
		return
	}
	key := mux.Vars(rq)["key"]
	qry, err := validQuery(rq, validKbdTapQuery)
	if err != nil {
		log.Errore(err)
		wr.WriteHeader(http.StatusBadRequest)
		return
	}
	args := qry["arg"]
	log.Infoa("keyboard/tap `key` with `args`", key, args)
	var tapargs = make([]interface{}, len(args))
	for i, a := range args {
		tapargs[i] = a
	}
	if res := robotgo.KeyTap(key, tapargs...); res != "" {
		log.Errora("keyboard/tap `error`", res)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	wr.WriteHeader(http.StatusNoContent)
}

func (g *Gamcro) handleClipStr(wr http.ResponseWriter, rq *http.Request) {
	if !g.RoboAPIs.Active(RoboClip) {
		wr.WriteHeader(http.StatusForbidden)
		return
	}
	body, err := g.rqBody(wr, rq)
	if err != nil {
		log.Errora("Read body failed with `err`", err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(body) > 0 {
		txt := cleanText(string(body))
		log.Infoa("clip `text`", txt)
		if err = clipboard.WriteAll(txt); err != nil {
			log.Errore(err)
			http.Error(wr, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	wr.WriteHeader(http.StatusNoContent)
}
