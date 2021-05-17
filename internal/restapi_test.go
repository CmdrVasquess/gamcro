package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidKbdTapQuery(t *testing.T) {
	type Query map[string][]string
	test := func(name string, expectError bool, q Query) {
		t.Run(name, func(t *testing.T) {
			err := validKbdTapQuery.Validate(q)
			if expectError {
				if err == nil {
					t.Error("validation did not fail")
				}
			} else if err != nil {
				t.Errorf("unexpectedly invalid: %s", err)
			}
		})
	}
	test("empty query", false, Query{})
	test("nil arg", true, Query{"arg": nil})
	test("empty arg", true, Query{"arg": []string{}})
	test("one arg", false, Query{"arg": []string{"foo"}})
	test("two args", false, Query{"arg": []string{"foo", "bar"}})
	test("wrong key", true, Query{"wrong": []string{"arg"}})
	test("additional wrong key", true, Query{
		"arg":   []string{"foo"},
		"wrong": []string{"arg"},
	})
}

func TestAPISwitches(t *testing.T) {
	var gamcro Gamcro
	var asTest = func(
		h http.HandlerFunc,
		method, path, body string,
	) {
		var brd io.Reader
		if body != "" {
			brd = strings.NewReader(body)
		}
		rq := httptest.NewRequest(method, path, brd)
		rq.RemoteAddr = "[::1]:4711"
		rrec := httptest.NewRecorder()
		h(rrec, rq)
		if rrec.Code != http.StatusForbidden {
			t.Errorf("api call accepted: %s", rrec.Result().Status)
		}
		if b := rrec.Body.String(); !strings.HasPrefix(b, "Inactive: ") {
			t.Errorf("Unxpected body: [%s]", b)
		}
	}

	asTest(gamcro.handleKeyboardType, http.MethodPost, "/keyboard/type", "keyboard type")
	asTest(gamcro.handleKeyboardTap, http.MethodPost, "/keyboard/tap/x", "")
	asTest(gamcro.handleClipPost, http.MethodPost, "/clip", "clip post")
	asTest(gamcro.handleClipGet, http.MethodGet, "/clip", "")
}
