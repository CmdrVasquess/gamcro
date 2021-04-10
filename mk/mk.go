package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/gomk/task"
)

type target = string

const (
	TOOLS target = "tools"
	GEN   target = "gen"
	BUILD target = "build"
	DIST  target = "dist"
)

var (
	buildCmd = []string{"build", "-a", "--trimpath"}
	tasks    = make(gomk.Tasks)
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func filter(dir string, info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "~")
}

func webAssetsFilter(dir string, info os.FileInfo) bool {
	if !filter(dir, info) {
		return false
	}
	name := info.Name()
	return !strings.HasPrefix(name, "vue.") || !strings.HasSuffix(name, ".js")
}

func init() {
	tasks.Def(TOOLS, func(dir *gomk.WDir) {
		task.GetVersioner(dir.Build())
	})

	tasks.Def(GEN, func(dir *gomk.WDir) {
		dir.Exec("go", "generate", "./...")
	}, TOOLS)

	tasks.Def(BUILD, func(dir *gomk.WDir) {
		dir.Exec("go", buildCmd...)
	})

	tasks.Def(DIST, nil, GEN, BUILD)
}

func main() {
	fCDir := flag.String("C", "", "change working dir")
	flag.Parse()
	if *fCDir != "" {
		must(os.Chdir(*fCDir))
	}
	build, _ := gomk.NewBuild("", os.Environ())
	log.Printf("project root: %s\n", build.PrjRoot)
	if len(flag.Args()) == 0 {
		tasks.Run(DIST, build.WDir())
	} else {
		for _, task := range flag.Args() {
			tasks.Run(task, build.WDir())
		}
	}
}
