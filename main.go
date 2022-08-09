package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go101.org/gotv/internal/util"
)

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

	tv := parseGoToolchainVersion(args[0])
	if invalid, message := tv.IsInvalid(); invalid {
		fmt.Fprintln(os.Stderr, message)
		os.Exit(1)
	}
	if err := gotv.tryRunningGoToolchainCommand(tv, args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printUsage(program string) {
	fmt.Printf(`GoTV %s

Usage (to use a specific Go toolchain version):
	%s ToolchainVersion [go-arguments...]

	where ToolchainVersion might be
	* a Go release version, such as 1.17.13, 1.18,
	  and 1.19rc1, which mean the release tags
	  go1.17.13,  go1.18, go1.19rc1, respectively,
	  in the Go git repository.
	* :tip, which means the local latest master
	  branch in the Go git repository.
	* :N.M, such as 1.17, 1.18 and 1.19, which mean
	  the local latest release-branch.goN.M branch
	  in the Go git repository.

GoTV specific commands:
	gotv fetch-versions
	gotv list-versions
	gotv cache-version ToolchainVersion [ToolchainVersion ...]
	gotv uncache-version ToolchainVersion [ToolchainVersion ...]
	gotv pin-version ToolchainVersion
	gotv unpin-version
`,
		Version,
		filepath.Base(program),
	)
}

const Version = "v0.0.2"

func releaseGoTV() {
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, "go", "test", "./..."); err != nil {
		log.Println(err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, "go", "fmt", "./..."); err != nil {
		log.Println(err)
		return
	}
	if _, err := util.RunShell(time.Minute*3, "", nil, nil, "go", "mod", "tidy"); err != nil {
		log.Println(err)
		return
	}

	const (
		VersionConstPrefix = `const Version = "v`
		PreviewSuffix      = "-preview"
	)

	var verisonGoFile = "main.go"

	oldContent, err := ioutil.ReadFile(verisonGoFile)
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
		return ioutil.WriteFile(verisonGoFile, buf.Bytes(), 0644)
	}

	if err := writeNewContent(newVersion); err != nil {
		log.Printf("write release version file error: %s", err)
		return
	}

	var gitTag = fmt.Sprintf("v%s", newVersion)
	if output, err := util.RunShellCommand(time.Second*5, "", nil, nil,
		"git", "commit", "-a", "-m", gitTag); err != nil {
		log.Printf("git commit error: %s\n%s", err, output)
	}
	if output, err := util.RunShellCommand(time.Second*5, "", nil, nil,
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
