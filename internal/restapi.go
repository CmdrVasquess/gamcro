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

type GamcroAPI uint32

//go:generate stringer -type GamcroAPI
const (
	TypeAPI GamcroAPI = (1 << iota)
	TapAPI
	ClipPostAPI
	ClipGetAPI

	GamcroAPI_end
)

func (r GamcroAPI) Active(a GamcroAPI) bool {
	return (r & a) == a
}

func (set GamcroAPI) FlagString() string {
	var sb strings.Builder
	for i := GamcroAPI(1); i < GamcroAPI_end; i <<= 1 {
		if set.Active(i) {
			if sb.Len() > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(i.String())
		}
	}
	return sb.String()
}

func ParseRoboAPISet(flag string) (set GamcroAPI) {
	apis := strings.Split(flag, ",")
	for i := GamcroAPI(1); i < GamcroAPI_end; i <<= 1 {
		inm := i.String()
		for _, api := range apis {
			if api == inm {
				set |= i
			}
		}
	}
	return set
}

func (g *Gamcro) mayRobo(api GamcroAPI, wr http.ResponseWriter) bool {
	res := g.APIs.Active(api)
	if !res {
		log.Warna("blocked `robo api`", api.String())
		wr.WriteHeader(http.StatusForbidden)
	}
	return res
}

func (g *Gamcro) apiRoutes(r *mux.Router) {
	r.HandleFunc("/keyboard/type", g.auth(g.handleKeyboardType)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
	r.HandleFunc("/keyboard/tap/{key}", g.auth(g.handleKeyboardTap)).
		Methods(http.MethodPost)
	r.HandleFunc("/clip", g.auth(g.handleClipPost)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
	r.HandleFunc("/clip", g.auth(g.handleClipGet)).
		Methods(http.MethodGet)
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
	if !g.mayRobo(TypeAPI, wr) {
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
	if !g.mayRobo(TapAPI, wr) {
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

func (g *Gamcro) handleClipPost(wr http.ResponseWriter, rq *http.Request) {
	if !g.mayRobo(ClipPostAPI, wr) {
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
		log.Infoa("clip `text` to board", txt)
		if err = clipboard.WriteAll(txt); err != nil {
			log.Errore(err)
			http.Error(wr, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	wr.WriteHeader(http.StatusNoContent)
}

func (g *Gamcro) handleClipGet(wr http.ResponseWriter, rq *http.Request) {
	if !g.mayRobo(ClipGetAPI, wr) {
		return
	}
	txt, err := clipboard.ReadAll()
	if err != nil {
		log.Errore(err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	log.Infoa("clip `text` from board", txt)
	wr.Header().Set("Content-Type", "text/plain")
	io.WriteString(wr, txt)
}
