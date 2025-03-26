package agent

import (
	"encoding/json"
	"fmt"

	"github.com/egobogo/aiagents/internal/board"
	"github.com/egobogo/aiagents/internal/context"
	"github.com/egobogo/aiagents/internal/docs"
	"github.com/egobogo/aiagents/internal/gitrepo"
	mclient "github.com/egobogo/aiagents/internal/model"
	pb "github.com/egobogo/aiagents/internal/promptbuilder"
)

// Agent defines the basic operations available to any agent.
type Agent interface {
	Act() error
	FindMyTickets() ([]board.Card, error)
	Think(senderContext, userInput, mode string, desiredOutput interface{}) (mclient.Message, error)
	Answer(senderContext, userInput string, desiredOutput interface{}) (mclient.Message, error)
	Summarize(inputStr string, inputSchema interface{}) ([]context.EasyMemory, error)
	createContext() error
}

// BaseAgent provides the common functionality for all agents.
type BaseAgent struct {
	Name            string
	CurrentTicketID string
	Role            string

	ModelClient   mclient.ModelClient
	BoardClient   board.BoardClient
	DocsClient    docs.DocumentationClient
	GitClient     *gitrepo.GitClient
	Context       context.ContextStorage
	PromptBuilder pb.PromptBuilder
}

// FindMyTickets retrieves board cards assigned to this agent.
func (a *BaseAgent) FindMyTickets() ([]board.Card, error) {
	return a.BoardClient.GetCardsAssignedTo(a.Name)
}

// Think builds a request, obtains a response, and updates context.
func (a *BaseAgent) Think(senderContext, userInput, mode string, desiredOutput interface{}) (mclient.Message, error) {
	combinedInput := fmt.Sprintf("Context of the sender:\n%s\n\nThe query of the sender:\n%s", senderContext, userInput)
	newMemories, err := a.Summarize(combinedInput, nil)
	if err != nil {
		return mclient.Message{}, fmt.Errorf("failed to summarize new input: %w", err)
	}

	relevantOldMemories := a.Context.FilterRelatedMemories(newMemories)
	updatedContext, err := a.BuildContext(newMemories, relevantOldMemories)
	if err != nil {
		return mclient.Message{}, fmt.Errorf("failed to build updated context: %w", err)
	}

	if err := a.Context.SetContext(updatedContext); err != nil {
		return mclient.Message{}, fmt.Errorf("failed to set hot context: %w", err)
	}

	if err := a.RefreshMemories(relevantOldMemories, newMemories); err != nil {
		fmt.Printf("Warning: RefreshMemories (first pass) failed: %v\n", err)
	}

	chatReq, err := a.PromptBuilder.Build(
		a.Role,
		mode,
		updatedContext,
		userInput,
		desiredOutput,
		a.ModelClient.GetTemperature(),
		a.ModelClient.GetModel(),
	)
	if err != nil {
		return mclient.Message{}, fmt.Errorf("failed to build task request: %w", err)
	}

	taskResponse, err := a.ModelClient.ChatAdvanced(chatReq)
	if err != nil {
		return mclient.Message{}, fmt.Errorf("failed to get task response: %w", err)
	}

	additionalMemories, err := a.Summarize(taskResponse, nil)
	if err != nil {
		fmt.Printf("Warning: failed to summarize task response for additional memories: %v\n", err)
		additionalMemories = []context.EasyMemory{}
	}

	relevantAdditional := a.Context.FilterRelatedMemories(additionalMemories)
	if err := a.RefreshMemories(relevantAdditional, additionalMemories); err != nil {
		fmt.Printf("Warning: RefreshMemories (second pass) failed: %v\n", err)
	}

	return mclient.Message{
		Role:    "assistant",
		Content: taskResponse,
	}, nil
}

// Answer is a wrapper around Think using mode "Answer".
func (a *BaseAgent) Answer(senderContext, userInput string, desiredOutput interface{}) (mclient.Message, error) {
	return a.Think(senderContext, userInput, "Answer", desiredOutput)
}

// Summarize requests a structured output of memories and unmarshals it into []EasyMemory.
func (a *BaseAgent) Summarize(inputStr string, inputSchema interface{}) ([]context.EasyMemory, error) {
	var userPrompt string
	if inputSchema != nil {
		userPrompt = fmt.Sprintf("Your task is to produce an array of memories from the information provided, given your role. Input:\n%s\nSchema: %v", inputStr, inputSchema)
	} else {
		userPrompt = fmt.Sprintf("Your task is to produce an array of memories from the information provided, given your role. Input:\n%s", inputStr)
	}

	// Pass an empty slice to trigger dynamic schema generation for []EasyMemory.
	desiredOutput := []context.EasyMemory{}

	chatReq, err := a.PromptBuilder.Build(
		a.Role,
		"Summarize",
		a.Context.GetContext(),
		userPrompt,
		desiredOutput,
		a.ModelClient.GetTemperature(),
		a.ModelClient.GetModel(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build chat request: %w", err)
	}

	// Unmarshal into a wrapper struct with a "result" field.
	var wrapper struct {
		Result []context.EasyMemory `json:"result"`
	}
	if err := a.ModelClient.ChatAdvancedParsed(chatReq, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse structured JSON: %w", err)
	}

	return wrapper.Result, nil
}

// BuildContext merges new and old memories into an updated context.
func (a *BaseAgent) BuildContext(newMemories []context.EasyMemory, oldMemories []context.MemoryEntry) (string, error) {
	priorHot := a.Context.GetContext()
	if priorHot == "" && len(oldMemories) == 0 {
		return fmt.Sprintf("Context:\n%v", newMemories), nil
	}

	prompt := fmt.Sprintf("New Memory Entries:\n%v\n\nOld Memories:\n%v", newMemories, oldMemories)
	chatReq, err := a.PromptBuilder.Build(
		a.Role,
		"ActualizeContext",
		priorHot,
		prompt,
		nil,
		a.ModelClient.GetTemperature(),
		a.ModelClient.GetModel(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to build hot context merge request: %w", err)
	}

	mergedHot, err := a.ModelClient.ChatAdvanced(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to merge hot context: %w", err)
	}

	return mergedHot, nil
}

// RefreshMemories asks the model which memories to delete and updates context accordingly.
func (a *BaseAgent) RefreshMemories(oldMems []context.MemoryEntry, newMems []context.EasyMemory) error {
	oldJSON, err := json.MarshalIndent(oldMems, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal old memories: %w", err)
	}
	newJSON, err := json.MarshalIndent(newMems, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal new memories: %w", err)
	}

	prompt := fmt.Sprintf("Old Memories:\n%s\nNew Memories:\n%s", string(oldJSON), string(newJSON))

	// Define the expected deletion response.
	type DeleteResponse struct {
		DeleteIDs []string `json:"delete_ids"`
	}
	desiredOutput := DeleteResponse{}

	chatReq, err := a.PromptBuilder.Build(
		a.Role,
		"RefreshMemories",
		a.Context.GetContext(),
		prompt,
		desiredOutput,
		a.ModelClient.GetTemperature(),
		a.ModelClient.GetModel(),
	)
	if err != nil {
		return fmt.Errorf("failed to build refreshMemories chat request: %w", err)
	}

	var delResp DeleteResponse
	if err := a.ModelClient.ChatAdvancedParsed(chatReq, &delResp); err != nil {
		return fmt.Errorf("failed to parse refreshMemories response: %w", err)
	}

	for _, id := range delResp.DeleteIDs {
		if err := a.Context.Forget(id); err != nil {
			fmt.Printf("Warning: failed to forget memory with ID %s: %v\n", id, err)
		}
	}

	for _, emem := range newMems {
		if err := a.Context.Remember(emem); err != nil {
			fmt.Printf("Warning: failed to add new memory: %v\n", err)
		}
	}
	return nil
}
