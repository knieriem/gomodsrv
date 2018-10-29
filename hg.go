package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/knieriem/gomodsrv/internal/go/modfetch"
	"github.com/knieriem/gomodsrv/internal/go/semver"
)

func ScanMercurialVCS(mm ModuleMap, pathPrefix, repoRoot string) error {
	cmd := exec.Command("hg", "-R", repoRoot, "tags")
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 2 {
			continue
		}
		v := f[0]
		fNode := strings.SplitN(f[1], ":", 2)
		if len(fNode) != 2 {
			continue
		}
		node := fNode[1]
		if v == "tip" {
		} else if !semver.IsValid(v) {
			continue
		}
		cmd := exec.Command("hg", "--cwd", repoRoot, "cat", "-r", node, "go.mod")
		out, err := cmd.Output()
		modName := strings.TrimPrefix(repoRoot, pathPrefix+"/")
		modName = strings.TrimSuffix(modName, "/")
		m := new(ModVersion)
		if err == nil {
			name, err := readGomodIncludePath(out)
			if err != nil {
				return err
			}
			modName = name
			m.GoMod = out
		}
		m.Info.Time, err = getHgRevTime(repoRoot, node)
		if err != nil {
			return err
		}
		if v == "tip" {
			v = modfetch.PseudoVersion("", "", m.Info.Time, node)
		}
		m.Info.Version = v
		m.Rev = &mercurialRev{
			node:       node,
			repoRoot:   repoRoot,
			pathPrefix: modName + "@" + v,
		}
		mm.AddVersion(modName, m)
	}
	return nil
}

type mercurialRev struct {
	pathPrefix string
	repoRoot   string
	node       string
}

func (r *mercurialRev) WriteZIP(w io.Writer) error {
	cmd := exec.Command("hg", "-R", r.repoRoot, "archive", "-r", r.node, "-t", "zip", "-p", r.pathPrefix, "-")
	out, err := cmd.Output()
	if err != nil {
		log.Println(err, cmd)
		return err
	}
	_, err = w.Write(out)
	return err
}

func getHgRevTime(repoRoot, node string) (time.Time, error) {
	cmd := exec.Command("hg", "-R", repoRoot, "log", "-r", node, "-T", `{date|rfc3339date}`)
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, string(out))
	if err == nil {
		t = t.UTC()
	}
	return t, err
}
