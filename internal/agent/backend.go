// internal/agent/backend.go
package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/egobogo/aiagents/internal/trello"
)

// BackendDeveloperAIAgent specializes in generating code.
type BackendDeveloperAIAgent struct {
	*AIAgent           // embed base agent
	Instruction string // System prompt defining its purpose, inputs, and outputs.
}

// NewBackendDeveloperAIAgent creates a new backend developer agent.
func NewBackendDeveloperAIAgent(base *AIAgent, instruction string) *BackendDeveloperAIAgent {
	return &BackendDeveloperAIAgent{
		AIAgent:     base,
		Instruction: instruction,
	}
}

// GenerateCode generates Go code based on a specific task.
func (b *BackendDeveloperAIAgent) GenerateCode(task string) (string, error) {
	// Combine the specialized instruction with the task.
	prompt := fmt.Sprintf("%s\nTask: %s", b.Instruction, task)
	return b.GPTClient.Chat(prompt)
}

// RequestClarification posts a comment asking for clarification on the ticket.
func (b *BackendDeveloperAIAgent) RequestClarification(ticket *trello.Card) error {
	comment := "Requesting clarification on the technical details. Please provide more info, @engManagerAgent."
	if err := b.WriteComment(ticket, comment); err != nil {
		return fmt.Errorf("failed to request clarification: %w", err)
	}
	return nil
}

// ExecuteTechnicalAssignment generates Go code based on the ticket's description and writes it to a file.
func (b *BackendDeveloperAIAgent) ExecuteTechnicalAssignment(ticket *trello.Card) error {
	prompt := fmt.Sprintf("Generate production-ready Go code with tests for the following technical assignment:\n%s", ticket.Desc)
	code, err := b.GPTClient.Chat(prompt)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}
	// Use the ticket ID to generate a unique file name.
	fileName := fmt.Sprintf("code_%s.go", ticket.ID)
	if err := b.WriteToGit(fileName, []byte(code)); err != nil {
		return fmt.Errorf("failed to write generated code to git: %w", err)
	}
	return nil
}

// CommitAndPushTicketResult commits all changes with a message and pushes them to the remote.
func (b *BackendDeveloperAIAgent) CommitAndPushTicketResult(ticket *trello.Card, commitMessage, authorName, authorEmail, gitUsername, gitToken string) error {
	fullMessage := fmt.Sprintf("%s (Ticket: %s)", commitMessage, ticket.Name)
	if err := b.GitClient.CommitChanges(fullMessage, authorName, authorEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	if err := b.GitClient.PushChanges(gitUsername, gitToken); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

// CloseTicket moves the ticket to the Done column and reassigns it.
func (b *BackendDeveloperAIAgent) CloseTicket(ticket *trello.Card, finalAssignee string) error {
	// Get the ID for the Done column. (Assume TrelloClient has GetDoneListID.)
	doneListID, err := b.TrelloClient.GetListIDByName("Done")
	if err != nil {
		return fmt.Errorf("failed to get Done list ID: %w", err)
	}
	if err := b.ChangeTicketColumn(ticket, doneListID); err != nil {
		return fmt.Errorf("failed to move ticket to Done: %w", err)
	}
	if err := b.ChangeTicketAssignee(ticket, finalAssignee); err != nil {
		return fmt.Errorf("failed to reassign ticket: %w", err)
	}
	return nil
}

// WaitForClarificationResponse polls the ticket's comments until a clarification response is found.
// It assumes that the clarifying comment from the engineering manager will contain "@engManagerAgent".
func (b *BackendDeveloperAIAgent) WaitForClarificationResponse(ticket *trello.Card) error {
	const (
		clarifierTag = "@engManagerAgent"
		pollInterval = 30 * time.Second
		maxAttempts  = 10 // Poll for up to 5 minutes (10 * 30 seconds)
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		comments, err := b.ReadComments(ticket)
		if err != nil {
			return fmt.Errorf("failed to read comments: %w", err)
		}
		// Check if any comment includes the clarifier tag.
		for _, comment := range comments {
			if strings.Contains(comment, clarifierTag) {
				// Clarification response found.
				return nil
			}
		}
		// Wait before polling again.
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("clarification response not received within the expected time")
}
