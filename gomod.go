package main

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"strings"
)

func readGomodIncludePath(gomod []byte) (string, error) {
	s := bufio.NewScanner(bytes.NewReader(gomod))
	n := 0
	for s.Scan() {
		n++
		line := s.Text()
		if !strings.HasPrefix(line, "module") {
			continue
		}
		f := strings.SplitN(line, " ", 2)
		if len(f) != 2 || f[0] != "module" {
			log.Println(len(f), f)
			return "", errors.New("syntax error when parsing module declaration")
		}
		return f[1], nil
	}
	return "", errors.New("module line not found")
}
