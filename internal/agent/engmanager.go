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

// HandleTicket processes a ticket by generating clarifications based on Git context and ticket details,
// posting them (tagging @bogoego), waiting for a reply that tags @egobogoengmanageragent,
// and finally passing that reply to GPT.
func (e *EngineeringManagerAIAgent) HandleTicket(card *trello.Card) ([]*trello.Card, error) {
	// 1. Scan and read the Git repository.
	gitFiles, err := e.ReadAllGitFiles() // method defined in the base agent
	if err != nil {
		return nil, fmt.Errorf("failed to read git repository: %w", err)
	}
	// Build a simple summary of the repository.
	var gitSummary strings.Builder
	for file, content := range gitFiles {
		gitSummary.WriteString(fmt.Sprintf("File: %s\nContent:\n%s\n----------------\n", file, content))
	}

	// 2. Pass the ticket details and git context to GPT to generate clarifications.
	ticketInfo := fmt.Sprintf("Ticket ID: %s\nTitle: %s\nDescription: %s", card.ID, card.Name, card.Desc)
	prompt := fmt.Sprintf(
		"Given the following Git repository context: % sand the ticket details:%s\n"+
			"You are the definitive technical authority for this project. You have complete expertise in all technical areas—choosing the best libraries, applying the most appropriate design patterns, and enforcing optimal coding standards and technical constraints. You already know what technical choices to make.\n"+
			"Your sole responsibility here is to ensure that the ticket’s requirements are unambiguous from a business perspective. If you detect any ambiguity or lack of clarity regarding the business objectives or requirements in the ticket, ask a concise question to the Product Manager to clarify these aspects. Do not ask about technical details such as libraries, design patterns, coding standards, or technical constraints.\n"+
			"Generate a list of clarifying questions that focus exclusively on any potential business ambiguities in the ticket. If the ticket is clear from a business standpoint, simply confirm that the task is technically sound.", gitSummary.String(), ticketInfo)
	clarifications, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate clarifications: %w", err)
	}

	// 3. Post the generated clarifications as a comment, tagging @bogoego.
	clarificationComment := clarifications + "\n@bogoego"
	if err := e.WriteComment(card, clarificationComment); err != nil {
		return nil, fmt.Errorf("failed to post clarification comment: %w", err)
	}
	log.Printf("Posted clarifications on ticket %s", card.ID)

	// 4. Wait until a reply is posted that tags @egobogoengmanageragent.
	reply, err := e.WaitForReply(card, "@"+e.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to receive reply: %w", err)
	}
	log.Printf("Received reply: %s", reply)

	// 5. Pass the reply to GPT for further processing.
	replyPrompt := fmt.Sprintf(
		"Given the following clarifications, create tenchnical clear a list of atomic technical tickets for the backend developer with only coding tasks. Each task should be clear and unambiguous, with a concise title. Each task should start with a title followed by new line and then have a precise technical specification for the developer.  I want the response to ONLY have actionable tickets withno additional fields, no general questins or comments, compact, precise. Each ticket should be separated one from eachother by \n@@@@\n")

	response, err := e.GPTClient.Chat(reply + "\n" + replyPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to process reply with GPT: %w", err)
	}

	// After receiving the response from GPT that contains the tasks:
	tasksParsed, err := parseTasksFromResponse(response)
	if err != nil {
		log.Printf("Error parsing tasks: %v", err)
		return nil, fmt.Errorf("failed to parse tasks: %w", err)
	}

	var createdTickets []*trello.Card

	// Get the list ID for the "Doing" column.
	doingListID, err := e.TrelloClient.GetListIDByName("Doing")
	if err != nil {
		return nil, fmt.Errorf("failed to get Doing list ID: %w", err)
	}

	// Iterate through the parsed tasks and create a technical ticket for each.
	for _, task := range tasksParsed {
		// Create the ticket using the title and description.
		techTicket, err := e.TrelloClient.CreateCard(task.Title, task.Description, doingListID)
		if err != nil {
			log.Printf("failed to create technical ticket for task '%s': %v", task.Title, err)
			continue
		}
		createdTickets = append(createdTickets, techTicket)
	}
	return createdTickets, nil
}

// parseTasksFromResponse takes the GPT response and extracts tasks.
// It splits the response on "\n@@@@\n" so that each block represents a task.
// The first line of each block is taken as the title and the rest as the description.
func parseTasksFromResponse(response string) ([]struct{ Title, Description string }, error) {
	var tasks []struct{ Title, Description string }

	// Split the response by the delimiter that separates tasks.
	taskBlocks := strings.Split(response, "\n@@@@\n")
	for _, block := range taskBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		// Split the block into lines.
		lines := strings.Split(block, "\n")
		if len(lines) == 0 {
			continue // Skip if no content is present.
		}

		// The first line is the title.
		title := strings.TrimSpace(lines[0])
		// The remaining lines (if any) are the description.
		description := ""
		if len(lines) > 1 {
			description = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}

		tasks = append(tasks, struct{ Title, Description string }{
			Title:       title,
			Description: description,
		})
	}

	return tasks, nil
}

// WaitForReply polls the ticket's comments until one is found that contains the required tag.
func (e *EngineeringManagerAIAgent) WaitForReply(card *trello.Card, requiredTag string) (string, error) {
	const pollInterval = 60 * time.Second
	const maxAttempts = 100 // Adjust as needed

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

// RespondToClarification generates a clarification response using ChatGPT and posts it as a comment.
func (e *EngineeringManagerAIAgent) RespondToClarification(ticket *trello.Card, clarificationRequest string, backend *BackendDeveloperAIAgent) (string, error) {
	// Build a prompt that includes the agent's instruction and the clarification request.
	prompt := fmt.Sprintf("%s\nPlease provide a detailed clarification for the following request: %s", e.Instruction, clarificationRequest)

	clarification, err := e.GPTClient.Chat(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate clarification: %w", err)
	}

	// Construct the response comment tagging the backend agent.
	responseComment := fmt.Sprintf("Engineering Manager Response: %s @%s", clarification, backend.Name)
	if err := e.WriteComment(ticket, responseComment); err != nil {
		return "", fmt.Errorf("failed to post clarification response: %w", err)
	}

	return clarification, nil
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
