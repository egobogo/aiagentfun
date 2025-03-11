// internal/agent/agent.go
package agent

import (
	"fmt"
	"log"
	"strings"

	"github.com/egobogo/aiagents/internal/gitrepo"
	chatgpt "github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/trello"
)

// AIAgent is the base type that provides common functionality for all agents.
type AIAgent struct {
	Name         string
	TrelloClient *trello.TrelloClient
	GitClient    *gitrepo.GitClient
	GPTClient    *chatgpt.ChatGPTClient
}

// TicketBelongsToMe checks if a Trello card is assigned to this agent.
// (This is a simplistic check comparing agent's Name with each assigned member ID.)
func (a *AIAgent) TicketBelongsToMe(card *trello.Card) bool {
	// For each memberID on the card, fetch the member details.
	for _, memberID := range card.IDMembers {
		member, err := a.TrelloClient.GetMember(memberID)
		if err != nil {
			continue // or log the error
		}
		// Compare the member's full name or username with the agent's name.
		if strings.EqualFold(member.Username, a.Name) {
			return true
		}
	}
	return false
}

func (a *AIAgent) ChangeTicketColumn(card *trello.Card, newListID string) error {
	myCard := trello.WrapCard(card)
	return myCard.Move(newListID)
}

// ChangeTicketAssignee updates a card's assignee.
func (a *AIAgent) ChangeTicketAssignee(card *trello.Card, newAssignee string) error {
	myCard := trello.WrapCard(card) // Wrap the card to get access to helper methods.
	return myCard.AssignMember(newAssignee)
}

// WriteComment adds a comment to a card.
func (a *AIAgent) WriteComment(card *trello.Card, comment string) error {
	myCard := trello.WrapCard(card)
	// Use PostComment instead of AddComment to avoid the conflict.
	return myCard.PostComment(comment, a.TrelloClient)
}

func (a *AIAgent) ReadComments(card *trello.Card) ([]string, error) {
	myCard := trello.WrapCard(card)
	return myCard.GetComments(a.TrelloClient)
}

// GetTaggedComments returns comments that mention the agent (e.g. "@AgentName").
func (a *AIAgent) GetTaggedComments(card *trello.Card) ([]string, error) {
	comments, err := a.ReadComments(card)
	if err != nil {
		return nil, err
	}
	var tagged []string
	tag := "@" + a.Name
	for _, c := range comments {
		if strings.Contains(c, tag) {
			tagged = append(tagged, c)
		}
	}
	return tagged, nil
}

// ReadAllGitFiles delegates to the Git client to read the repository files.
func (a *AIAgent) ReadAllGitFiles() (map[string]string, error) {
	return a.GitClient.ReadAllFiles()
}

// WriteToGit writes content to a file in the repository via the Git client.
func (a *AIAgent) WriteToGit(fileName string, content []byte) error {
	return a.GitClient.WriteFile(fileName, content)
}

// StartRoutine reminds the agent of its goal and observes the Git repository.
func (a *AIAgent) StartRoutine() error {
	log.Printf("Agent %s starting routine: observing repository...", a.Name)
	files, err := a.ReadAllGitFiles()
	if err != nil {
		return err
	}
	for path := range files {
		log.Printf("Found file: %s", path)
	}
	return nil
}

// GetAssignedTickets fetches all Trello cards on the board and returns those assigned to the agent.
func (a *AIAgent) GetAssignedTickets() ([]*trello.Card, error) {
	board, err := a.TrelloClient.GetBoard()
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}
	// Retrieve all cards; passing nil for options.
	cards, err := board.GetCards(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	var assigned []*trello.Card
	for _, card := range cards {
		if a.TicketBelongsToMe(card) {
			assigned = append(assigned, card)
		}
	}
	return assigned, nil
}
