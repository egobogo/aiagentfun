package test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fmt"

	modelClient "github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/model/chatgpt"
	"github.com/joho/godotenv"
)

func TestFileManipulation(t *testing.T) {
	if err := godotenv.Load("../.env"); err != nil {
		t.Log("No .env file found; using system environment variables")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Fatalf("OPENAI_API_KEY not set, skipping integration tests")
	}

	// Optionally, you can set an initial VectorStoreID if you already have one,
	// but here we will create a new one.
	client := chatgpt.NewChatGPTClient(apiKey, "gpt-4o-mini", "")

	// Create a temporary file for testing.
	tmpDir := os.TempDir()
	testFilePath := filepath.Join(tmpDir, fmt.Sprintf("testfile_%d.txt", time.Now().Unix()))
	testContent := []byte("Hello, world! This is a test file for vector storage integration.")
	if err := ioutil.WriteFile(testFilePath, testContent, 0644); err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(testFilePath)
	t.Logf("Temporary file created: %s", testFilePath)

	// Step 1: Upload the file using the files API.
	uploadedFile, err := client.UploadFile(testFilePath, string(modelClient.FilePurposeAssistants))
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	t.Logf("File uploaded: ID=%s, Filename=%s, Purpose=%s", uploadedFile.ID, uploadedFile.Filename, uploadedFile.Purpose)

	// Step 2: Retrieve file metadata using GetFile.
	retrievedFile, err := client.GetFile(uploadedFile.ID)
	if err != nil {
		t.Fatalf("GetFile failed: %v", err)
	}
	if retrievedFile.ID != uploadedFile.ID {
		t.Fatalf("GetFile returned wrong file, got %s, expected %s", retrievedFile.ID, uploadedFile.ID)
	}
	t.Logf("File retrieved: ID=%s, Filename=%s", retrievedFile.ID, retrievedFile.Filename)

	// Step 3: Create a new vector store for our project.
	vectorStoreName := fmt.Sprintf("Test Vector Store %d", time.Now().Unix())
	vectorStore, err := client.CreateVectorStore(vectorStoreName)
	if err != nil {
		t.Fatalf("CreateVectorStore failed: %v", err)
	}
	t.Logf("Vector store created: ID=%s, Name=%s", vectorStore.ID, vectorStore.Name)

	// Step 4: Attach the uploaded file to the vector store.
	attachedFile, err := client.AddFileToVectorStore(vectorStore.ID, uploadedFile.ID)
	if err != nil {
		t.Fatalf("AddFileToVectorStore failed: %v", err)
	}
	t.Logf("File attached to vector store: FileID=%s", attachedFile.ID)

	// Step 5: Delete all files (cleanup).
	if err := client.DeleteAllFiles(); err != nil {
		t.Fatalf("DeleteAllFiles failed: %v", err)
	}
	t.Log("All files deleted successfully")
}
