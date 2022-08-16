package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gittransport "github.com/go-git/go-git/v5/plumbing/transport"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	//gitobject "github.com/go-git/go-git/v5/plumbing/object"
	//gitconfig "github.com/go-git/go-git/v5/config"
)

func gitAuth(repoAddr string) (gittransport.AuthMethod, error) {
	if strings.HasPrefix(repoAddr, "git@github.com:") {
		var homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		sshKey, err := os.ReadFile(filepath.Join(homeDir, ".ssh", "id_rsa"))
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey([]byte(sshKey))
		if err != nil {
			if _, ok := err.(*ssh.PassphraseMissingError); !ok {
				return nil, err
			}
			fmt.Print(`Passphase: `)
			var passphase string
			_, err = fmt.Scanln(&passphase)
			if err != nil {
				return nil, err
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(sshKey), []byte(passphase))
			if err != nil {
				return nil, err
			}
		}
		hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		}

		return &gitssh.PublicKeys{
			User:   "git",
			Signer: signer,
			HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
				HostKeyCallback: hostKeyCallback,
			},
		}, nil
	}

	return nil, nil
}

func gitClone(repoAddr, toDir string) error {
	var auth, err = gitAuth(repoAddr)
	if err != nil {
		return err
	}

	_, err = git.PlainClone(toDir, false,
		&git.CloneOptions{
			Auth:     auth,
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

	remote, err := repo.Remote("origin")
	if err != nil {
		return err
	}

	repoAddr := remote.Config().URLs[0]
	auth, err := gitAuth(repoAddr)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	var o = git.PullOptions{
		Auth:  auth,
		Force: true,
	}
	return worktree.Pull(&o)
}

func gitFetch(repoDir string) error {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return err
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return err
	}

	repoAddr := remote.Config().URLs[0]
	auth, err := gitAuth(repoAddr)
	if err != nil {
		return err
	}

	var o = git.FetchOptions{
		Auth:  auth,
		Force: true,
	}
	return repo.Fetch(&o)
}

func gitWorktree(repoDir string) (*git.Worktree, error) {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return nil, err
	}
	return repo.Worktree()
}

func gitCheckout(repoDir string, opt *git.CheckoutOptions) error {
	worktree, err := gitWorktree(repoDir)
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
