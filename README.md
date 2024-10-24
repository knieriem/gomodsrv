# gomodsrv

Gomodsrv is acting as a [Go module proxy],
serving versioned modules from local VCS repositories.
This may be useful for cases where module paths like your-domain.com/x/y are used,
but no Git or Mercurial servers provide the repositories,
instead the repositories are kept private in a file system structure
(that's a different use case than the one supported by the GOPRIVATE variable).

When preliminary support for modules was added to Go as part of the 1.11 release in August 2018,
I wrote `gomodsrv` as a simple tool to be able to serve my private repositories via
the [GOPROXY protocol].
There is initial support for Git and Mercurial repositories,
based on the `go` command's internal `modfetch/codehost` package.
This way module ZIP files are created the same way as by the `go` tool.
The program has been in an early stage for some years,
but it got gradually cleaned up to make it easier to use.

[Go module proxy]: https://go.dev/ref/mod#module-proxy
[GOPROXY protocol]: https://go.dev/ref/mod#goproxy-protocol


## Configuration

Gomodsrv first looks for a configuration file `gomodsrv.ini` in the current directory or in one of its parent directories.
If no file can be found,
gomodsrv will use the location specified in the environment variable `GOMODSRVINI`.
This variable can be set using `gomodsrv env -w GOMODSRVINI=/path/to/file`.

The configuration file has the following, tab-indented structure:

	vcs-module-roots
		<path to the root of a file system tree containing vcs-controlled modules>
		<another path>

	service-addr	<address to listen to>

For example:

	vcs-module-roots
		/home/src

	service-addr	:7070

An alternative configuration file may be specified by overriding `GOMODSRVINI`.

Similar to the `go` cmd,
_gomodsrv_ keeps environment variables in a file located under `os.UserConfigDir() + "github.com-knieriem-gomodsrv/env"`.

_Gomodsrv_ is using a cache managed by `modfetch/codehost`,
keeping local copies of repositories.
The location can be configured using the `GOMODSRVCACHE` environment variable,
which is `os.UserConfDir() + "github.com-knieriem-gomodsrv"` on default.


## Running gomodsrv

Gomodsrv supports subcommands similar to the `go` command, actually using code from the latter.

-	serve

	`gomodsrv serve` sets up the proxy and listens for connections.
	The server can be stopped using Control-C.

-	sh

	From a shell, typing

		gomodsrv sh

	will, similar like `serve`, set up a proxy listening for connections,
	but also spawn a sub shell with GOPROXY set appropriately.


## TODO

-	update cache if new module versions are checked into local
	repositories (Currently, `gomodsrv` needs to be restarted in
	order to make changes to local repositories visible)

-	to enable faster startup, speed up Mercurial access, caching some information

-	support module versions ≥v2 that are managed in subdirectories

-	support multiple modules per repository
