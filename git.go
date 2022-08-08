package main

import (
	"encoding/hex"
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	//gitobject "github.com/go-git/go-git/v5/plumbing/object"
	//gitconfig "github.com/go-git/go-git/v5/config"
)

func gitClone(repoAddr, toDir string) error {
	//cmdAndArgs := []string{"git", "clone", repoAddr, toDir}
	//_, err := util.RunShell(time.Hour, "", nil, os.Stdout, cmdAndArgs...)

	// ToDo: make "git@github.com:golang/go.git" work.
	var _, err = git.PlainClone(toDir, false,
		&git.CloneOptions{
			URL:      repoAddr,
			Progress: os.Stdout,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func gitPull(repoDir string) error {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	var o = git.PullOptions{
		Force: true,
	}
	return worktree.Pull(&o)
}

func gitFetch(repoDir string) error {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return err
	}

	var o = git.FetchOptions{
		Force: true,
	}
	return repo.Fetch(&o)
}

func gitCheckout(repoDir string, opt *git.CheckoutOptions) error {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	return worktree.Checkout(opt)
}

func gitListTagsAndRemoteBranches(repoDir string) (tags map[string]string, bras map[string]string, err error) {
	repo, err := git.PlainOpen(repoDir)
	if err != nil {
		return
	}

	iter, err := repo.References()
	if err != nil {
		return
	}

	const TagRefPrefix = "refs/tags/"
	const BranchRefPrefix = "refs/remotes/origin/"

	tags = make(map[string]string, 1024)
	bras = make(map[string]string, 128)
	iter.ForEach(func(ref *plumbing.Reference) error {
		switch name := string(ref.Name()); {
		case strings.HasPrefix(name, TagRefPrefix):
			var hash = ref.Hash()
			tags[name[len(TagRefPrefix):]] = hex.EncodeToString(hash[:])
		case strings.HasPrefix(name, BranchRefPrefix):
			var hash = ref.Hash()
			bras[name[len(BranchRefPrefix):]] = hex.EncodeToString(hash[:])
		default:
			//fmt.Println(ref.String())
		}
		return nil
	})

	return
}

type versionKind int

// Note: git allows a branch name collides with a revision hash or a tag name.
//       Doing so will cause ambiguousness in "git checkout such-a-branch-name".

const (
	kind_Invalid versionKind = iota // must be 0

	kind_Tag
	kind_Branch
	kind_Revision

	// 1         (<=> name:go1)
	// 1.16.3    (<=> name:go1.16.3)
	// 1.17      (<=> name:go1.17)
	// 1.18rc2   (<=> name:go1.18rc2)
	// 1.19beta1 (<=> name:go1.19beta1)
	kind_Release
	// ToDo:
	// 1.        (<=> name:go1.Latest.Latest)
	// 1.18.     (<=> name:go1.18.Latest)

	// :tip  (<=> name:master)
	// :1.18 (<=> name:release-branch.go1.18)
	kind_Alias
)

type toolchainVersion struct {
	kind    versionKind
	version string
}

func (tv toolchainVersion) IsInvalid() (bool, string) {
	if tv.kind == kind_Invalid {
		return true, tv.version
	}
	return false, ""
}

func (tv toolchainVersion) String() string {
	switch tv.kind {
	case kind_Tag:
		return "tag:" + tv.version
	case kind_Branch:
		return "bra:" + tv.version
	case kind_Revision:
		return "rev:" + tv.version
	case kind_Release:
		return "" + tv.version
	case kind_Alias:
		return ":" + tv.version
	}

	return "" // invalid
}

func (tv toolchainVersion) folderName() string {
	var folder string
	switch tv.kind {
	default:
		panic("bad version: " + tv.String())
	case kind_Tag:
		folder = "tag_" + tv.version
	case kind_Branch:
		folder = "bra_" + tv.version
	case kind_Revision:
		folder = "rev_" + tv.version
	}

	return folder
}

// The second result means "consumed".
func parseGoToolchainVersion(arg string) toolchainVersion {
	arg = strings.TrimSpace(arg)
	if len(arg) == 0 {
		return toolchainVersion{kind_Invalid, "version is unspecified"}
	}

	if c := arg[0]; '0' <= c && c <= '9' {
		return toolchainVersion{kind_Release, trimTaillingDotZeros(arg)}
	}

	i := strings.IndexByte(arg, ':')
	if i < 0 {
		return toolchainVersion{kind_Invalid, "invalid version: " + arg}
	}

	kind, version := arg[:i], arg[i+1:]

	if len(version) == 0 {
		return toolchainVersion{kind_Invalid, "unspecified version for kind (" + kind + ")"}
	}

	switch kind {
	default:
		return toolchainVersion{kind_Invalid, "undetermined version kind: " + kind}
	case "tag":
		return toolchainVersion{kind_Tag, version}
	case "bra":
		return toolchainVersion{kind_Branch, version}
	case "rev":
		return toolchainVersion{kind_Revision, version}
	case "": // alias verisons
	}

	// validate alias names (roughly)

	if version != "tip" {
		if c := version[0]; c < '0' || c > '9' {
			return toolchainVersion{kind_Invalid, "an alias version must be tip or a go version"}
		}
		version = trimTaillingDotZeros(version)
	}

	return toolchainVersion{kind_Alias, version}
}

func trimTaillingDotZeros(version string) string {
	for strings.HasSuffix(version, ".0") {
		version = version[:len(version)-2]
	}
	return version
}
