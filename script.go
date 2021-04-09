package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/glycerine/zygomys/zygo"
	rogo "github.com/go-vgo/robotgo"
)

var (
	script = zygo.NewZlispSandbox()
)

func init() {
	script.AddFunction("type-str", zyRoboTypeStr)
}

func loadScripts(files ...string) (err error) {
	for _, f := range files {
		err = loadScript(f)
	}
	return err
}

func loadScript(file string) error {
	rd, err := os.Open(file)
	if err != nil {
		return err
	}
	defer rd.Close()
	return script.LoadFile(rd)
}

func handleScript(wr http.ResponseWriter, rq *http.Request) {
	path := strings.Split(rq.URL.Path, "/")
	if len(path) != 3 {
		log.Errora("illegal `script path`", rq.URL.Path)
		http.Error(wr, "illegal script path", http.StatusNotFound)
		return
	}
	cmd := path[2]
	switch cmd {
	case "def", "set":
		log.Warna("script `called` `from`", cmd, rq.RemoteAddr)
		http.Error(wr, "Forbidden", http.StatusForbidden)
		return
	}
	if err := rq.ParseForm(); err != nil {
		log.Errore(err)
		http.Error(wr, "form", http.StatusInternalServerError)
		return
	}
	args := []zygo.Sexp{script.MakeSymbol(cmd)}
	argv := rq.Form["args"]
	if len(argv) > 0 {
		for _, arg := range argv {
			args = append(args, &zygo.SexpStr{S: arg})
		}
	}
	call := zygo.MakeList(args)
	log.Debuga("`script` with `args`", cmd, argv)
	if _, err := script.EvalExpressions([]zygo.Sexp{call}); err != nil {
		script.Clear()
		log.Errore(err)
		http.Error(wr, fmt.Sprint("eval:", err), http.StatusBadRequest)
	}
}

func zyRoboTypeStr(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
	for _, args := range args {
		switch a := args.(type) {
		case *zygo.SexpStr:
			rogo.TypeStr(a.S)
		}
	}
	return zygo.SexpNull, nil
}
