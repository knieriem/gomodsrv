Gomodsrv is acting as a GOPROXY (see `go help goproxy`), serving
versioned modules from local VCS repositories. Currently only
Mercurial repositories are supported. The program is still
in an early stage, but so far appears sufficient to make my
private modules available to Go.

It can be run without arguments; a configuration file
$HOME/lib/gomodsrv.ini has to be created with the following
contents:

	vcs-module-roots
		<path to the root of a file system tree containing vcs-controlled modules>
		<another path>

	fallback	<go.mod cache to use if an unknown module is requested>

	service-addr	<address to listen to>

For example:

	vcs-module-roots
		/home/src

	fallback	/home/go/pkg/mod/cache/download

	service-addr	:7070


### TODO

-	update cache if new module versions are checked into local
	repositories (Currently, `gomodsrv` needs to be restarted in
	order to make changes to local repositories visible)

-	support Git.
