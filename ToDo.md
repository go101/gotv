


* now, "set CGO_ENABLED=0" on windows.
  ToDo: download zig and "set CC=zig cc", ...

* more tests
  * need a way to simulate the remote clone
  * need a -silent option to hide gotv messages
    * good for testing

* gotv list-versions [-cached] [-pinned] [-releases] [-branches] [-incomplete]
* gotv cache-version [ToolchainVersion ...]
	* none for listing
*gotv uncache-version [ToolchainVersion ...]
	* none for all

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer (as a indovidual module)

* error if there are local modificaitons, notify clean these modificaitons?

* download a bootstrap version if no system go installation found
  * use go1.17.13.GOOS-GOARCH.[tar.gz|zip].
  * and https://github.com/golang/go/issues/54265
  * create some bootstrap projects, embedding toolchain tar.gz in code.
    use "go install go101.org/bootstrap-xxx@latest" to download.
    untar them in gotc_cache/dir/bootstrap-xxx
    * https://github.com/go101/bootstrap-go1.17.13
    * https://github.com/go101/bootstrap-go1.4.3

* unable to build toolchain with versions <= 1.5.n

* building a toolchain in a temp dir under cache dir,
  then rename it when succeeds (might need to delete the old outdated dir)

* handle os.Interrupt, syscall.SIGTERM

* Not totally the same as the releases downloaded from Go website.
  use Go 1.19rc1 to build Go 1.19?
  Using a lower version will make the go command in 1.19 not recognize GOMEMLIMIT env var.
  Or a version itself (built with an older verison and pin it firstly) to build the version.
