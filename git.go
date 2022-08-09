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

func gitClone(repoAddr, toDir string) error {
	var auth gittransport.AuthMethod
	if strings.HasPrefix(repoAddr, "git@github.com:") {
		var homeDir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
		sshKey, err := os.ReadFile(filepath.Join(homeDir, ".ssh", "id_rsa"))
		if err != nil {
			return err
		}
		signer, err := ssh.ParsePrivateKey([]byte(sshKey))
		if err != nil {
			if _, ok := err.(*ssh.PassphraseMissingError); !ok {
				return err
			}
			fmt.Print(`Passphase: `)
			var passphase string
			_, err = fmt.Scanln(&passphase)
			if err != nil {
				return err
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(sshKey), []byte(passphase))
			if err != nil {
				return err
			}
		}
		hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		}

		auth = &gitssh.PublicKeys{
			User:   "git",
			Signer: signer,
			HostKeyCallbackHelper: gitssh.HostKeyCallbackHelper{
				HostKeyCallback: hostKeyCallback,
			},
		}
	}

	var _, err = git.PlainClone(toDir, false,
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

func gitWorktree(repoDir string) (*git.Worktree, error) {
	var repo, err = git.PlainOpen(repoDir)
	if err != nil {
		return nil, err
	}
	return repo.Worktree()
}

func gitPull(repoDir string) error {
	worktree, err := gitWorktree(repoDir)
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
