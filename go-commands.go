package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go101.org/gotv/internal/util"
)

func (gotv *gotv) tryRunningGoToolchainCommand(tv toolchainVersion, args []string) error {
	if _, err := gotv.ensureToolchainVersion(&tv); err != nil {
		return err
	}

	return gotv.runGoToolchainCommand(tv, args)
}

// After normalization, tv.kind may be only tag/brranch/revision
func (gotv *gotv) normalizeToolchainVersion(tv *toolchainVersion) error {
	if tv.kind == kind_Tag || tv.kind == kind_Branch || tv.kind == kind_Revision {
		return nil
	}

	if tv.kind == kind_Release {
		if strings.HasSuffix(tv.version, ".") {
			var prefix = tv.version[:len(tv.version)-1]
			var latest = ""
			for tag := range gotv.repoInfo.releaseTags {
				if strings.HasPrefix(tag, prefix) {
					if compareVersions(latest, tag) {
						latest = tag
					}
				}
			}
			if latest == "" {
				return fmt.Errorf("not latest version found for fake verison: %s", tv.version)
			}

			tv.version = latest
		}

		if v := gotv.repoInfo.releaseTags[tv.version]; v == "" {
			return fmt.Errorf("release version %s not found", tv.version)
		} else {
			tv.version = v
			tv.kind = kind_Tag
			return nil
		}
	}

	if tv.kind == kind_Alias {
		if tv.version == "tip" {
			tv.version = "master"
		} else if v := gotv.repoInfo.versionBranches[tv.version]; v != "" {
			tv.version = v
		} else {
			return fmt.Errorf("release branch %s not found", tv.version)
		}

		tv.kind = kind_Branch
		return nil
	}

	panic("unreachable")
}

func (gotv *gotv) ensureToolchainVersion(tv *toolchainVersion) (_ string, err error) {
	if err := gotv.ensureGoRepository(false); err != nil {
		return "", err
	}

	if repoInfo, err := collectRepositoryInfo(gotv.repositoryDir); err != nil {
		return "", err
	} else {
		gotv.repoInfo = repoInfo
	}

	if err := gotv.normalizeToolchainVersion(tv); err != nil {
		return "", err
	}

	var folder = tv.folderName()

	toochainBinDir := filepath.Join(gotv.cacheDir, folder, "bin")
	toolchainDir := filepath.Dir(toochainBinDir)
	defer func() {
		if err == nil {
			gotv.versionBinDirs[*tv] = toochainBinDir
		}
	}()

	const gotvInfoFile = "gotv.info"

	type InfoFile struct {
		Revision string `json:"revision"`
	}

	var infoFilePath = filepath.Join(toolchainDir, gotvInfoFile)
	var revision = gotv.toolchainVersion2Revision(*tv)

	if _, err := os.Stat(toochainBinDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	} else {
		var outdated = true
		var info, err = os.ReadFile(infoFilePath)
		if err == nil {
			var file InfoFile
			err = json.Unmarshal(info, &file)
			if err == nil {
				outdated = file.Revision != revision
			}
		}

		if !outdated {
			return toolchainDir, nil
		}
	}

	if err := os.RemoveAll(toolchainDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(toolchainDir)
		}
	}()

	if err := gotv.copyBranchShallowly(*tv, toolchainDir); err != nil {
		return "", err
	}

	defer func() {
		if err == nil {
			os.RemoveAll(filepath.Join(toolchainDir, ".git"))
			os.Remove(filepath.Join(toolchainDir, ".gitignore"))
			os.RemoveAll(filepath.Join(toolchainDir, ".github"))
		}
	}()

	var makeScript string
	if runtime.GOOS == "windows" {
		makeScript = filepath.Join(toolchainDir, "src", "make.bat")
	} else {
		makeScript = filepath.Join(toolchainDir, "src", "make.bash")
	}
	var toolchainSrcDir = filepath.Dir(makeScript)
	if _, err := util.RunShell(time.Hour, toolchainSrcDir, nil, os.Stdout, makeScript); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `{"revision": "%s"}`, revision)
	if err := os.WriteFile(infoFilePath, buf.Bytes(), 0644); err != nil {
		return "", err
	}

	return toolchainDir, nil
}

func (gotv *gotv) runGoToolchainCommand(tv toolchainVersion, args []string) error {
	toolchainBinDir, ok := gotv.versionBinDirs[tv]
	if !ok {
		panic("toochain version " + tv.String() + " is not built?")
	}

	goCommandPath := filepath.Join(toolchainBinDir, "go") // ToDo: other OSes

	fmt.Print("[Run]: ", gotv.replaceHomeDir(goCommandPath))
	for _, a := range args {
		fmt.Print(" ", a)
	}
	fmt.Println()
	_, err := util.RunShellCommand(time.Hour, "", nil, os.Stdout, goCommandPath, args...)
	if err != nil {
		return err
	}

	return nil
}
