package modfetch

import (
	"github.com/knieriem/gomodsrv/internal/go.cmd/modfetch/codehost"
)

func NewCodeRepo(code codehost.Repo, codeRoot, path string) (Repo, error) {
	return newCodeRepo(code, codeRoot, path)
}
