// cmd/aiagent/main.go

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/egobogo/aiagents/internal/agent"
	"github.com/egobogo/aiagents/internal/trello"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load environment variables or config
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or could not load it. Relying on system environment variables.")
	}

	apiKey := os.Getenv("TRELLO_API_KEY")
	apiToken := os.Getenv("TRELLO_API_TOKEN")
	if apiKey == "" || apiToken == "" {
		log.Fatal("TRELLO_API_KEY or TRELLO_API_TOKEN not set")
	}

	// 2. Create the TrelloClient
	trelloClient := trello.NewTrelloClient(apiKey, apiToken)

	// 3. Create the agent
	// In Trello, you might have a user with username "backend-bot" or ID "5f..."
	backendAgent := agent.NewAgent(
		"BackendBot",
		"Backend Developer",
		"trelloUserIDOrUsername", // e.g., "backend-bot"
		trelloClient,
	)

	// 3) Retrieve the Trello board columns
	boardID := os.Getenv("BOARD_ID")
	columns, err := backendAgent.ListColumns(boardID)
	if err != nil {
		log.Fatalf("Error retrieving columns: %v", err)
	}

	fmt.Println("Columns on the board:", columns)
}
