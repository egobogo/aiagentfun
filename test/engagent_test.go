// File: test/engagent_test.go
package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"

	"github.com/egobogo/aiagents/internal/agent"
	trelloClient "github.com/egobogo/aiagents/internal/board/trello"
	"github.com/egobogo/aiagents/internal/config"
	"github.com/egobogo/aiagents/internal/config/filesys"
	"github.com/egobogo/aiagents/internal/context/embedding/openai"
	"github.com/egobogo/aiagents/internal/context/inmemory"
	"github.com/egobogo/aiagents/internal/context/similarity/hnsw"
	"github.com/egobogo/aiagents/internal/docs/notion"
	"github.com/egobogo/aiagents/internal/gitrepo"
	"github.com/egobogo/aiagents/internal/model/chatgpt"
	"github.com/egobogo/aiagents/internal/model/chatgpt/vectorstorage"
	"github.com/egobogo/aiagents/internal/promptbuilder/chatgptpromptbuilder"
)

func TestEngineeringManagerAgentContext(t *testing.T) {
	t.Log("Starting Engineering Manager Agent context test")

	// Load environment variables.
	if err := godotenv.Load("../.env"); err != nil {
		t.Log("No .env file found; using system environment variables")
	}

	// Load configuration from YAML file.
	prov, err := filesys.NewFilesysConfigProvider("../cfg/main.cfg.yaml")
	if err != nil {
		t.Fatalf("Could not create config provider: %v", err)
	}
	config.SetProvider(prov)
	if err := config.Load("../cfg/main.cfg.yaml"); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Retrieve required environment variables.
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping test")
	}
	notionToken := os.Getenv("NOTION_TOKEN")
	if notionToken == "" {
		t.Skip("NOTION_TOKEN not set, skipping test")
	}
	notionParent := os.Getenv("NOTION_PARENT_PAGE")
	if notionParent == "" {
		t.Skip("NOTION_PARENT_PAGE not set, skipping test")
	}
	trelloAPIKey := os.Getenv("TRELLO_API_KEY")
	trelloToken := os.Getenv("TRELLO_TOKEN")
	trelloBoardID := os.Getenv("TRELLO_BOARD_ID")

	// Create the VectorStorage client.
	vsClient := vectorstorage.NewClient(openaiAPIKey)

	// Create the ChatGPT model client with vector storage.
	modelClient := chatgpt.NewChatGPTClient(openaiAPIKey, "gpt-4o-mini", vsClient)

	// Create the prompt builder.
	promptBuilder := chatgptpromptbuilder.New()

	// Create the Docs client using Notion.
	docsClient := notion.NewNotionClient(notionToken, notionParent)

	// Create the Git client (requires GIT_REPO_URL and GIT_REPO_PATH to be set).
	gitRepoURL := os.Getenv("GIT_REPO_URL")
	gitRepoPath := os.Getenv("GIT_REPO_PATH")
	if gitRepoURL == "" || gitRepoPath == "" {
		t.Skip("GIT_REPO_URL or GIT_REPO_PATH not set, skipping test")
	}
	gitClient, err := gitrepo.NewGitClient(gitRepoURL, gitRepoPath)
	if err != nil {
		t.Fatalf("Failed to create GitClient: %v", err)
	}

	// Create a board client if Trello credentials are provided.
	var boardClient *trelloClient.TrelloClient
	if trelloAPIKey != "" && trelloToken != "" && trelloBoardID != "" {
		boardClient = trelloClient.NewTrelloClient(trelloAPIKey, trelloToken, trelloBoardID)
	}

	// Create context storage with concrete implementations.
	embeddingProvider := openai.NewOpenAIEmbeddingProvider(openaiAPIKey, "text-embedding-ada-002")
	hnswSearcher, err := hnsw.New(1536)
	if err != nil {
		t.Fatalf("Failed to create HNSW SimilaritySearcher: %v", err)
	}
	ctxStorage := inmemory.NewInMemoryContextStorage(embeddingProvider, hnswSearcher)

	// Create a BaseAgent with the concrete dependencies.
	baseAgent := &agent.BaseAgent{
		Name:          "EngineeringManager",
		Role:          "EngineeringManager",
		ModelClient:   modelClient,
		BoardClient:   boardClient,
		DocsClient:    docsClient,
		GitClient:     gitClient,
		Context:       ctxStorage,
		PromptBuilder: promptBuilder,
		VectorStorage: vsClient, // Pass the vector storage client here.
	}

	// Create the Engineering Manager agent.
	engAgent := agent.NewEngineeringManagerAgent(baseAgent)

	// Retrieve and log the hot context.
	hotContext := engAgent.Context.GetContext()
	t.Log("Hot Context:")
	t.Log(hotContext)

	// Retrieve the memory records (cold storage) as a JSON string.
	memoriesJSON := engAgent.Context.GetMemories()
	if err != nil {
		t.Logf("Failed to get memories: %v", err)
	} else {
		// Write the memory records to a logs file under test/logs.
		logsDir := filepath.Join("test", "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			t.Logf("Failed to create logs directory: %v", err)
		}
		if err != nil {
			t.Logf("Failed to get memories: %v", err)
		} else {
			// Marshal the slice into JSON.
			marshaled, err := json.MarshalIndent(memoriesJSON, "", "  ")
			if err != nil {
				t.Logf("Failed to marshal memories: %v", err)
			} else {
				logsDir := filepath.Join("test", "logs")
				if err := os.MkdirAll(logsDir, 0755); err != nil {
					t.Logf("Failed to create logs directory: %v", err)
				}
				logFilePath := filepath.Join(logsDir, "agent_memories.json")
				if err := os.WriteFile(logFilePath, marshaled, 0644); err != nil {
					t.Logf("Failed to write memories to file: %v", err)
				}
				t.Logf("Memory records written to: %s", logFilePath)
			}
		}
	}
}
