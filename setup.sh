#!/bin/sh

set -e

gopackages="\
	lazyregexp\
"

gocmdpackages="\
	par\
	modfetch/codehost\
	lockedfile\
	str\
	module/module.go\
	modfile\
	modfetch/repo.go\
	modfetch/coderepo.go\
	modfetch/pseudo.go\
	semver/semver.go\
"

goroot=`go env GOROOT`

mkdir internal
mkdir internal/go

(cd $goroot/src/internal && tar cf - $gopackages) | (cd internal/go && tar xf - )

mkdir internal/go.cmd

(cd $goroot/src/cmd/go/internal && tar cf - $gocmdpackages) | (cd internal/go.cmd && tar xf - )

ed < modfetch_repo.go.ed
cp _coderepo_ext.go internal/go.cmd/modfetch/coderepo_ext.go

for f in `find internal -type f -name '*.go'`; do
	mv $f $f,
	sed 's,"cmd/go/internal,"github.com/knieriem/gomodsrv/internal/go.cmd,;s,"internal/,"github.com/knieriem/gomodsrv/internal/go/,' <$f, >$f
	rm -f $f,
done

mkdir internal/go.cmd/cfg
cat <<EOF > internal/go.cmd/cfg/cfg.go
package cfg

var BuildX bool
EOF
