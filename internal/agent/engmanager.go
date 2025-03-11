// internal/agent/engmanager.go
package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

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
	member, err := e.TrelloClient.GetMemberByName(agentName)
	if member == nil {
		return fmt.Errorf("failed to find an agent %s: %w", agentName, err)
	}

	if err := myCard.AssignMember(member.ID); err != nil {
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

// HandleTicket processes a ticket by generating clarifications based on Git context and ticket details,
// posting them (tagging @bogoego), waiting for a reply that tags @egobogoengmanageragent,
// and finally passing that reply to GPT.
func (e *EngineeringManagerAIAgent) HandleTicket(card *trello.Card) error {
	// 1. Scan and read the Git repository.
	gitFiles, err := e.ReadAllGitFiles() // method defined in the base agent
	if err != nil {
		return fmt.Errorf("failed to read git repository: %w", err)
	}
	// Build a simple summary of the repository.
	var gitSummary strings.Builder
	for file, content := range gitFiles {
		gitSummary.WriteString(fmt.Sprintf("File: %s\nContent:\n%s\n----------------\n", file, content))
	}

	// 2. Pass the ticket details and git context to GPT to generate clarifications.
	ticketInfo := fmt.Sprintf("Ticket ID: %s\nTitle: %s\nDescription: %s", card.ID, card.Name, card.Desc)
	prompt := fmt.Sprintf(
		"Given the following Git repository context:\n%s\nand the ticket:\n%s\nGenerate a list of clarifying questions.",
		gitSummary.String(), ticketInfo,
	)
	clarifications, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return fmt.Errorf("failed to generate clarifications: %w", err)
	}

	// 3. Post the generated clarifications as a comment, tagging @bogoego.
	clarificationComment := clarifications + "\n@bogoego"
	if err := e.WriteComment(card, clarificationComment); err != nil {
		return fmt.Errorf("failed to post clarification comment: %w", err)
	}
	log.Printf("Posted clarifications on ticket %s", card.ID)

	// 4. Wait until a reply is posted that tags @egobogoengmanageragent.
	reply, err := e.WaitForReply(card, "@egobogoengmanageragent")
	if err != nil {
		return fmt.Errorf("failed to receive reply: %w", err)
	}
	log.Printf("Received reply: %s", reply)

	// 5. Pass the reply to GPT for further processing.
	prompt := fmt.Sprintf(
		"Given the following clarifications, create tenchnical specification and split it into a list of atomic technical tasks. " +
			"Each task should be clear and unambiguous, with a concise title.")

	response, err := e.GPTClient.Chat(prompt + " " + reply)
	if err != nil {
		return fmt.Errorf("failed to process reply with GPT: %w", err)
	}

	//TODO add parsing and ticket creation for the gpt reply here

	return nil
}

// WaitForReply polls the ticket's comments until one is found that contains the required tag.
func (e *EngineeringManagerAIAgent) WaitForReply(card *trello.Card, requiredTag string) (string, error) {
	const pollInterval = 30 * time.Second
	const maxAttempts = 10 // Adjust as needed

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		comments, err := e.ReadComments(card) // defined in the base agent
		if err != nil {
			return "", fmt.Errorf("failed to read comments: %w", err)
		}
		for _, comment := range comments {
			if strings.Contains(comment, requiredTag) {
				return comment, nil
			}
		}
		log.Printf("No reply with %s found, attempt %d/%d. Waiting...", requiredTag, attempt, maxAttempts)
		time.Sleep(pollInterval)
	}
	return "", fmt.Errorf("reply with tag %s not received after polling", requiredTag)
}

// SplitTicketIntoAtomicTasks takes a high-level ticket and splits it into atomic technical tasks.
// It returns the newly created technical tickets (cards).
func (e *EngineeringManagerAIAgent) SplitTicketIntoAtomicTasks(ticket *trello.Card) ([]*trello.Card, error) {
	// 1. Use GPT to generate a list of atomic tasks.
	prompt := fmt.Sprintf(
		"Given the following technical specification, split it into a list of atomic technical tasks. "+
			"Each task should be clear and unambiguous, with a concise title. \n\nSpecification:\n%s",
		ticket.Desc)
	response, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate atomic tasks: %w", err)
	}

	// 2. Assume tasks are returned as a newline-separated list.
	tasks := strings.Split(response, "\n")
	var createdTickets []*trello.Card

	// 3. Get the list ID for the "Doing" column.
	doingListID, err := e.TrelloClient.GetListIDByName("Doing")
	if err != nil {
		return nil, fmt.Errorf("failed to get Doing list ID: %w", err)
	}

	// 4. For each non-empty task, create a technical ticket.
	for _, task := range tasks {
		task = strings.TrimSpace(task)
		if task == "" {
			continue
		}
		title := "Technical: " + task
		// Optionally, you could further refine the description here.
		techTicket, err := e.TrelloClient.CreateCard(title, task, doingListID)
		if err != nil {
			// Log error and continue with next task.
			log.Printf("failed to create technical ticket for task '%s': %v", task, err)
			continue
		}
		createdTickets = append(createdTickets, techTicket)
	}

	if len(createdTickets) == 0 {
		return nil, fmt.Errorf("no technical tickets were created")
	}
	return createdTickets, nil
}

// CreateTechnicalTicket generates a technical ticket based on a high-level ticket.
func (e *EngineeringManagerAIAgent) CreateTechnicalTicket(ticket *trello.Card) (*trello.Card, error) {
	prompt := fmt.Sprintf("Decompose this ticket into detailed technical tasks with clear atomic assignments:\n%s", ticket.Desc)
	techDesc, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate technical description: %w", err)
	}
	// Create a new card with a technical header.
	// In the Engineering Manager agent when creating a technical ticket:
	doingListID, err := e.TrelloClient.GetListIDByName("Doing")
	if err != nil {
		return nil, fmt.Errorf("failed to get Doing list ID: %w", err)
	}
	techTicket, err := e.TrelloClient.CreateCard("Technical: "+ticket.Name, techDesc, doingListID)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical ticket: %w", err)
	}
	return techTicket, nil
}
