package agent

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egobogo/aiagents/internal/context"
	"github.com/egobogo/aiagents/internal/model"
)

// EngineeringManagerAgent implements the Agent interface.
type EngineeringManagerAgent struct {
	*BaseAgent
}

// NewEngineeringManagerAgent creates a new EngineeringManagerAgent.
func NewEngineeringManagerAgent(base *BaseAgent) *EngineeringManagerAgent {
	engManagerAgent := &EngineeringManagerAgent{
		BaseAgent: base,
	}
	if err := engManagerAgent.createContext(); err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
	}
	return engManagerAgent
}

// logStep appends a log entry to "context_debug.log".
func logStep(step, content string) {
	logFile := "context_debug.log"
	timestamp := time.Now().Format(time.RFC3339)
	entry := fmt.Sprintf("[%s] %s: %s\n", timestamp, step, content)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		fmt.Printf("Error writing log entry: %v\n", err)
	}
}

// stripMemories returns a summary of memory entries.
func stripMemories(memories []context.MemoryEntry) string {
	var summaries []string
	for _, mem := range memories {
		summary := fmt.Sprintf("Category: %s | Importance: %d | Content: %s", mem.Category, mem.Importance, mem.Content)
		summaries = append(summaries, summary)
	}
	return strings.Join(summaries, "\n")
}

// createContext gathers documentation and repository info, generates memories, and updates the agent's context.
func (em *EngineeringManagerAgent) createContext() error {
	// ------------------------------
	// Step 1: Process Documentation Info.
	// ------------------------------
	docTree, err := em.DocsClient.PrintTree()
	if err != nil {
		return fmt.Errorf("failed to get documentation tree: %w", err)
	}
	pages, err := em.DocsClient.ListPages()
	if err != nil {
		return fmt.Errorf("failed to list documentation pages: %w", err)
	}
	var pagesInfo string
	for _, p := range pages {
		content, _ := em.DocsClient.ReadPage(p.ID)
		pagesInfo += fmt.Sprintf("Title: %s\nContent: %s\n", p.Title, content)
	}
	docPrompt := "Below you can find information about the documentation of the project you are working on. Your task is to form human-like specific memories that help you execute your role. Try not to remember obvious statements but focus on specifics that aid your day-to-day tasks. Below you will find the tree of the documentation structure, followed by the actual documentation articles."
	combinedDocContent := docPrompt + "\n" + docTree + "\n" + pagesInfo

	// Generate documentation memories using CreateThoughts.
	docMemories, err := em.CreateThoughts(combinedDocContent, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create thoughts from documentation: %w", err)
	}
	for _, mem := range docMemories {
		em.Context.Remember(mem)
	}

	initialContext, err := em.BuildContext(docMemories, []context.MemoryEntry{})
	if err != nil {
		return fmt.Errorf("failed to build initial context: %w", err)
	}
	if err := em.Context.SetContext(initialContext); err != nil {
		return fmt.Errorf("failed to set hot context: %w", err)
	}

	// ------------------------------
	// Step 2: Process Repository (Code) Files.
	// ------------------------------
	// Retrieve code files via GitClient.
	codeFiles, err := em.GitClient.ListCodeFiles()
	if err != nil {
		return fmt.Errorf("failed to list code files: %w", err)
	}

	// Ensure the vector storage client is configured.
	vsClient := em.VectorStorage
	if vsClient == nil {
		return fmt.Errorf("vector storage client not configured")
	}

	// Check for a vector store named "aiagents" and create if missing.
	vectorStoreID := ""
	storages, err := vsClient.ListStorages()
	if err != nil {
		return fmt.Errorf("failed to list vector stores: %w", err)
	}
	for _, vs := range storages {
		if vs.Name == "aiagents" {
			vectorStoreID = vs.ID
			break
		}
	}
	if vectorStoreID == "" {
		newVS, err := vsClient.CreateStorage("aiagents")
		if err != nil {
			return fmt.Errorf("failed to create vector store: %w", err)
		}
		vectorStoreID = newVS.ID
	}

	// Prepare an array of file attachments (each with file ID and vector store ID).
	var fileTuple []model.FileAttachment
	for _, filePath := range codeFiles {
		uploaded, err := em.ModelClient.UploadFile(filePath, string(model.FilePurposeAssistants))
		if err != nil {
			return fmt.Errorf("failed to upload file %s: %w", filePath, err)
		}
		// Attach the file and wait until it's processed.
		_, err = vsClient.AttachFile(vectorStoreID, uploaded.ID)
		if err != nil {
			return fmt.Errorf("failed to attach file %s to vector store: %w", filePath, err)
		}
		// Append the tuple with correct field names.
		fileTuple = append(fileTuple, model.FileAttachment{FileID: uploaded.ID, VectorStoreID: vectorStoreID})
	}

	// Get repository structure (code tree) from GitClient.
	gitTree, err := em.GitClient.PrintTree()
	if err != nil {
		return fmt.Errorf("failed to gather repository info: %w", err)
	}
	// Construct a prompt for repository info.
	repoInput := fmt.Sprintf("In the attachments you can find the code of the repository. Study it carefully and extract memories about each struct, function, and purpose for your further development. GitStructure:\n%s", gitTree)

	// Generate repository memories using CreateThoughts with the file attachments.
	repoMemories, err := em.CreateThoughts(repoInput, fileTuple, nil)
	if err != nil {
		return fmt.Errorf("failed to create thoughts from repository info: %w", err)
	}

	// ------------------------------
	// Step 3: Merge and Refresh Context.
	// ------------------------------
	// Combine the new memories.
	newMemories := append(docMemories, repoMemories...)
	// Filter related old memories.
	collectedOldMemories := em.Context.FilterRelatedMemories(newMemories)
	// Build the updated context.
	updatedContext, err := em.BuildContext(newMemories, collectedOldMemories)
	if err != nil {
		return fmt.Errorf("failed to build updated context: %w", err)
	}
	if err := em.Context.SetContext(updatedContext); err != nil {
		return fmt.Errorf("failed to set hot context: %w", err)
	}
	// Refresh memories.
	if err := em.RefreshMemories(collectedOldMemories, newMemories); err != nil {
		return fmt.Errorf("failed to refresh memories: %w", err)
	}

	return nil
}
