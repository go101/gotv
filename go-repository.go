package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	//"time"

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
		// ToDo: verify the local repository is okay.
		// if okay, return, otherwise, delete the dir and coninue

		okay := true
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

Specify it here: `)

	var repoAddr string
	_, err = fmt.Scanln(&repoAddr)
	if err != nil {
		return err
	}

	// ToDo: if the "git@github.com:golang/go.git" url is chosen, need config auth

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

func sortVersions(versions []string) {
	var indexNonDigits = func(str string) int {
		for i, b := range str {
			if b < '0' || b > '9' {
				return i
			}
		}
		return len(str)
	}

	sort.Slice(versions, func(a, b int) bool {
		x, y := versions[a], versions[b]
		xs, ys := strings.SplitN(x, ".", -1), strings.SplitN(y, ".", -1)
		var true, false = true, false
		if len(xs) > len(ys) {
			xs, ys = ys, xs
			true, false = false, true
		}
		for i, r := range xs {
			s := ys[i]
			rk := indexNonDigits(r)
			sk := indexNonDigits(s)
			rn, _ := strconv.Atoi(r[:rk])
			sn, _ := strconv.Atoi(s[:sk])
			if rn < sn {
				return true
			}
			if rn > sn {
				return false
			}
			if len(r) == rk {
				if len(s) == sk {
					continue
				}
				return false
			} else if len(s) == sk {
				return true
			} else {
				return r[rk:] <= s[sk:]
			}
		}

		return true
	})
}
