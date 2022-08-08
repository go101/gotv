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

	versionBinDirs map[toolchainVersion]string
}

type repoInfo struct {
	releaseTags     map[string]string // simplified name to full tag name
	allTags         map[string]string // tag name to hash hex
	versionBranches map[string]string // simplified name to full branch name
	allBranches     map[string]string // branch name to head hash hex
}

func born() (gotv gotv, err error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	gotv.homeDir, _ = os.UserHomeDir()

	gotv.repositoryDir = filepath.Join(cacheDir, "gotv", "the-repository")
	gotv.cacheDir = filepath.Dir(gotv.repositoryDir)
	gotv.pinnedToolchainDir = filepath.Join(gotv.cacheDir, "pinned-toolchain")

	gotv.versionBinDirs = make(map[toolchainVersion]string, 64)

	return
}

func (gotv *gotv) replaceHomeDir(in string) string {
	if gotv.homeDir == "" {
		return in
	}
	if strings.HasPrefix(in, gotv.homeDir) {
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
