package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.fractalqb.de/fractalqb/c4hgol"
)

func TestAuth(t *testing.T) {
	if testing.Verbose() {
		logCfg.SetLevel(c4hgol.Debug)
	} else {
		logCfg.SetOutput(io.Discard)
	}
	cfg.authCreds = "test:test"
	hdlr := auth(func(wr http.ResponseWriter, rq *http.Request) {
		wr.WriteHeader(http.StatusOK)
	})
	t.Run("wrong auth", func(t *testing.T) {
		rq := httptest.NewRequest("", "/", nil)
		rq.RemoteAddr = "[::1]:4711"
		rq.SetBasicAuth("test", "text")
		rrec := httptest.NewRecorder()
		hdlr(rrec, rq)
		if rrec.Code != http.StatusForbidden {
			t.Errorf("expect 403 forbidden, got: %s", rrec.Result().Status)
		}
	})
	t.Run("correct auth", func(t *testing.T) {
		rq := httptest.NewRequest("", "/", nil)
		rq.RemoteAddr = "[::1]:4711"
		rq.SetBasicAuth("test", "test")
		rrec := httptest.NewRecorder()
		hdlr(rrec, rq)
		if rrec.Code != http.StatusOK {
			t.Errorf("expect 200 OK, got: %s", rrec.Result().Status)
		}
	})
	t.Run("reject 2nd client", func(t *testing.T) {
		rq := httptest.NewRequest("", "/", nil)
		rq.RemoteAddr = "127.0.0.1:4711"
		rq.SetBasicAuth("test", "test")
		rrec := httptest.NewRecorder()
		hdlr(rrec, rq)
		if rrec.Code != http.StatusForbidden {
			t.Errorf("expect 403 Forbidden, got: %s", rrec.Result().Status)
		}
	})
	cfg.singleClient = ""
	t.Run("reject non-local address", func(t *testing.T) {
		rq := httptest.NewRequest("", "/", nil)
		rq.RemoteAddr = "8.8.8.8:4711"
		rq.SetBasicAuth("test", "test")
		rrec := httptest.NewRecorder()
		hdlr(rrec, rq)
		if rrec.Code != http.StatusForbidden {
			t.Errorf("expect 403 Forbidden, got: %s", rrec.Result().Status)
		}
	})
}
