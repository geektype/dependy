package main

import (
	"fmt"
	"time"

	"github.com/geektype/dependy/domain"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

func NewGitManager(config GitConfig) *GitManager {
	fs := memfs.New()
	storer := memory.NewStorage()
	return &GitManager{
		FileSystem: fs,
		Storer:     storer,
		Username:   "dependy",
		Password:   "glpat-QNhnWWdJwYFwAevZRZxf",
		Config:     config,
	}
}

type GitAuthor struct {
	Name  string
	Email string
}

type GitConfig struct {
	PatchBranchPrefix string
	CommitTitlePrefix string
	Author            GitAuthor
}

type GitManager struct {
	FileSystem billy.Filesystem
	Storer     *memory.Storage
	Username   string
	Password   string
	MainBranch string
	Repository *git.Repository
	WorkTree   *git.Worktree
	Config     GitConfig
}

func (g *GitManager) CloneRepo(repo domain.Repository) error {
	r, err := git.Clone(g.Storer, g.FileSystem, &git.CloneOptions{
		URL: repo.Url,
		Auth: &http.BasicAuth{
			Username: g.Username,
			Password: g.Password,
		},
	})
	if err != nil {
		return err
	}

	g.Repository = r
	g.WorkTree, err = r.Worktree()
	if err != nil {
		return err
	}
	g.MainBranch = repo.Branch

	return nil
}

func (g *GitManager) BranchMain() error {
	headRef, err := g.Repository.Head()
	if err != nil {
		return err
	}
	branchRefName := plumbing.NewBranchReferenceName(g.Config.PatchBranchPrefix)
	branchHashRef := plumbing.NewHashReference(branchRefName, headRef.Hash())
	g.Repository.Storer.SetReference(branchHashRef)

	// Checkout DependyBranch

	if err := g.WorkTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(branchRefName),
	}); err != nil {
		return err
	}

	return nil
}

func (g *GitManager) OpenFile(fileName string) ([]byte, error) {
	f, err := g.FileSystem.Open(fileName)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 50000)
	i, err := f.Read(buffer)
	if err != nil {
		return nil, err
	}

	return buffer[:i], nil
}

func (g *GitManager) OverwriteFile(filename string, content []byte) error {
	f, err := g.FileSystem.Create(filename)
	if err != nil {
		return err
	}

	_, err = f.Write(content)
	if err != nil {
		return nil
	}

	return nil
}

func (g *GitManager) CommitFile(filename string, content []byte) error {
	err := g.OverwriteFile(filename, content)
	if err != nil {
		return err
	}

	_, err = g.WorkTree.Add(filename)
	if err != nil {
		return err
	}

	commitOptions := &git.CommitOptions{
		Author: &object.Signature{
			Name:  g.Config.Author.Name,
			Email: g.Config.Author.Email,
			When:  time.Now(),
		},
	}
	commitMessage := fmt.Sprintf("%s Update dependencies", g.Config.CommitTitlePrefix)
	_, err = g.WorkTree.Commit(commitMessage, commitOptions)
	if err != nil {
		return err
	}
	return nil
}

func (g *GitManager) Push() error {
	return g.Repository.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: g.Username,
			Password: g.Password,
		},
	})
}
