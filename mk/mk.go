package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"git.fractalqb.de/fractalqb/gomk"
	"git.fractalqb.de/fractalqb/gomk/pack"
	"git.fractalqb.de/fractalqb/gomk/task"
	"git.fractalqb.de/fractalqb/pack/versions"
)

type target = string

const (
	tTools   target = "tools"
	tGen     target = "gen"
	tWebUI   target = "webui"
	tBuild   target = "build"
	tTest    target = "test"
	tPredist target = "predist"
	tDist    target = "dist"
	tDeps    target = "deps"
)

var (
	update   bool
	buildCmd = []string{"build",
		"-ldflags", "-s -w", // "strip" executable
		"-a",         // clean build
		"--trimpath", // relative source paths
	}
	tasks = make(gomk.Tasks)
	vDef  map[string]string
	vStr  string
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
		task.GetVersioner(dir.Build(), update)
		err := task.MkGetTool(
			dir.Build(), update,
			"fyne",
			"fyne.io/fyne/v2/cmd/fyne",
		)
		if err != nil {
			panic(err)
		}
	})

	tasks.Def(tGen, func(dir *gomk.WDir) {
		gomk.Exec(dir, "go", "generate", "./...")
	}, tTools)

	tasks.Def(tWebUI, func(dir *gomk.WDir) {
		gomk.Exec(dir.Cd("web-ui"), "npm", "run", "build")
	})

	tasks.Def(tBuild, func(dir *gomk.WDir) {
		crowd := dir.Cd("gamcrow")
		gomk.Exec(dir, "go", buildCmd...)
		switch runtime.GOOS {
		case "windows":
			gomk.Exec(crowd, "fyne", "package", "-icon", "gamcrow.png")
		default:
			gomk.Exec(crowd, "go", buildCmd...)
		}
	}, tTools, tTest)

	tasks.Def(tTest, func(dir *gomk.WDir) {
		gomk.Exec(dir, "go", "test", "./...")
	})

	tasks.Def(tPredist, nil, tWebUI, tGen, tBuild)

	tasks.Def(tDeps, func(dir *gomk.WDir) {
		task.DepsGraph(dir.Build(), update)
	})

	tasks.Def(tDist, func(dir *gomk.WDir) {
		exes := []string{
			"gamcro",
			"gamcro.exe",
			"gamcrow/gamcrow",
			"gamcrow/gamcrow.exe",
		}
		distDir := dir.Cd("dist")
		must(os.RemoveAll(distDir.Join()))
		must(os.MkdirAll(distDir.Join(), 0777))
		pack.CopyToDir(dir, "dist", nil, exes...)
		for i, exe := range exes {
			base := filepath.Base(exe)
			gomk.Exec(distDir, "upx", base)
			if filepath.Ext(exe) == ".exe" {
				exes[i] = winDist(distDir, base)
			} else {
				exes[i] = linuxDist(distDir, base)
			}
		}
		shaSumFile := fmt.Sprintf("gamcro-%s.sha256", vStr)
		gomk.ExecFile(distDir, shaSumFile, "sha256sum", exes...)
		for _, exe := range exes {
			sigFile := exe + ".asc"
			gomk.Exec(distDir, "gpg", "-b", "--armor",
				"-u", "CmdrVasquess",
				"-o", sigFile,
				exe)
		}
	})
}

func winDist(dir *gomk.WDir, exe string) string {
	base := exe[:len(exe)-4]
	zip := fmt.Sprintf("%s-%s-windows-amd64.zip", base, vStr)
	gomk.Exec(dir, "zip", zip, exe)
	must(os.Remove(dir.Join(exe)))
	return zip
}

func linuxDist(dir *gomk.WDir, exe string) string {
	const suffix = ".gz"
	gzip := fmt.Sprintf("%s-%s-%s-%s.tgz", exe, vStr, runtime.GOOS, runtime.GOARCH)
	gomk.Exec(dir, "tar", "czf", gzip, exe)
	os.Remove(dir.Join(exe))
	return gzip
}

func usage() {
	flag.PrintDefaults()
	wr := flag.CommandLine.Output()
	fmt.Fprintln(wr, "Tasks:")
	tasks.Fprint(wr, "- ")
}

func main() {
	flag.Usage = usage
	fCDir := flag.String("C", "", "change working dir")
	flag.BoolVar(&update, "update", update, "check for tool updates")
	flag.Parse()
	if *fCDir != "" {
		must(os.Chdir(*fCDir))
	}
	build, _ := gomk.NewBuild("", os.Environ())
	log.Printf("project root: %s\n", build.PrjRoot)
	var err error
	vDef, err = versions.ReadFile("VERSION")
	if err != nil {
		log.Fatal(err)
	}
	vStr = fmt.Sprintf("v%s.%s.%s", //-%s+%s",
		vDef[versions.SemVerMajor.String()],
		vDef[versions.SemVerMinor.String()],
		vDef[versions.SemVerPatch.String()],
		// vDef[versions.SemVerPreRelease.String()],
		// vDef["build_no"],
	)
	log.Println(vStr)
	if len(flag.Args()) == 0 {
		tasks.Run(tDist, build.WDir())
	} else {
		for _, task := range flag.Args() {
			tasks.Run(task, build.WDir())
		}
	}
}
