


* "gotv uncache-version !." removes all non-latest versions
  "gotv uncache-version !1.n" removes all not-latest version of 1.n

* "gotv :tip" print the tip version firstly

* reimplement gtv commands for godev, but not build toolchains from git repo,
  but download from https://dl.google.com instead

* download pre-built release bootstrap versions wben needed.
  from https://dl.google.com,
  so that no go installation is needed to process (code: https://github.com/golang/dl).
  * Downloaded bootstrap toolchains are also put in "tag_go1.x.y" folders,
    and put a line "how: download | build" in gotv.info.

* download a bootstrap version if no system go installation found
  * use 1.4 for 1.20-
  * use go1.17.13.GOOS-GOARCH.[tar.gz|zip] for 1.20+
  * and https://github.com/golang/go/issues/54265
  * and https://github.com/golang/go/issues/64751
  * create some bootstrap projects, embedding toolchain tar.gz in code.
    use "go install go101.org/bootstrap-xxx@latest" to download.
    untar them in gotv_cache/dir/bootstrap-xxx
    * https://github.com/go101/bootstrap-go1.17.13
    * https://github.com/go101/bootstrap-go1.4

* gtv 1.18 -env:CGO_ENBLED=1



* unable to build toolchain with versions <= 1.5.n
  * download them (need to set GOROOT etc env vars before run them)

* https://go.dev/VERSION?m=text check lastest released version

* add a `gotv gofmt` custom command, to call the `gofmt` coomand.

* pin-version new implementation
  * build fake "go" and "gofmt" commands, which will call the real commands.
    The pinned version info is recorded in config file.

* now, "set CGO_ENABLED=0" on windows.
  ToDo: download zig and "set CC=zig cc", ...

* more tests
  * need a way to simulate the remote clone
  * need a -silent option to hide gotv messages
    * good for testing

* gotv list-versions [-cached] [-releases] [-branches]
* gotv uncache-version `gotv list-versions -cached -oneline`

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer

* error if there are local modificaitons, notify clean these modificaitons?

* handle os.Interrupt, syscall.SIGTERM
