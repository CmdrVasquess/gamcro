package main

import (
	"io"
	"net/http"

	"github.com/atotto/clipboard"
	"github.com/go-vgo/robotgo"
	"github.com/gorilla/mux"
)

func apiRoutes(r *mux.Router) {
	r.HandleFunc("/keyboard/type", auth(handleKeyboardType)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
	r.HandleFunc("/clip", auth(handleClipStr)).
		Methods(http.MethodPost).
		HeadersRegexp("Content-Type", "text/plain")
}

func rqBodyRd(rq *http.Request) io.Reader {
	var rd io.Reader = rq.Body
	if inLimit > 0 {
		rd = io.LimitReader(rd, int64(inLimit))
	}
	return rd
}

func rqBody(rq *http.Request) ([]byte, error) {
	rd := rqBodyRd(rq)
	return io.ReadAll(rd)
}

func handleKeyboardType(wr http.ResponseWriter, rq *http.Request) {
	body, err := rqBody(rq)
	if err != nil {
		log.Errora("Read body failed with `err`", err)
		http.Error(wr, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		return
	}
	txt := string(body)
	log.Debuga("type `str`", txt)
	robotgo.TypeStr(txt)
}

func handleClipStr(wr http.ResponseWriter, rq *http.Request) {
	body, err := rqBody(rq)
	if err == nil && len(body) > 0 {
		s := string(body)
		log.Debuga("`clip`", s)
		if err = clipboard.WriteAll(s); err != nil {
			log.Errore(err)
		}
	} else if err != nil {
		log.Errora("Read body failed with `err`", err)
	}
}
