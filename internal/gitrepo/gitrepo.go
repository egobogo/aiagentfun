package gitrepo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"                         // go-git library
	"github.com/go-git/go-git/v5/plumbing/object"         // for commit signatures
	"github.com/go-git/go-git/v5/plumbing/transport/http" // for basic auth
)

// GitClient defines basic Git operations.
type GitClient struct {
	RepoURL  string
	RepoPath string
	Repo     *git.Repository
}

// RepoFile represents a single file within the repository in JSON form.
type RepoFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// RepoSnapshot is the top-level JSON structure.
type RepoSnapshot struct {
	Files []RepoFile `json:"files"`
}

// NewGitClient creates a new GitClient.
// If the repository does not exist at repoPath, it clones from repoURL; otherwise, it opens the existing repo.
func NewGitClient(repoURL, repoPath string) (*GitClient, error) {
	var repo *git.Repository
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		// Clone repository if it doesn't exist.
		repo, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL: repoURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Open existing repository.
		var err error
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository: %w", err)
		}
	}
	return &GitClient{
		RepoURL:  repoURL,
		RepoPath: repoPath,
		Repo:     repo,
	}, nil
}

// WriteFile writes content to a file relative to the repository path.
func (g *GitClient) WriteFile(fileName string, content []byte) error {
	fullPath := filepath.Join(g.RepoPath, fileName)
	return os.WriteFile(fullPath, content, 0644)
}

// CommitChanges stages all changes in the repository and commits them with the provided commit message and author info.
func (g *GitClient) CommitChanges(commitMessage, authorName, authorEmail string) error {
	worktree, err := g.Repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Stage all changes.
	if err := worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Create a commit.
	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// PushChanges pushes commits to the remote repository using basic authentication.
func (g *GitClient) PushChanges(username, token string) error {
	err := g.Repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username, // For GitHub, this is usually "git" when using a token.
			Password: token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

// GatherRepoInfo walks the repository path and gathers code file information.
// It returns a JSON string of the repository snapshot, a schema describing its structure, and an error.
func (g *GitClient) GatherRepoInfo() (string, interface{}, error) {
	// Define types for our repo snapshot.
	type RepoFile struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	type RepoSnapshot struct {
		Files []RepoFile `json:"files"`
	}

	snapshot := RepoSnapshot{}

	// Walk the repository folder.
	err := filepath.Walk(g.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip .git folder.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		// Filter: only process code files.
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".go") ||
			strings.HasSuffix(info.Name(), ".py") ||
			strings.HasSuffix(info.Name(), ".js") ||
			strings.HasSuffix(info.Name(), ".ts") ||
			strings.HasSuffix(info.Name(), ".java") ||
			strings.HasSuffix(info.Name(), ".rb") ||
			strings.HasSuffix(info.Name(), ".cs") ||
			strings.HasSuffix(info.Name(), ".cpp") ||
			strings.HasSuffix(info.Name(), ".c")) {
			relativePath, _ := filepath.Rel(g.RepoPath, path)
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", relativePath, err)
			}
			snapshot.Files = append(snapshot.Files, RepoFile{
				Path:    relativePath,
				Content: string(content),
			})
		}
		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("error walking repo path: %w", err)
	}

	// Marshal the snapshot into a formatted JSON string.
	repoJSONBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal repo snapshot: %w", err)
	}

	// Define the schema describing the structure of the repo JSON.
	schema := map[string]interface{}{
		"files": []map[string]string{
			{
				"path":    "string",
				"content": "string",
			},
		},
	}

	return string(repoJSONBytes), schema, nil
}
