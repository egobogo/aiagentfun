// File: test/agent_git_push_pull_cleanup_test.go
package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egobogo/aiagents/internal/gitrepo"
	"github.com/joho/godotenv"
)

func TestAgentGitPushPullAndCleanup(t *testing.T) {
	// Load environment variables from the .env file in the project root.
	if err := godotenv.Load("../.env"); err != nil {
		t.Log("No .env file found; using system environment variables")
	}

	// Read required environment variables.
	repoPath := os.Getenv("GIT_REPO_PATH")
	repoURL := os.Getenv("GIT_REPO_URL")
	username := os.Getenv("GIT_USERNAME")
	token := os.Getenv("GIT_TOKEN")
	if repoPath == "" || repoURL == "" || username == "" || token == "" {
		t.Fatalf("Required environment variables (GIT_REPO_PATH, GIT_REPO_URL, GIT_USERNAME, GIT_TOKEN) not set; skipping test")
	}

	// Create a GitClient instance for your live repository.
	client, err := gitrepo.NewGitClient(repoURL, repoPath)
	if err != nil {
		t.Fatalf("NewGitClient failed: %v", err)
	}

	// First, pull remote changes to update the local repository.
	if err := client.PullChanges(username, token); err != nil {
		t.Logf("Initial PullChanges error (possibly already up-to-date): %v", err)
	}

	// Generate a unique file name and content.
	fileName := fmt.Sprintf("real_test_%d.txt", time.Now().UnixNano())
	uniqueContent := fmt.Sprintf("Test file created at %s", time.Now().Format(time.RFC3339))

	// Write the file using your wrapper.
	if err := client.WriteFile(fileName, []byte(uniqueContent)); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Commit the changes with a unique commit message.
	commitMsg := "Test commit " + time.Now().Format("20060102150405")
	if err := client.CommitChanges(commitMsg, username, "tester@example.com"); err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Push the commit to the remote repository.
	if err := client.PushChanges(username, token); err != nil {
		t.Fatalf("PushChanges failed: %v", err)
	}

	t.Logf("File %s created and pushed with commit message: %s", fileName, commitMsg)

	// Wait a short time to allow push propagation.
	time.Sleep(2 * time.Second)

	// Cleanup: Delete the test file.
	testFilePath := filepath.Join(repoPath, fileName)
	if err := os.Remove(testFilePath); err != nil {
		t.Fatalf("Failed to remove test file: %v", err)
	}

	// Commit the deletion.
	cleanupMsg := "Cleanup: remove test file " + fileName
	if err := client.CommitChanges(cleanupMsg, username, "tester@example.com"); err != nil {
		t.Fatalf("CommitChanges for cleanup failed: %v", err)
	}

	// Push the cleanup commit.
	if err := client.PushChanges(username, token); err != nil {
		t.Fatalf("PushChanges for cleanup failed: %v", err)
	}

	t.Logf("Cleanup complete: test file %s deleted with commit message: %s", fileName, cleanupMsg)
}
