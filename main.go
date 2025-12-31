package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go101.org/gotv/internal/util"
)

func _(x []int) *[1]int {
	return (*[1]int)(x) // requires 1.17+ toolchains
}

func main() {
	args := make([]string, len(os.Args))
	copy(args, os.Args)

	program := args[0]
	if len(args) < 2 || args[1] == "-h" || args[1] == "--help" {
		printUsage(program)
		return
	}

	if args[1] == "release" {
		releaseGoTV()
		return
	}

	gotv, err := born()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args = args[1:]

	err = gotv.tryRunningSpecialCommand(args)
	switch err {
	case nil:
		return
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	case unknownCommand{}:
		break
	}

	if tv := parseGoToolchainVersion(args[0], false); tv.kind == kind_Default {
		tv = gotv.DefaultVersion()
		if invalid, _ := tv.IsInvalid(); invalid {
			//fmt.Print(".\n\n")
			//printSetDefaultVersion(program)
			//os.Exit(1)
			fmt.Print("No toolchain version is provided, try to use the latest release version.\n\n")
			tv = parseGoToolchainVersion(".", true)
		} else {
			fmt.Printf("No toolchain version is provided, try to use default version (%v).\n\n", tv)
		}

		if err := gotv.tryRunningGoToolchainCommand(tv, args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if invalid, message := tv.IsInvalid(); invalid {
		fmt.Fprintln(os.Stderr, message)
		os.Exit(1)
	} else {
		if err := gotv.tryRunningGoToolchainCommand(tv, args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

const descToolchainVersion = `where ToolchainVersion might be
	* a Go release version, such as 1.17.13, 1.18,
	  and 1.19rc1, which mean the release tags
	  go1.17.13, go1.18, go1.19rc1, respectively,
	  in the Go git repository.
	  Note:
	  * 1.N. means the latest release of Go 1.N
	    versions. If N >= 21, then 1.N also means
	    the latest release of Go 1.N versions.
	  * 1. means the latest Go 1 release version.
	  * . means the latest Go release version.
	* :tip, which means the local latest master
	  branch in the Go git repository.
	* :N.M, such as :1.17, :1.18 and :1.19, which mean
	  the local latest release-branch.goN.M branch
	  in the Go git repository.`

func printSetDefaultVersion(program string) {
	fmt.Fprintf(os.Stderr, `It looks you want to use the default toolchain version,
but it has not been set yet. Please run the following
command to set it:

	%s default-version ToolchainVersion

	%s
`,
		filepath.Base(program),
		descToolchainVersion,
	)
}

func printUsage(program string) {
	fmt.Printf(`GoTV %s

Usage (to use a specific Go toolchain version):
	%s ToolchainVersion [go-arguments...]

	%s

	A ToolchainVersion suffixed with ! means remote
	versions are needed to be fetched firstly.

GoTV specific commands:
	gotv fetch-versions
		fetch remote versions (sync git repository)
	gotv list-versions
		list all (local) releases and versions branches
	gotv cache-version ToolchainVersion [ToolchainVersion ...]
		cache one or more versions
	gotv uncache-version ToolchainVersion [ToolchainVersion ...]
		uncache one or more versions
	gotv pin-version ToolchainVersion
		pin a specified version
	gotv unpin-version
		unpin the current pinned version
	gotv default-version ToolchainVersion
		set the default version
`,
		Version,
		filepath.Base(program),
		descToolchainVersion,
	)
}

const Version = "v0.2.8"

func releaseGoTV() {
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, nil, nil, "go", "test", "./..."); err != nil {
		log.Println("go test error:", err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, nil, nil, "go", "fmt", "./..."); err != nil {
		log.Println("go fmt error:", err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, nil, nil, "go", "mod", "tidy"); err != nil {
		log.Println("go mod tidy error:", err)
		return
	}

	const (
		VersionConstPrefix = `const Version = "v`
		PreviewSuffix      = "-preview"
	)

	var versionGoFile = "main.go"

	oldContent, err := os.ReadFile(versionGoFile)
	if err != nil {
		log.Printf("failed to load version.go: %s", err)
		return
	}

	m, n := bytes.Index(oldContent, []byte(VersionConstPrefix)), 0
	if m > 0 {
		m += len(VersionConstPrefix)
		n = bytes.IndexByte(oldContent[m:], '"')
		if n >= 0 {
			n += m
		}
	}
	if m <= 0 || n <= 0 {
		log.Printf("Version string not found (%d : %d)", m, n)
		return
	}

	oldVersion := bytes.TrimSuffix(oldContent[m:n], []byte(PreviewSuffix))
	noPreviewSuffix := len(oldVersion) == n-m
	mmp := bytes.SplitN(oldVersion, []byte{'.'}, -1)
	if len(mmp) != 3 {
		log.Printf("Version string not in MAJOR.MINOR.PATCH format: %s", oldVersion)
		return
	}

	major, err := strconv.Atoi(string(mmp[0]))
	if err != nil {
		log.Printf("parse MAJOR version (%s) error: %s", mmp[0], err)
		return
	}

	minor, err := strconv.Atoi(string(mmp[1]))
	if err != nil {
		log.Printf("parse MINOR version (%s) error: %s", mmp[1], err)
		return
	}

	patch, err := strconv.Atoi(string(mmp[2]))
	if err != nil {
		log.Printf("parse PATCH version (%s) error: %s", mmp[2], err)
		return
	}

	var incVersion = func() {
		patch = (patch + 1) % 10
		if patch == 0 {
			minor = (minor + 1) % 10
			if minor == 0 {
				major++
			}
		}
	}

	newContentLength := len(oldContent) + 1
	if noPreviewSuffix {
		newContentLength += len(PreviewSuffix)
		incVersion()
	}

	var newVersion, newPreviewVersion []byte

	var buf = bytes.NewBuffer(make([]byte, 0, newContentLength))
	{
		buf.Reset()
		fmt.Fprintf(buf, "%d.%d.%d", major, minor, patch)
		newVersion = append(newVersion, buf.Bytes()...)
	}
	{
		incVersion()
		buf.Reset()
		fmt.Fprintf(buf, "%d.%d.%d", major, minor, patch)
		buf.WriteString(PreviewSuffix)
		newPreviewVersion = append(newPreviewVersion, buf.Bytes()...)
	}

	var writeNewContent = func(version []byte) error {
		buf.Reset()
		buf.Write(oldContent[:m])
		buf.Write(version)
		buf.Write(oldContent[n:])
		return os.WriteFile(versionGoFile, buf.Bytes(), 0644)
	}

	if err := writeNewContent(newVersion); err != nil {
		log.Printf("write release version file error: %s", err)
		return
	}

	var gitTag = fmt.Sprintf("v%s", newVersion)
	if output, err := util.RunShellCommand(time.Second*5, "", nil, nil, nil, nil,
		"git", "commit", "-a", "-m", gitTag); err != nil {
		log.Printf("git commit error: %s\n%s", err, output)
	}
	if output, err := util.RunShellCommand(time.Second*5, "", nil, nil, nil, nil,
		"git", "tag", gitTag); err != nil {
		log.Printf("git commit error: %s\n%s", err, output)
	}

	if err := writeNewContent(newPreviewVersion); err != nil {
		log.Printf("write preview version file error: %s", err)
		return
	}

	log.Printf("new version: %s", newVersion)
	log.Printf("new preview version: %s", newPreviewVersion)
}
