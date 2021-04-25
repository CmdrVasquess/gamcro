package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"git.fractalqb.de/fractalqb/c4hgol"
)

func TestAuthCreds(t *testing.T) {
	var ac AuthCreds
	if err := ac.Set("foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if ac.check("foo", "baz") {
		t.Error("Accepted wrong password 'baz'")
	}
	if !ac.check("foo", "bar") {
		t.Error("Rejected correct password 'bar'")
	}
}

func TestAuthCreds_file(t *testing.T) {
	var ac1 AuthCreds
	if err := ac1.Set("foo", "bar"); err != nil {
		t.Fatal(err)
	}
	if err := ac1.WriteFile(t.Name()); err != nil {
		t.Fatal(err)
	}
	var ac2 AuthCreds
	if err := ac2.ReadFile(t.Name()); err != nil {
		t.Fatal(err)
	}
	if ac1.user != ac2.user {
		t.Errorf("user: %s != %s", ac1.user, ac2.user)
	}
	if !reflect.DeepEqual(ac1.salt, ac2.salt) {
		t.Errorf("salt differs")
	}
	if !reflect.DeepEqual(ac1.pass, ac2.pass) {
		t.Errorf("pass differs")
	}
}

func TestAuth(t *testing.T) {
	if testing.Verbose() {
		LogCfg.SetLevel(c4hgol.Debug)
	} else {
		LogCfg.SetOutput(io.Discard)
	}
	var gamcro Gamcro
	gamcro.ClientAuth.Set("test", "test")
	hdlr := gamcro.auth(func(wr http.ResponseWriter, rq *http.Request) {
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
	gamcro.singleClient = ""
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
