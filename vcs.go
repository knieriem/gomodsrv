package main

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/knieriem/gointernal/cmd/go/modfetch"
	"github.com/knieriem/gointernal/cmd/go/modfetch/codehost"
	"golang.org/x/mod/semver"
)

func ScanVCS(mm ModuleMap, vcs, pathPrefix, repoRoot string) error {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	remote := "file://" + root
	repo, err := codehost.NewRepo(vcs, remote)
	if err != nil {
		return err
	}
	tags, err := repo.Tags("v")
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if !semver.IsValid(tag) {
			continue
		}
		err := setupModVersion(mm, repo, tag, tag, repoRoot, pathPrefix)
		if err != nil {
			return err
		}
	}
	info, err := repo.Latest()
	if err != nil {
		return err
	}
	v := modfetch.PseudoVersion("", "", info.Time, info.Short)
	err = setupModVersion(mm, repo, info.Short, v, repoRoot, pathPrefix)
	if err != nil {
		return err
	}
	return nil
}

func setupModVersion(mm ModuleMap, repo codehost.Repo, rev, v, repoRoot, pathPrefix string) error {
	m, modName, err := parseMod(repo, rev, repoRoot, pathPrefix)
	if err != nil {
		return err
	}
	m.Info.Version = v
	m.WriteZIP = func(w io.Writer) error {
		r, err := modfetch.NewCodeRepo(repo, modName, modName)
		if err != nil {
			return err
		}
		err = r.Zip(w, v)
		if err != nil {
			return err
		}
		return nil
	}
	mm.AddVersion(modName, m)
	return nil
}

func parseMod(repo codehost.Repo, rev string, repoRoot, pathPrefix string) (m *ModVersion, name string, err error) {
	out, err := repo.ReadFile(rev, "go.mod", 1e9)
	modName := strings.TrimPrefix(repoRoot, pathPrefix+"/")
	modName = strings.TrimSuffix(modName, "/")
	m = new(ModVersion)
	if err == nil {
		name, err := readGomodIncludePath(out)
		if err != nil {
			return nil, "", err
		}
		modName = name
		m.GoMod = out
	}
	return m, modName, nil
}
