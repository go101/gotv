package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go101.org/gotv/internal/util"
)

func (gotv *gotv) tryRunningGoToolchainCommand(tv toolchainVersion, args []string) error {
	if _, err := gotv.ensureToolchainVersion(&tv, false); err != nil {
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

	panic("unreachable. tv: " + tv.String())
}

func (gotv *gotv) ensureToolchainVersion(tv *toolchainVersion, forPinning bool) (_ string, err error) {

	if _, err := gotv.ensureGoRepository(tv.questioned); err != nil {
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

	var bootstrapRoot = ""
	if bootstrapRoot = os.Getenv("GOROOT_BOOTSTRAP"); bootstrapRoot == "" {
		var bootstrapTV = determineBootstrapToolchainVersion(*tv)
		if bootstrapTV == nil {
			return "", errors.New("unable to build toolchain " + tv.String())
		} else if bootstrapTV.kind != kind_Invalid {
			bootstrapRoot, err = gotv.ensureToolchainVersion(bootstrapTV, false)
			if err != nil {
				return "", err
			}
		} else if runtime.GOOS == "windows" {
			// It looks "make.bat" is unable to determine GOROOT_BOOTSTRAP,
			// but "make.bash" is able to.
			goExePath, err := exec.LookPath("go")
			if err != nil {
				fmt.Println(err)
			}
			bootstrapRoot = filepath.Dir(filepath.Dir(goExePath))
		}
	}

	var goCommandFilename string
	if runtime.GOOS == "windows" {
		goCommandFilename = "go.exe"
	} else {
		goCommandFilename = "go"
	}

	var goCommandPath, toolchainDir string
	if forPinning {
		toolchainDir = gotv.pinnedToolchainDir
		goCommandPath = filepath.Join(toolchainDir, "bin", goCommandFilename)

		// toolchainDir will be modify later.
		var realToolchainDir = toolchainDir
		defer func() {
			if err != nil || realToolchainDir == toolchainDir {
				return
			}

			err = os.Rename(toolchainDir, realToolchainDir)
		}()
	} else {
		goCommandPath = filepath.Join(gotv.cacheDir, tv.folderName(), "bin", goCommandFilename)
		toolchainDir = filepath.Dir(filepath.Dir(goCommandPath))

		defer func() {
			if err == nil {
				gotv.versionGoCmdPaths[*tv] = goCommandPath
			}
		}()
	}

	const gotvInfoFile = "gotv.info"
	type InfoFile struct {
		Revision string `json:"revision"`
	}
	var infoFilePath = filepath.Join(toolchainDir, gotvInfoFile)

	var revision = gotv.toolchainVersion2Revision(*tv)

	if _, err := os.Stat(infoFilePath); err != nil {
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

	if forPinning {
		toolchainDir = gotv.pinnedToolchainDir + "_temp"
		goCommandPath = filepath.Join(toolchainDir, "bin", goCommandFilename)
		infoFilePath = filepath.Join(toolchainDir, gotvInfoFile)
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
	fmt.Println("[Run]:", gotv.replaceHomeDir(makeScript))

	var envs []string
	if runtime.GOOS == "windows" {
		if bootstrapRoot != "" {
			envs = []string{"CGO_ENABLED=0", "GOROOT_BOOTSTRAP=" + bootstrapRoot}
		} else {
			envs = []string{"CGO_ENABLED=0"}
		}
	} else if bootstrapRoot != "" {
		envs = []string{"GOROOT_BOOTSTRAP=" + bootstrapRoot}
	}

	if _, err := util.RunShell(time.Hour, toolchainSrcDir, envs, os.Stdout, os.Stdout, makeScript); err != nil {
		return "", err
	}

	if _, err := os.Stat(goCommandPath); err != nil {
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
	goCommandPath, ok := gotv.versionGoCmdPaths[tv]
	if !ok {
		panic("toochain version " + tv.String() + " is not built?")
	}
	if _, err := os.Stat(goCommandPath); err != nil {
		return err
	}

	fmt.Print("[Run]: ", gotv.replaceHomeDir(goCommandPath))
	for _, a := range args {
		fmt.Print(" ", a)
	}
	fmt.Println()
	_, err := util.RunShellCommand(time.Hour, "", nil, os.Stdout, os.Stderr, goCommandPath, args...)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok { // always okay
			os.Exit(ee.ExitCode())
		}
	}

	return err // must be nil
}
