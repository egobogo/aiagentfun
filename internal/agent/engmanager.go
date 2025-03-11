// internal/agent/engmanager.go
package agent

import (
	"fmt"
	"strings"

	"github.com/egobogo/aiagents/internal/trello"
)

// EngineeringManagerAIAgent specializes in ticket analysis and task decomposition.
type EngineeringManagerAIAgent struct {
	*AIAgent
	Instruction string // System prompt detailing its purpose.
}

// AssignTicketToAgent assigns the given ticket (card) to the specified agent by updating its member assignment.
func (e *EngineeringManagerAIAgent) AssignTicketToAgent(card *trello.Card, agentName string) error {
	// Wrap the card to get access to our helper methods.
	myCard := trello.WrapCard(card)
	// Call our helper method to update the assignment.
	if err := myCard.AssignMember(agentName); err != nil {
		return fmt.Errorf("failed to assign ticket to agent %s: %w", agentName, err)
	}
	return nil
}

// NewEngineeringManagerAIAgent creates a new engineering manager agent.
func NewEngineeringManagerAIAgent(base *AIAgent, instruction string) *EngineeringManagerAIAgent {
	return &EngineeringManagerAIAgent{
		AIAgent:     base,
		Instruction: instruction,
	}
}

// DecomposeTicket decomposes a ticket into atomic technical tasks.
func (e *EngineeringManagerAIAgent) DecomposeTicket(ticket string) ([]string, error) {
	prompt := fmt.Sprintf("%s\nDecompose the following ticket into atomic, actionable tasks:\n%s", e.Instruction, ticket)
	response, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose ticket: %w", err)
	}
	// For simplicity, assume each task is separated by a newline.
	tasks := strings.Split(response, "\n")
	return tasks, nil
}

// HandleTicket allows the Engineering Manager to process a ticket.
// It displays the ticket, allows entering clarifications (or approval), and posts a comment.
func (e *EngineeringManagerAIAgent) HandleTicket(card *trello.Card) error {
	// Display ticket details.
	fmt.Printf("Ticket [%s]: %s\nDescription: %s\n", card.ID, card.Name, card.Desc)

	// Prompt for clarification input.
	fmt.Println("Enter clarifications (or type 'approve' if clear):")
	var input string
	fmt.Scanln(&input)
	if strings.ToLower(input) != "approve" {
		// Post the clarification request as a comment.
		if err := e.WriteComment(card, "Clarification requested: "+input); err != nil {
			return fmt.Errorf("failed to write clarification comment: %w", err)
		}
		// In production, poll until an "approval" comment appears.
		fmt.Println("Waiting for approval... (simulate by typing 'approve')")
		fmt.Scanln(&input)
		if strings.ToLower(input) != "approve" {
			return fmt.Errorf("ticket handling aborted")
		}
	}
	return nil
}

// CreateTechnicalTicket generates a technical ticket based on a high-level ticket.
func (e *EngineeringManagerAIAgent) CreateTechnicalTicket(ticket *trello.Card) (*trello.Card, error) {
	prompt := fmt.Sprintf("Decompose this ticket into detailed technical tasks with clear atomic assignments:\n%s", ticket.Desc)
	techDesc, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate technical description: %w", err)
	}
	// Create a new card with a technical header.
	techTicket, err := e.TrelloClient.CreateCard("Technical: "+ticket.Name, techDesc, "Doing")
	if err != nil {
		return nil, fmt.Errorf("failed to create technical ticket: %w", err)
	}
	return techTicket, nil
}
