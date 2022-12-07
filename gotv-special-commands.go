package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	//"go101.org/gotv/internal/util"
)

type unknownCommand struct{}

func (unknownCommand) Error() string {
	return "unknown command"
}

func (gotv *gotv) tryRunningSpecialCommand(args []string) error {
	command, args := args[0], args[1:]
	switch command {
	case "fetch-versions":
		if len(args) > 0 {
			return errors.New(`fetch-versions needs no arguments`)
		}
		return gotv.fetchVersions()
	case "list-versions":
		return gotv.listVersions(args...)
	case "cache-version":
		if len(args) == 0 {
			return errors.New(`cache-version needs at least one argument`)
		}
		return gotv.cacheVersion(args...)
	case "uncache-version":
		if len(args) == 0 {
			return errors.New(`uncache-version needs at least one argument`)
		}
		return gotv.uncacheVersion(args...)
	case "pin-version":
		if len(args) != 1 {
			return errors.New(`pin-version needs exact one argument`)
		}
		return gotv.pinVersion(args[0])
	case "unpin-version":
		if len(args) > 0 {
			return errors.New(`unpin-version needs no arguments`)
		}
		return gotv.unpinVersion()
	}

	return unknownCommand{}
}

func (gotv *gotv) fetchVersions() error {
	var err error
	var cloned bool
	if cloned, err = gotv.ensureGoRepository(false); err != nil {
		return err
	}

	var oldRepoInfo repoInfo
	if !cloned {
		oldRepoInfo, err = collectRepositoryInfo(gotv.repositoryDir)
		if err != nil {
			return err
		}

		fmt.Println("[Run]: git pull -a (in " + gotv.replaceHomeDir(gotv.repositoryDir) + ")")

		err = gitPull(gotv.repositoryDir)
		if err != nil {
			return err
		}
	}

	newRepoInfo, err := collectRepositoryInfo(gotv.repositoryDir)
	if err != nil {
		return err
	}

	var updatedVersionBranches = make([]string, 0, 32)
	var newVersionBranches = make([]string, 0, 8)
	var newReleaseTags = make([]string, 0, 32)
	for bra := range newRepoInfo.versionBranches {
		if fullName, ok := oldRepoInfo.versionBranches[bra]; ok {
			if oldRepoInfo.allBranches[fullName] != newRepoInfo.allBranches[fullName] {
				updatedVersionBranches = append(updatedVersionBranches, bra)
			}
		} else {
			newVersionBranches = append(newVersionBranches, bra)
		}
	}
	for tag := range newRepoInfo.releaseTags {
		if _, ok := oldRepoInfo.releaseTags[tag]; ok {
		} else {
			newReleaseTags = append(newReleaseTags, tag)
		}
	}
	var tipChanged = oldRepoInfo.tipHash != newRepoInfo.tipHash

	sortVersions(updatedVersionBranches)
	sortVersions(newVersionBranches)
	sortVersions(newReleaseTags)

	var needNewLine = false

	if len(updatedVersionBranches) > 0 {
		if needNewLine {
			fmt.Println()
		}
		fmt.Println("Updated version branches:")
		for _, bra := range updatedVersionBranches {
			fmt.Printf("\t%s\n", bra)
		}
		needNewLine = true
	}

	if len(newReleaseTags) == 0 && len(newVersionBranches) == 0 {
		if needNewLine {
			fmt.Println()
		}
		fmt.Println("No new releases and version branches are found.")
	}

	if len(newVersionBranches) > 0 {
		if needNewLine {
			fmt.Println()
		}
		fmt.Println("New version branches:")
		for _, bra := range newVersionBranches {
			fmt.Printf("\t%s\n", bra)
		}
		needNewLine = true
	}
	if len(newReleaseTags) > 0 {
		if needNewLine {
			fmt.Println()
		}
		fmt.Println("New releases:")
		for _, tag := range newReleaseTags {
			fmt.Printf("\t%s\n", tag)
		}
		needNewLine = true
	}

	if tipChanged {
		if needNewLine {
			fmt.Println()
		}
		fmt.Println("Tip changed.")
	}

	return nil
}

func (gotv *gotv) listVersions(args ...string) error {
	if _, err := gotv.ensureGoRepository(false); err != nil {
		return err
	}

	reposInfo, err := collectRepositoryInfo(gotv.repositoryDir)
	if err != nil {
		return err
	}

	var versionBranches = make([]string, 0, 8)
	var releaseTags = make([]string, 0, 32)
	for bra := range reposInfo.versionBranches {
		versionBranches = append(versionBranches, bra)
	}
	for tag := range reposInfo.releaseTags {
		releaseTags = append(releaseTags, tag)
	}

	if len(releaseTags) == 0 && len(versionBranches) == 0 {
		fmt.Println("No releases and version branches are found.")
	}

	sortVersions(versionBranches)
	sortVersions(releaseTags)

	if len(versionBranches) > 0 {
		fmt.Println("Version branches:")
		for _, bra := range versionBranches {
			fmt.Printf("\t%s\n", bra)
		}
	}
	fmt.Println()
	if len(releaseTags) > 0 {
		fmt.Println("Releases:")
		for _, tag := range releaseTags {
			fmt.Printf("\t%s\n", tag)
		}
	}

	// ToDo: labels: cached, pinned, outdated, ...

	return nil
}

func (gotv *gotv) cacheVersion(versions ...string) error {
	tvs, err := parseGoToolchainVersions(versions...)
	if err != nil {
		return err
	}

	for i := range tvs {
		if _, err := gotv.ensureToolchainVersion(&tvs[i], false); err != nil {
			return err
		}
	}

	return nil
}

func (gotv *gotv) uncacheVersion(versions ...string) error {
	_, err := gotv.ensureGoRepository(false)
	if err != nil {
		return err
	}

	gotv.repoInfo, err = collectRepositoryInfo(gotv.repositoryDir)

	tvs, err := parseGoToolchainVersions(versions...)
	if err != nil {
		return err
	}

	for i := range tvs {
		if err := gotv.normalizeToolchainVersion(&tvs[i]); err != nil {
			return err
		}

		var folder = tvs[i].folderName()
		var toolchainDir = filepath.Join(gotv.cacheDir, folder)
		if _, err := os.Stat(toolchainDir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Printf("Vesion %s is not cached.\n", &tvs[i])
				continue
			}
			return err
		}

		fmt.Print("[Run]: rm -rf", gotv.replaceHomeDir(toolchainDir))
		err := os.RemoveAll(toolchainDir)
		fmt.Print(err)
		if err == nil {
			fmt.Println()
			continue
		}

		//if errors.Is(err, fs.ErrNotExist) {
		//	fmt.Println(" (not found)")
		//}

		return err
	}

	return nil
}

func (gotv *gotv) pinVersion(version string) error {
	var tv = parseGoToolchainVersion(version)
	if invalid, message := tv.IsInvalid(); invalid {
		return errors.New(message)
	}

	var _, err = gotv.ensureToolchainVersion(&tv, true)
	if err != nil {
		return err
	}

	fmt.Printf(`Pinned %s at %s.

Please put the following shown pinned toolchain path in
your PATH environment variable to use go commands directly:

	%s

`, tv, gotv.replaceHomeDir(gotv.pinnedToolchainDir), filepath.Join(gotv.pinnedToolchainDir, "bin"))

	return nil
}

func (gotv *gotv) unpinVersion() error {
	if err := os.RemoveAll(gotv.pinnedToolchainDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return nil
}
