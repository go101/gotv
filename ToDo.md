* don't output "[Run]: ..." logs for some commands, such as 'gotv env` and `gotv version`.
  The output might be captured in some shell scripts.

* "gotv uncache-version !." removes all non-latest versions
  "gotv uncache-version !1.n" removes all not-latest version of 1.n

* gotv list-versions [-cached] [-releases] [-branches]
* gotv uncache-version -keep-latest `gotv list-versions -cached -oneline`
  * keep-latest: keep the latest release
  * keep-latests: keep the latest releases for each minor version.

* reimplement gtv commands for godev, but not build toolchains from git repo,
  but download from https://dl.google.com instead

* download pre-built release bootstrap versions wben needed.
  from https://dl.google.com,
  so that no go installation is needed to process (code: https://github.com/golang/dl).
  * Downloaded bootstrap toolchains are also put in "tagtvgo1.x.y" folders,
    and put a line "how: download | build" in gotv.info.
tv
  * create some bootstrap projects, embedding toolchain tar.gz in code.
    use "go install go101.org/bootstrap-xxx@latest" to download.
    untar them in gotv_cache/dir/bootstrap-xxx
    * https://github.com/go101/bootstrap-go1.17.13
    * https://github.com/go101/bootstrap-go1.4
  * now, if pinned version is 1.21.x, and to build 1.24 versions, get
    building_Go_requires_Go_1_22_6_or_later alike errors.

* gotv 1.18 -env:CGO_ENABLED=1


* unable to build toolchain with versions <= 1.5.n
  * download them (need to set GOROOT etc env vars before run them)

* https://go.dev/VERSION?m=text check latest released version

* add a `gotv gofmt` custom command, to call the `gofmt` command.

* pin-version new implementation
  * build wrapper "go" and "gofmt" commands, which will call the real commands.
    The pinned version info is recorded in config file.

* now, "set CGO_ENABLED=0" on windows.
  ToDo: download zig and "set CC=zig cc", ...

* more tests
  * need a way to simulate the remote clone
  * need a -silent option to hide gotv messages
    * good for testing

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer

* error if there are local modifications, notify clean these modifications?

* handle os.Interrupt, syscall.SIGTERM
