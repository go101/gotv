package main

import (
	"os"
	"path/filepath"
	"strings"
)

type gotv struct {
	homeDir  string
	cacheDir string

	repositoryDir string

	pinnedToolchainDir string

	repoInfo repoInfo

	versionGoCmdPaths map[toolchainVersion]string
}

type repoInfo struct {
	releaseTags     map[string]string // simplified name to full tag name
	allTags         map[string]string // tag name to hash hex
	versionBranches map[string]string // simplified name to full branch name
	allBranches     map[string]string // branch name to head hash hex
	tipHash         string
}

func born() (_ gotv, err error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	return bornWithCacheDir(cacheDir)
}

func bornWithCacheDir(cacheDir string) (gotv gotv, err error) {
	gotv.homeDir, _ = os.UserHomeDir() // used to read user ssh key

	gotv.repositoryDir = filepath.Join(cacheDir, "gotv", "the-repository")
	gotv.cacheDir = filepath.Dir(gotv.repositoryDir)
	gotv.pinnedToolchainDir = filepath.Join(gotv.cacheDir, "pinned-toolchain")

	gotv.versionGoCmdPaths = make(map[toolchainVersion]string, 128)

	return
}

func (gotv *gotv) replaceHomeDir(in string) string {
	if gotv.homeDir == "" {
		return in
	}
	if len(in) == len(gotv.homeDir) {
		return in
	}
	if strings.HasPrefix(in, gotv.homeDir) && in[len(gotv.homeDir)] == filepath.Separator {
		return "$HOME" + in[len(gotv.homeDir):]
	}

	return in
}

func (gotv *gotv) toolchainVersion2Revision(tv toolchainVersion) string {
	switch tv.kind {
	case kind_Tag:
		var rev, ok = gotv.repoInfo.allTags[tv.version]
		if !ok {
			panic("tag not found: " + tv.version)
		}
		return rev
	case kind_Branch:
		var rev, ok = gotv.repoInfo.allBranches[tv.version]
		if !ok {
			panic("branch not found: " + tv.version)
		}
		return rev
	case kind_Revision:
		return tv.version
	}

	panic("unreachable")
}
