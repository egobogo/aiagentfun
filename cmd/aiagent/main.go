// cmd/aiagent/main.go

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/egobogo/aiagents/internal/agent"
	"github.com/egobogo/aiagents/internal/gitrepo"
	"github.com/egobogo/aiagents/internal/trello"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables or config
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or could not load it. Relying on system environment variables.")
	}

	// Load trello details.
	apiKey := os.Getenv("TRELLO_API_KEY")
	apiToken := os.Getenv("TRELLO_API_TOKEN")
	boardID := os.Getenv("TRELLO_BOARD_ID")
	if apiKey == "" || apiToken == "" {
		log.Fatal("TRELLO_API_KEY or TRELLO_API_TOKEN not set")
	}

	// Load Git repository details.
	repoPath := os.Getenv("GIT_REPO_PATH")
	gitUsername := os.Getenv("GIT_USERNAME")
	gitToken := os.Getenv("GIT_TOKEN")
	if repoPath == "" || gitUsername == "" || gitToken == "" {
		log.Fatal("GIT_REPO_PATH, GIT_USERNAME, or GIT_TOKEN is missing")
	}

	// Create the TrelloClient
	trelloClient := trello.NewTrelloClient(apiKey, apiToken, boardID)

	// Create the Git client.
	gitClient, err := gitrepo.NewGitClient(repoPath)
	if err != nil {
		log.Fatalf("Error creating Git client: %v", err)
	}

	// Initialize the Agent with both clients.
	backendAgent := agent.NewAgent("BackendBot", "Backend Dev", "egobogoaiagent1", trelloClient, gitClient)

	// Retrieve the Trello board columns
	columns, err := backendAgent.ListColumns()
	if err != nil {
		log.Fatalf("Error retrieving columns: %v", err)
	}

	// List all the tickets assigned to me
	if err := backendAgent.ListMyTickets(); err != nil {
		log.Fatalf("Error listing tickets: %v", err)
	}

	// Read all files from the repository
	if err := backendAgent.ListRepositoryFiles(); err != nil {
		log.Fatalf("Error reading repository files: %v", err)
	}

	fmt.Println("Columns on the board:", columns)
}
