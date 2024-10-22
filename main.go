package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/knieriem/text/ini"
	"github.com/knieriem/tool"

	"github.com/knieriem/gointernal/cmd/cli"
	"github.com/knieriem/gointernal/cmd/go/base"
	"github.com/knieriem/gointernal/cmd/go/cfg"
	"github.com/knieriem/gointernal/cmd/go/envcmd"
)

var serviceAddr = ":7070"

type confData struct {
	ServiceAddr     string
	VcsModulesRoots []string
}

type ModuleMap map[string]*Module

func (mm ModuleMap) AddVersion(module string, v *ModVersion) {
	m := mm[module]
	if m == nil {
		m = new(Module)
		m.Name = module
		mm[module] = m
	}
	gomodState := ""
	if v.GoMod != nil {
		gomodState = "mod"
	}
	fmt.Fprintln(info, "\t\t"+module, v.Info.Version, gomodState)
	m.Versions = append(m.Versions, v)
}

type Module struct {
	Name     string
	Versions []*ModVersion
}

type RevInfo struct {
	Version string
	Time    time.Time
}
type ModVersion struct {
	Info     RevInfo
	GoMod    []byte
	WriteZIP func(io.Writer) error
}

type VCSRevision interface {
	WriteZIP(w io.Writer) error
}

var prog = &cli.Command{
	UsageLine: "gomodsrv",
	Long:      "Gomodsrv makes local repos accessible via the GOPROXY protocol",
	Commands: []*cli.Command{
		cmdServe,
		cmdSh,
		envcmd.CmdEnv,
	},
}

func setupEnv() {
	cfg.EnvName = "GOMODSRVENV"
	cfg.ConfigDirname = "github.com-knieriem-gomodsrv"

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		errExit(err)
	}

	env := []cfg.EnvVar{
		{Name: "GOMODSRVCACHE", Value: filepath.Join(cacheDir, cfg.ConfigDirname), Var: &cfg.GOMODCACHE},
		{Name: "GOMODSRVINI", Value: "gomodsrv.ini", Var: &confFilename},
	}
	cfg.SetupEnv(env)

}

func main() {
	setupEnv()

	base.Prog = prog
	flag.Parse()

	cli.EvalArgs(flag.Args())
}

var cmdServe = &cli.Command{
	UsageLine: "gomodsrv serve [flags]",
	Short:     "serve local repositories via the GOPROXY protocol",
	Run:       serveRepos,
}

var cmdSh = &cli.Command{
	UsageLine: "gomodsrv sh [flags]",
	Short:     "like serve, but running an interactive shell with GOPROXY adjusted",
	Run:       serveShell,
}

func serveRepos(_ context.Context, cmd *cli.Command, args []string) {
	err := setupProxy()
	if err != nil {
		errExit(err)
	}
	if conf.ServiceAddr != "" {
		serviceAddr = conf.ServiceAddr
	}
	fmt.Println("* listening on", serviceAddr)
	err = http.ListenAndServe(serviceAddr, nil)
	if err != nil {
		errExit(err)
	}
}

func serveShell(_ context.Context, _ *cli.Command, args []string) {
	err := setupProxy()
	if err != nil {
		errExit(err)
	}
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		errExit(err)
	}
	addr := ln.Addr().String()
	go http.Serve(ln, nil)

	output, err := exec.Command("go", "env", "GOPROXY").Output()
	if err != nil {
		errExit(err)
	}

	tail := ""
	if len(output) != 0 {
		tail = "," + string(output)
	}

	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		shell = "/bin/bash"
	}
	env := os.Environ()
	if strings.HasSuffix(shell, "rc") {
		env = setenv(env, "prompt", "goproxy% \001 ")
	}

	goproxy := "http://" + addr + tail
	fmt.Println("* setting up GOPROXY as", goproxy)
	env = setenv(env, "GOPROXY", goproxy)

	cmd := exec.Command(shell, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		errExit(err)
	}
}

func setenv(env []string, name, value string) []string {
	namePrefix := name + "="
	keyval := namePrefix + value
	for i, ent := range env {
		if strings.HasPrefix(ent, namePrefix) {
			env[i] = keyval
			return env
		}
	}
	return append(env, keyval)
}

var conf confData
var confFilename string
var info = new(bytes.Buffer)

func setupProxy() error {
	confFilename, err := filepath.Abs(confFilename)
	if err != nil {
		return err
	}

	ini.BindOS("/", "os")
	_, err = ini.ParseFile(confFilename, &conf)
	if err != nil {
		return err
	}
	roots := conf.VcsModulesRoots
	if len(roots) == 0 {
		fmt.Println("No vcs module root defined. Exiting.")
		os.Exit(0)
	}
	mm := make(ModuleMap, 128)

	fmt.Println("* scanning repositories...")
	confDir := filepath.Dir(confFilename)
	for _, root := range roots {
		rootAbs := root
		if !filepath.IsAbs(rootAbs) {
			rootAbs = filepath.Clean(filepath.Join(confDir, root))
		}
		err = vcsRootScanModules(info, mm, rootAbs)
		if err != nil {
			return err
		}
	}
	for path, mod := range mm {
		r := mux.NewRouter()
		s := r.PathPrefix("/" + path + "/").Subrouter()
		s.HandleFunc("/@v/latest", func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL.Path)
		})
		m := mod
		s.HandleFunc("/@v/list", func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL.Path)
			for _, v := range m.Versions {
				fmt.Fprintln(w, v.Info.Version)
			}

		})
		s.HandleFunc("/@v/{version}.info", func(w http.ResponseWriter, r *http.Request) {
			v := m.modVersion(r)
			if v == nil {
				return
			}
			b, err := json.Marshal(&v.Info)
			if err != nil {
				return
			}
			w.Write(b)
		})
		s.HandleFunc("/@v/{version}.mod", func(w http.ResponseWriter, r *http.Request) {
			v := m.modVersion(r)
			if v == nil {
				http.Error(w, "module version not found", http.StatusNotFound)
				return
			}
			if len(v.GoMod) == 0 {
				fmt.Fprintln(w, "module", m.Name)
				return
			}
			w.Write(v.GoMod)
		})
		s.HandleFunc("/@v/{version}.zip", func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.URL.Path)
			v := m.modVersion(r)
			if v == nil {
				return
			}
			var b bytes.Buffer
			err := v.WriteZIP(&b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			io.Copy(w, &b)
		})
		http.Handle("/"+path+"/", r)
	}
	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		w.Write(info.Bytes())
	})
	return nil
}

func vcsRootScanModules(w io.Writer, dest ModuleMap, baseDir string) error {
	fmt.Fprintln(w, "# directories below", baseDir+":")
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), "_") {
			return filepath.SkipDir
		}
		root, _ := filepath.Split(path)
		if info.Name() == ".hg" {
			if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
				fmt.Fprintln(w, "\n\t"+root, "(hg. skipped)")
				return filepath.SkipDir
			}
			fmt.Fprintln(w, "\n\t"+root, "(hg)")
			err := ScanVCS(dest, "hg", baseDir, root)
			if err != nil {
				return fmt.Errorf("vcs %q: %w", root, err)
			}
			return filepath.SkipDir
		}
		if info.Name() == ".git" {
			fmt.Fprintln(w, "\n\t"+root, "(git)")
			err := ScanVCS(dest, "git", baseDir, root)
			if err != nil {
				return fmt.Errorf("vcs %q: %w", root, err)
			}
			return filepath.SkipDir
		}
		return nil
	})
	fmt.Fprintln(w)
	return err
}

func (m *Module) modVersion(r *http.Request) *ModVersion {
	log.Println(r.URL.Path)
	vars := mux.Vars(r)
	want := vars["version"]
	for _, v := range m.Versions {
		if v.Info.Version == want {
			return v
		}
	}
	return nil
}

func errExit(err error) {
	tool.PrintErrExit(err)
	os.Exit(1)
}
