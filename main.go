package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/knieriem/text/ini"
	"github.com/knieriem/tool"

	"github.com/knieriem/gomodsrv/internal/go.cmd/modfetch/codehost"
)

var serviceAddr = ":7070"

type confData struct {
	ServiceAddr        string
	VcsModulesRoots    []string
	FallbackToModCache bool
	CodeHostDir        string
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
	fmt.Println("\t\t"+module, v.Info.Version, gomodState)
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

func main() {
	var conf confData
	f := ini.NewFile("gomodsrv.ini", ".ini", "ini")
	ini.BindHomeLib()
	flag.Parse()

	err := f.Parse(&conf)
	fmt.Println("# Go mod proxy v0.1 (" + f.Using + ")")
	if err != nil {
		errExit(err)
	}
	codehost.WorkRoot = conf.CodeHostDir
	roots := conf.VcsModulesRoots
	if len(roots) == 0 {
		fmt.Println("No vcs module root defined. Exiting.")
		os.Exit(0)
	}
	mm := make(ModuleMap, 128)
	for _, root := range roots {
		err := vcsRootScanModules(mm, root)
		if err != nil {
			errExit(err)
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
			v.WriteZIP(w)
		})
		http.Handle("/"+path+"/", r)
	}
	if conf.FallbackToModCache {
		list := filepath.SplitList(build.Default.GOPATH)
		if len(list) != 0 {
			goModCache := filepath.Join(list[0], "pkg", "mod", "cache", "download")
			http.Handle("/", http.FileServer(http.Dir(goModCache)))
		}
	}

	if conf.ServiceAddr != "" {
		serviceAddr = conf.ServiceAddr
	}
	fmt.Println("\n* listening on", serviceAddr)
	http.ListenAndServe(serviceAddr, nil)
}

func vcsRootScanModules(dest ModuleMap, baseDir string) error {
	fmt.Println("* scanning directories below", baseDir+":")
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
				fmt.Println("\n\t"+root, "(hg. skipped)")
				return filepath.SkipDir
			}
			fmt.Println("\n\t"+root, "(hg)")
			err := ScanVCS(dest, "hg", baseDir, root)
			if err != nil {
				return err
			}
			return filepath.SkipDir
		}
		if info.Name() == ".git" {
			fmt.Println("\n\t"+root, "(git)")
			err := ScanVCS(dest, "git", baseDir, root)
			if err != nil {
				return err
			}
			return filepath.SkipDir
		}
		return nil
	})
	fmt.Println()
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
