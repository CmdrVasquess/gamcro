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
	tTools target = "tools"
	tGen   target = "gen"
	tWebUI target = "web-ui"
	tBuild target = "build"
	tTest  target = "test"
	tDist  target = "dist"
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
	tasks.Def(tTools, func(dir *gomk.WDir) {
		task.GetVersioner(dir.Build())
	})

	tasks.Def(tGen, func(dir *gomk.WDir) {
		dir.Exec("go", "generate", "./...")
	}, tTools)

	tasks.Def(tWebUI, func(dir *gomk.WDir) {
		dir.Cd("web-ui").
			Exec("npm", "run", "build")
	})

	tasks.Def(tBuild, func(dir *gomk.WDir) {
		dir.Exec("go", buildCmd...)
		dir.Cd("gamcrow").Exec("go", buildCmd...)
	})

	tasks.Def(tTest, func(dir *gomk.WDir) {
		dir.Exec("go", "test", "./...")
	})

	tasks.Def(tDist, nil, tWebUI, tGen, tTest, tBuild)
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
		tasks.Run(tDist, build.WDir())
	} else {
		for _, task := range flag.Args() {
			tasks.Run(task, build.WDir())
		}
	}
}
