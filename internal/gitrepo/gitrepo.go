package gitrepo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5" // go-git library
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitClient wraps a Git repository.
type GitClient struct {
	RepoPath   string
	Repository *git.Repository
}

// NewGitClient opens the Git repository located at repoPath.
func NewGitClient(repoPath string) (*GitClient, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %v", err)
	}
	return &GitClient{
		RepoPath:   repoPath,
		Repository: repo,
	}, nil
}

// CommitChanges stages all changes and commits with the given message and author info.
func (gc *GitClient) CommitChanges(commitMessage, authorName, authorEmail string) error {
	worktree, err := gc.Repository.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	// Stage all changes.
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to add changes: %v", err)
	}

	// Create a commit.
	commitHash, err := worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	fmt.Println("Commit successful:", commitHash.String())
	return nil
}

// PushChanges pushes the commits to the remote repository using basic authentication.
func (gc *GitClient) PushChanges(username, password string) error {
	if err := gc.Repository.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username, // typically "git" for GitHub when using a token
			Password: password,
		},
	}); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}
	fmt.Println("Push successful")
	return nil
}

// ReadAllFiles traverses the repository's directory (skipping the .git folder)
// and returns a map of relative file paths to their contents.
func (gc *GitClient) ReadAllFiles() (map[string]string, error) {
	filesContent := make(map[string]string)
	err := filepath.Walk(gc.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the .git directory.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(gc.RepoPath, path)
			if err != nil {
				return err
			}
			filesContent[relPath] = string(data)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return filesContent, nil
}
