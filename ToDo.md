
* since Go 1.21, version "1.x" should be equvalent to "1.x.".
  * remove line "*** You need to add ... to your PATH." in outout.

* download pre-built release bootstrap versions wben needed.
  from https://dl.google.com,
  so that no go installation is needed to process (code: https://github.com/golang/dl).
  * Downloaded bootstrap toolchains are also put in "tag_go1.x.y" folders,
    and put a line "how: download | build" in gotv.info.

* gotv set-default VERSION
  then "gotv go-command ..." will use the default version
  "default-version" dir only contains one file, which record a toolchainVersion.
  * and put a line "why: default | cache" in gotv.info.

* pin-version new implementation
  * check if pinned, if true, do nothing.
    check if cached, if true, then copy to pin dir and write a file.
    if false, cached it in a temp dir, then rename it to pin dir
  * deprecate pinned-version and commands.
    For backward-capacity, keep them, but must set GOTOOLCHAIN=local.
    * Use a new implementation: build a speical "go" and "gofmt" commands
      which will call a cached toolchain instaed (with set evn GOTOOLCHAIN=local).

* cache/uncache-version -> install/uninstall-version ?

* always set enve var GOTOOLCHAIN=local

* now, "set CGO_ENABLED=0" on windows.
  ToDo: download zig and "set CC=zig cc", ...

* more tests
  * need a way to simulate the remote clone
  * need a -silent option to hide gotv messages
    * good for testing

* gotv list-versions [-cached] [-pinned] [-releases] [-branches] [-incomplete]
* gotv cache-version [ToolchainVersion ...]
	* none for listing
* gotv uncache-version [ToolchainVersion ...]
	* none for all

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer (as a indovidual module)

* error if there are local modificaitons, notify clean these modificaitons?

* download a bootstrap version if no system go installation found
  * use 1.4 for 1.20-
  * use go1.17.13.GOOS-GOARCH.[tar.gz|zip] for 1.20+
  * and https://github.com/golang/go/issues/54265
  * create some bootstrap projects, embedding toolchain tar.gz in code.
    use "go install go101.org/bootstrap-xxx@latest" to download.
    untar them in gotv_cache/dir/bootstrap-xxx
    * https://github.com/go101/bootstrap-go1.17.13
    * https://github.com/go101/bootstrap-go1.4

* unable to build toolchain with versions <= 1.5.n

* building a toolchain in a temp dir under cache dir,
  then rename it when succeeds (might need to delete the old outdated dir)

* handle os.Interrupt, syscall.SIGTERM

* Not totally the same as the releases downloaded from Go website.
  use Go 1.19rc1 to build Go 1.19?
  Using a lower version will make the go command in 1.19 not recognize GOMEMLIMIT env var.
  Or a version itself (built with an older verison and pin it firstly) to build the version.

* https://go.dev/VERSION?m=text check lastest released version
