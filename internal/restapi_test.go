package internal

import (
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
