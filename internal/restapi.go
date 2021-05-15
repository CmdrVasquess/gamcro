package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	SaveTexts

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
	r.HandleFunc("/texts", g.auth(g.listTexts)).
		Methods(http.MethodGet)
	r.HandleFunc("/texts/{set}", g.auth(g.loadText)).
		Methods(http.MethodGet)
	r.HandleFunc("/texts/{set}", g.auth(g.saveText)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "application/json")
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

func httpError(wr http.ResponseWriter, err error, sllm string, args ...interface{}) bool {
	if err != nil {
		log.Errora(sllm+": `error`", append(args, err))
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return true
	}
	return false
}

func (g *Gamcro) handleKeyboardType(wr http.ResponseWriter, rq *http.Request) {
	if !g.mayRobo(TypeAPI, wr) {
		return
	}
	body, err := g.rqBody(wr, rq)
	if httpError(wr, err, "read body") {
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
	if httpError(wr, err, "read body") {
		return
	}
	if len(body) > 0 {
		txt := cleanText(string(body))
		log.Infoa("clip `text` to board", txt)
		err = clipboard.WriteAll(txt)
		if httpError(wr, err, "clip write") {
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
	if httpError(wr, err, "clip read") {
		return
	}
	log.Infoa("clip `text` from board", txt)
	wr.Header().Set("Content-Type", "text/plain")
	io.WriteString(wr, txt)
}

func (g *Gamcro) listTexts(wr http.ResponseWriter, rq *http.Request) {
	log.Debugs("list texts")
	dir, err := os.Open(g.TextsDir)
	if httpError(wr, err, "read `dir`", g.TextsDir) {
		return
	}
	defer dir.Close()
	entries, err := dir.ReadDir(0)
	if err != nil {
		log.Errore(err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	ls := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if filepath.Ext(n) == ".json" {
			n = n[:len(n)-5]
			ls = append(ls, n)
		}
	}
	enc := json.NewEncoder(wr)
	wr.Header().Set("Content-Type", "application/json")
	enc.Encode(ls)
}

func (g *Gamcro) loadText(wr http.ResponseWriter, rq *http.Request) {
	setName := mux.Vars(rq)["set"]
	log.Debuga("load `text`", setName)
	if dir, _ := filepath.Split(setName); dir != "" || len(setName) > 64 {
		log.Errora("tried to save texts to `path`", setName)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	file := filepath.Join(g.TextsDir, setName+".json")
	rd, err := os.Open(file)
	if httpError(wr, err, "open `file`", file) {
		return
	}
	defer rd.Close()
	wr.Header().Set("Content-Type", "application/json")
	io.Copy(wr, rd)
}

func (g *Gamcro) saveText(wr http.ResponseWriter, rq *http.Request) {
	setName := mux.Vars(rq)["set"]
	if dir, _ := filepath.Split(setName); dir != "" || len(setName) > 64 {
		log.Errora("tried to save texts to `path`", setName)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := os.Stat(g.TextsDir); os.IsNotExist(err) {
		log.Infoa("create `texts dir`", g.TextsDir)
		if err = os.MkdirAll(g.TextsDir, 0777); err != nil {
			log.Errore(err)
			http.Error(wr, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	file := filepath.Join(g.TextsDir, setName+".json")
	tmpf := file + "~"
	log.Infoa("save to `texts file`", file)
	txtwr, err := os.Create(tmpf)
	if err != nil {
		log.Errore(err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	defer txtwr.Close()
	if _, err := io.Copy(txtwr, rq.Body); err != nil { // TODO limit size?
		log.Errore(err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	txtwr.Close()
	if err = os.Rename(tmpf, file); err != nil {
		log.Errore(err)
	}
}
