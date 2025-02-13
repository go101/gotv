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
func (gotv *gotv) normalizeToolchainVersion(tv *toolchainVersion, dontChangeKind bool) error {
	if tv.kind == kind_Tag || tv.kind == kind_Branch || tv.kind == kind_Revision {
		return nil
	}

	if tv.kind == kind_Release {
		var checkLatest = strings.HasSuffix(tv.version, ".")
		var prefix string
		if !checkLatest {
			checkLatest = tv.version >= "1.21" && strings.Count(tv.version, ".") == 1
			prefix = tv.version
		} else {
			prefix = tv.version[:len(tv.version)-1]
		}

		if checkLatest {
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

		if dontChangeKind {
			// For setDefaultVersion
			return nil
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
	if _, err := gotv.ensureGoRepository(tv.forceSyncRepo); err != nil {
		return "", err
	}

	if repoInfo, err := collectRepositoryInfo(gotv.repositoryDir); err != nil {
		return "", err
	} else {
		gotv.repoInfo = repoInfo
	}

	if err := gotv.normalizeToolchainVersion(tv, false); err != nil {
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
				return "", err
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
			// ToDo: realToolchainDir == toolchainDir is always false?
			if err != nil || realToolchainDir == toolchainDir {
				return
			}

			err = os.RemoveAll(realToolchainDir)
			if err != nil {
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

	buildEnvs := func() []string {
		var envs []string
		//if runtime.GOOS == "windows" {
		if bootstrapRoot != "" {
			envs = []string{"CGO_ENABLED=0", "GOROOT_BOOTSTRAP=" + bootstrapRoot}
		} else {
			envs = []string{"CGO_ENABLED=0"}
		}

		return envs
	}

	if _, err := util.RunShell(time.Hour, toolchainSrcDir, buildEnvs, nil, os.Stdout, nil, makeScript); err != nil {
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

	// change PATH env var
	{
		goBinPath := filepath.Dir(goCommandPath)
		oldpath, newpath := os.Getenv("PATH"), ""
		if oldpath == "" {
			newpath = goBinPath
		} else {
			newpath = goBinPath + string(os.PathListSeparator) + oldpath
		}
		os.Setenv("PATH", newpath)
		defer func() {
			os.Setenv("PATH", oldpath)
		}()
	}
	buildEnv := func() []string {
		return []string{
			// https://github.com/golang/go/issues/57001
			"GOTOOLCHAIN=local",
		}
	}
	_, err := util.RunShellCommand(time.Hour, "", buildEnv, os.Stdin, os.Stdout, os.Stderr, goCommandPath, args...)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok { // always okay
			os.Exit(ee.ExitCode())
		}
	}

	return err // must be nil
}
