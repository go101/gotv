
# gotv (Go Toolchain Version)

**gotv** is tool which provides a way to manage multiple versions of the official Go toolchain.
It is kinda of a re-implementation of [gvm](https://github.com/moovweb/gvm) but with a different command set.
Each Go toolchain version is built from the Go git repository.

This tool is mainly built for me to check differeces of official Go toolchain verisons in writing [Go 101 books](https://go101.org).

## Installation

A recent Go toolchain version (1.17+) is needed to install **gotv**.
Now, the toolchain version is also used as the bootstrap toolchain for building and caching
other toolchain versions during `gotv` command executions.

```
go install go101.org/gotv@latest
```

## Usage

Most `gotv` commands are in the following format:

```
gotv ToolchainVersion [go-arguments...]
```

During running the first such a command, the Go git repository will be cloned (need several minutes to finish).

`ToolchainVersion` might be
* a Go release version, such as `1.17.13`, `1.18`, `1.19rc1`,
  which mean the release tag `go1.17.13`, `go1.18`, `go1.19rc1`, respectively,
  in the Go git repository.
  Note: `1.N.` means the latest release of `1.N` and `1.` means the latest Go 1 release verison.
* `:tip`, which means the local latest `master` branch in the Go git repository.
* `:1.N`, which means the local latest `release-branch.go1.N` branch in the Go git repository.

Examples:

```
$ gotv 1.17.12 version
[Run]: $HOME/.cache/gotv/tag_go1.17.12/bin/go version
go version go1.17.12 linux/amd64

$ gotv 1.18.3 version
[Run]: $HOME/.cache/gotv/tag_go1.18.3/bin/go version
go version go1.18.3 linux/amd64

$ cat main.go
package main

const A = 3

func main() {
	const (
		A = A + A
		B
	)
	println(A, B)
}

$ gotv 1.17.12 run main.go
[Run]: $HOME/.cache/gotv/tag_go1.17.12/bin/go run main.go
6 6

$ gotv 1.18.3 run  main.go
[Run]: $HOME/.cache/gotv/tag_go1.18.3/bin/go run main.go
6 12
```

Other `gotv` commands:

```
gotv fetch-versions
gotv list-versions
gotv cache-version ToolchainVersion [ToolchainVersion ...]
gotv uncache-version ToolchainVersion [ToolchainVersion ...]
gotv pin-version ToolchainVersion
gotv unpin-version
```