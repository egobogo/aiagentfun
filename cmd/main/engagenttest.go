package main

import (
	"log"
	"os"
	"strings"

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
	"github.com/egobogo/aiagents/internal/promptbuilder/chatgptpromptbuilder"
	// for ChatRequest and Message types
)

func main() {
	log.Println("Fetching env")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Load configuration from YAML file.
	prov, err := filesys.NewFilesysConfigProvider("cfg/main.cfg.yaml")
	if err != nil {
		log.Fatalf("Could not create config provider: %v", err)
	}
	config.SetProvider(prov)
	if err := config.Load("cfg/main.cfg.yaml"); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Retrieve required environment variables.
	openaiAPIKey := os.Getenv("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		log.Println("OPENAI_API_KEY not set")
	}
	notionToken := os.Getenv("NOTION_TOKEN")
	if notionToken == "" {
		log.Println("NOTION_TOKEN not set")
	}
	notionParent := os.Getenv("NOTION_PARENT_PAGE")
	if notionParent == "" {
		log.Println("NOTION_PARENT_PAGE not set")
	}
	trelloAPIKey := os.Getenv("TRELLO_API_KEY")
	trelloToken := os.Getenv("TRELLO_TOKEN")
	trelloBoardID := os.Getenv("TRELLO_BOARD_ID")

	// Create the ChatGPT model client.
	modelClient := chatgpt.NewChatGPTClient(openaiAPIKey, "gpt-4o-mini")

	// Create the prompt builder.
	promptBuilder := chatgptpromptbuilder.New()

	// Create the Docs client using Notion.
	docsClient := notion.NewNotionClient(notionToken, notionParent)

	// Use the correct environment variable name for the Git repository path.
	repoPath := strings.TrimSpace(os.Getenv("GIT_REPO_PATH"))
	repoURL := os.Getenv("GIT_REPO_URL")
	if repoPath == "" || repoURL == "" {
		log.Println("GIT_REPO_PATH or GIT_REPO_URL not set")
	}

	// Create the Git client (this will open the existing repository if it already exists).
	gitClient, err := gitrepo.NewGitClient(repoURL, repoPath)
	if err != nil {
		// Log error using proper formatting.
		log.Printf("Failed to create GitClient: %v", err)
	} else {
		log.Println("GitClient created successfully")
	}

	// Create a board client if Trello credentials are provided; otherwise, leave it nil.
	boardClient := trelloClient.NewTrelloClient(trelloAPIKey, trelloToken, trelloBoardID)

	// Create context storage with concrete implementations:
	// OpenAIEmbeddingProvider (for embeddings) and HNSWSimilaritySearcher.
	embeddingProvider := openai.NewOpenAIEmbeddingProvider(openaiAPIKey, "text-embedding-ada-002")
	hnswSearcher, err := hnsw.New(1536)
	if err != nil {
		log.Println("Failed to create HNSW SimilaritySearcher: %v", err)
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
	}

	// Create the Engineering Manager agent.
	engAgent := agent.NewEngineeringManagerAgent(baseAgent)
	log.Println(engAgent.Name)
}
