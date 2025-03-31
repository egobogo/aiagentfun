package test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/egobogo/aiagents/internal/config"
	"github.com/egobogo/aiagents/internal/config/filesys"
	modelClient "github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/model/chatgpt"
	"github.com/egobogo/aiagents/internal/promptbuilder/chatgptpromptbuilder"
)

func TestWebSearch(t *testing.T) {
	// Load configuration from YAML file.
	prov, err := filesys.NewFilesysConfigProvider("../cfg/main.cfg.yaml")
	if err != nil {
		t.Fatalf("Could not create config provider: %v", err)
	}
	config.SetProvider(prov)
	if err := config.Load("../cfg/main.cfg.yaml"); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Load environment variables from ../.env.
	err = godotenv.Load("../.env")
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("OPENAI_API_KEY not set in .env")
	}

	// Initialize ChatGPTClient (no vector store ID needed for web search).
	client := chatgpt.NewChatGPTClient(apiKey, "gpt-4o-mini", "")

	// Build a ChatRequest using ChatGPTPromptBuilder.
	builder := chatgptpromptbuilder.New()
	query := "What was a positive news story from today?"
	chatReq, err := builder.Build("Empty", "Empty", "", query, nil, 1.2, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Create a WebSearch configuration.
	webTool := modelClient.WebSearch{
		Type: "web_search_preview",
		UserLocation: map[string]interface{}{
			"type":    "approximate",
			"country": "US",
			"city":    "San Francisco",
			"region":  "California",
		},
		ContextSize: modelClient.SearchContextSizeMedium,
	}

	// Attach the web search tool block to the ChatRequest.
	err = builder.AddWeb(&chatReq, webTool)
	if err != nil {
		t.Fatalf("AddWeb failed: %v", err)
	}
	t.Logf("ChatRequest with web search: %+v", chatReq)

	// Send the ChatRequest using ChatAdvanced.
	response, err := client.ChatAdvanced(chatReq)
	if err != nil {
		t.Fatalf("ChatAdvanced failed: %v", err)
	}
	t.Logf("Response from ChatAdvanced: %s", response)

	// Verify that the response is not empty.
	if response == "" {
		t.Fatalf("Expected non-empty response from web search query")
	}
}
