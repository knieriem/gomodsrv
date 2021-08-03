Gomodsrv is acting as a GOPROXY (see `go help goproxy`),
serving versioned modules from local VCS repositories.
This may be useful for cases where you are using module paths like your-domain.com/x/y,
but no Git or Mercurial servers provide the repositories,
but instead the repositories are kept private in a file system structure
(that's a different use case than the one supported by the GOPRIVATE variable).

Currently there is initial support for Git and Mercurial repositories,
based on the `go` command's internal `modfetch/codehost` package.
This way module ZIP files are created the same way as by the `go` tool.
The program is still in an early stage,
but so far appears sufficient to make my private modules available to Go locally.

Gomodsrv can be run without arguments;
a configuration file $HOME/lib/gomodsrv.ini has to be created with the following contents:

	vcs-module-roots
		<path to the root of a file system tree containing vcs-controlled modules>
		<another path>

	code-host-dir	<a directory, managed by modfetch/codehost, keeping local copies of repositories>

	service-addr	<address to listen to>

For example:

	vcs-module-roots
		/home/src

	code-host-dir	/home/gomodsrv/codehost

	service-addr	:7070

An alternative configuration file may be specified using option `-c`.

### TODO

-	update cache if new module versions are checked into local
	repositories (Currently, `gomodsrv` needs to be restarted in
	order to make changes to local repositories visible)

-	to enable faster startup, speed up Mercurial access, caching some information

-	support module versions ≥v2 that are managed in subdirectories

-	support multiple modules per repository
