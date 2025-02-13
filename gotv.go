package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type gotv struct {
	homeDir string

	cacheDir           string
	repositoryDir      string
	pinnedToolchainDir string

	repoInfo          repoInfo
	versionGoCmdPaths map[toolchainVersion]string

	configDir      string
	configFilePath string
}

type repoInfo struct {
	releaseTags     map[string]string // simplified name to full tag name
	allTags         map[string]string // tag name to hash hex
	versionBranches map[string]string // simplified name to full branch name
	allBranches     map[string]string // branch name to head hash hex
	tipHash         string
}

type configFile struct {
	DefaultVersion string `json:"default-version"`
}

func born() (_ gotv, err error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	configDir, err := os.UserConfigDir()
	if err == nil {
		configDir, _ = filepath.Abs(configDir)
	}

	return bornWithCacheAndConfigDir(cacheDir, configDir)
}

func bornWithCacheAndConfigDir(cacheDir, configDir string) (gotv gotv, err error) {
	gotv.homeDir, _ = os.UserHomeDir() // used to read user ssh key

	gotv.repositoryDir = filepath.Join(cacheDir, "gotv", "the-repository")
	gotv.cacheDir = filepath.Dir(gotv.repositoryDir)
	gotv.pinnedToolchainDir = filepath.Join(gotv.cacheDir, "pinned-toolchain")

	gotv.versionGoCmdPaths = make(map[toolchainVersion]string, 128)

	if configDir != "" {
		gotv.configFilePath = filepath.Join(configDir, "gotv", "config.info")
		gotv.configDir = filepath.Dir(gotv.configFilePath)
	}

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

func (gotv *gotv) loadConfig() (config configFile, err error) {
	if gotv.configFilePath == "" {
		err = errors.New("Config path is undetermined.")
		return
	}

	data, err := os.ReadFile(gotv.configFilePath)
	if err != nil {
		err = nil
		return
	} else {
		data = bytes.TrimSpace(data)
		if len(data) == 0 {
			return
		}
		err = json.Unmarshal(data, &config)
	}
	return
}

func (gotv *gotv) DefaultVersion() (tv toolchainVersion) {
	var config, err = gotv.loadConfig()
	if err != nil {
		return
	}

	return parseGoToolchainVersion(config.DefaultVersion, true)
}

func (gotv *gotv) changeDefaultVersion(tv toolchainVersion) (err error) {
	config, err := gotv.loadConfig()
	if err != nil {
		return
	}

	config.DefaultVersion = tv.String()

	data, err := json.Marshal(&config)
	if err != nil {
		return
	}

	err = os.MkdirAll(gotv.configDir, 0700)
	if err != nil {
		return
	}

	err = os.WriteFile(gotv.configFilePath, data, 0644)
	return
}
