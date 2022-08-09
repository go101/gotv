package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	cp "github.com/otiai10/copy"
	//"go101.org/gotv/internal/util"
)

func (gotv *gotv) ensureGoRepository(pullOnExist bool) (err error) {
	_, err = os.Stat(gotv.repositoryDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	} else {
		var okay = true
		_, err = gitWorktree(gotv.repositoryDir)
		if err != nil {
			okay = false
		}

		if okay {
			if pullOnExist {
				err = gitPull(gotv.repositoryDir)
			}

			return
		} else {
			err = os.RemoveAll(gotv.repositoryDir)
			if err != nil {
				return err
			}
		}
	}

	// clone it

	defer func() {
		if err != nil {
			os.RemoveAll(gotv.repositoryDir)
		}
	}()

	err = os.MkdirAll(gotv.cacheDir, 0700)
	if err != nil {
		return err
	}

	fmt.Print(`Please specify the Go project repositry git address.
Generally, it should be one of the following ones:
* https://go.googlesource.com/go
* https://github.com/golang/go.git
* git@github.com:golang/go.git

Specify it here: `)

	var repoAddr string
	_, err = fmt.Scanln(&repoAddr)
	if err != nil {
		return err
	}

	fmt.Println("[Run]: git clone", gotv.replaceHomeDir(repoAddr), gotv.replaceHomeDir(gotv.repositoryDir))
	err = gitClone(repoAddr, gotv.repositoryDir)
	if err != nil {
		return err
	}

	return nil
}

func (gotv *gotv) copyBranchShallowly(tv toolchainVersion, toDir string) error {
	var repoDir = gotv.repositoryDir

	switch tv.kind {
	case kind_Tag, kind_Branch, kind_Revision:
		fmt.Println("[Run]: cp -r", gotv.replaceHomeDir(repoDir), gotv.replaceHomeDir(toDir))
		var err = cp.Copy(repoDir, toDir)
		if err != nil {
			return err
		}

		fmt.Println("[Run]: cd", gotv.replaceHomeDir(toDir))

		var o = git.CheckoutOptions{Force: true, Keep: false}
		if tv.kind == kind_Revision {
			o.Hash = plumbing.NewHash(tv.version)
		} else if tv.kind == kind_Tag {
			o.Branch = plumbing.NewTagReferenceName(tv.version)
		} else { // tv.kind == kind_Branch
			o.Branch = plumbing.NewRemoteReferenceName("origin", tv.version)
			// o.Create = true // ToDo: not work. how to create local branches?
		}

		fmt.Println("[Run]: git checkout", tv.version)
		err = gitCheckout(toDir, &o)
		if err != nil {
			return err
		}
	default:
		panic("unsupoorted version kinds")
	}

	return nil
}

var (
	releaseTagRegexp    = regexp.MustCompile(`^go([1-9]([0-9]*).*)`)
	releaseBranchRegexp = regexp.MustCompile(`^(release-branch.go([1-9]([0-9]*).*))`)
)

func collectRepositoryInfo(repoDir string) (repoInfo repoInfo, err error) {
	tags, branches, err := gitListTagsAndRemoteBranches(repoDir)
	if err != nil {
		return
	}

	repoInfo.allTags = tags
	repoInfo.releaseTags = make(map[string]string, len(tags))
	for t := range tags {
		if ms := releaseTagRegexp.FindAllStringSubmatch(t, 1); len(ms) > 0 {
			repoInfo.releaseTags[ms[0][1]] = t

			// ToDo: find latest for version 1, and 1.n. (Make a tree structure?)
		}
	}

	repoInfo.allBranches = branches
	repoInfo.versionBranches = make(map[string]string, len(branches))
	for b := range branches {
		if ms := releaseBranchRegexp.FindAllStringSubmatch(b, 1); len(ms) > 0 {
			repoInfo.versionBranches[ms[0][2]] = ms[0][1]
		}
	}

	return
}
