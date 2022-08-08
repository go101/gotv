
* do auth for git@github.com:golang/go.git

* need a -silent option to hide gotv messages
  * good for testing

* handle os.Interrupt, syscall.SIGTERM

* replace /home/user/.cache to $HOME/.cache in all outputs
  * need to implement a ReplaceWriter io.Writer (as a indovidual module)

* error if there are local modificaitons, notify clean these modificaitons?

* bootstrap version:
  * use go1.17.13.GOOS-GOARCH.[tar.gz|zip].
  * and https://github.com/golang/go/issues/54265

* run "gotv 1.9 ...", get error
  ERROR: Cannot find /home/lx/go1.4/bin/go.
  Set $GOROOT_BOOTSTRAP to a working Go tree >= Go 1.4.

* building a toolchain in a temp dir under cache dir,
  then rename it when succeeds (might need to delete the old outdated dir)
