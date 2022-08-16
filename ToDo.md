
* more tests
  * need a way to simulate the remote clone
  * need a -silent option to hide gotv messages
    * good for testing

* gotv list-versions [-cached] [-pinned] [-releases] [-branches]

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer (as a indovidual module)

* error if there are local modificaitons, notify clean these modificaitons?

* download a bootstrap version if no system go installation found
  * use go1.17.13.GOOS-GOARCH.[tar.gz|zip].
  * and https://github.com/golang/go/issues/54265

* unable to build toolchain with versions <= 1.5.n

* building a toolchain in a temp dir under cache dir,
  then rename it when succeeds (might need to delete the old outdated dir)

* handle os.Interrupt, syscall.SIGTERM
