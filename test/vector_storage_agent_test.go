package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/egobogo/aiagents/internal/config"
	"github.com/egobogo/aiagents/internal/config/filesys"
	"github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/model/chatgpt"
	"github.com/egobogo/aiagents/internal/model/chatgpt/vectorstorage"
	"github.com/egobogo/aiagents/internal/promptbuilder/chatgptpromptbuilder"
)

func TestAskAboutFileContent_WithVectorStore(t *testing.T) {
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
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("OPENAI_API_KEY not set in .env")
	}

	// Create a new vector storage client.
	vsClient := vectorstorage.NewClient(apiKey)

	// Step 1: Create a new vector store for this test.
	vsName := fmt.Sprintf("TestVectorStore_%d", time.Now().Unix())
	vectorStore, err := vsClient.CreateStorage(vsName)
	if err != nil {
		t.Fatalf("CreateStorage failed: %v", err)
	}
	t.Logf("Vector store created: ID=%s, Name=%s", vectorStore.ID, vectorStore.Name)
	vectorStoreID := vectorStore.ID

	// Step 2: Create a temporary file with known content.
	testContent := "This is a test file with secret content: Hello, world!"
	tempFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("testfile_%d.txt", time.Now().Unix()))
	if err := ioutil.WriteFile(tempFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFilePath)
	t.Logf("Temporary file created: %s", tempFilePath)

	// Step 3: Initialize ChatGPTClient with the vector storage client.
	client := chatgpt.NewChatGPTClient(apiKey, "gpt-4o-mini", vsClient)

	// Step 4: Upload the file.
	uploadedFile, err := client.UploadFile(tempFilePath, string(model.FilePurposeAssistants))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	t.Logf("File uploaded: ID=%s, Filename=%s, Purpose=%s", uploadedFile.ID, uploadedFile.Filename, uploadedFile.Purpose)

	// Step 5: Attach the uploaded file to the vector store using the vector storage client.
	attachedFile, err := vsClient.AttachFile(vectorStoreID, uploadedFile.ID)
	if err != nil {
		t.Fatalf("AttachFile failed: %v", err)
	}
	t.Logf("File attached to vector store: ID=%s", attachedFile.ID)

	// Step 6: Build a ChatRequest using ChatGPTPromptBuilder.
	// We modify the query to specifically ask: "What secret content is contained in the attached file?"
	builder := chatgptpromptbuilder.New()
	query := "What secret content is contained in the attached file?"
	chatReq, err := builder.Build("Empty", "Empty", "", query, nil, 1.2, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Step 7: Attach the file tool block to the ChatRequest.
	err = builder.AddFile(&chatReq, []string{vectorStoreID})
	if err != nil {
		t.Fatalf("AddFile (prompt builder) failed: %v", err)
	}
	t.Logf("ChatRequest after attaching file: %+v", chatReq)

	// Step 8: Send the ChatRequest using ChatAdvanced.
	response, err := client.ChatAdvanced(chatReq)
	if err != nil {
		t.Fatalf("ChatAdvanced failed: %v", err)
	}
	t.Logf("Response from ChatAdvanced: %s", response)

	// Step 9: Verify that the response contains "Hello, world!".
	if !strings.Contains(response, "Hello, world!") {
		t.Fatalf("expected response to contain 'Hello, world!', got: %s", response)
	}

	// Step 10: Cleanup: Delete all uploaded files.
	if err := client.DeleteAllFiles(); err != nil {
		t.Fatalf("DeleteAllFiles failed: %v", err)
	}
	t.Log("Cleanup: All files deleted successfully")

	// Step 11: Delete the created vector store.
	if err := vsClient.DeleteStorage(vectorStoreID); err != nil {
		t.Fatalf("DeleteStorage failed: %v", err)
	}
	t.Log("Cleanup: Vector store deleted successfully")
}
