// internal/agent/aiagent1.go

package agent

import (
	"fmt"
	"log"

	"github.com/egobogo/aiagents/internal/trello"

	trelloAPI "github.com/adlio/trello"
)

type Agent struct {
	name         string
	role         string
	trelloUser   string // The corresponding Trello username or member ID
	trelloClient *trello.TrelloClient
}

// NewAgent creates a new agent with the given name, role, Trello user, and TrelloClient
func NewAgent(
	name string,
	role string,
	trelloUser string,
	trelloClient *trello.TrelloClient,
) *Agent {
	return &Agent{
		name:         name,
		role:         role,
		trelloUser:   trelloUser,
		trelloClient: trelloClient,
	}
}

// Example method that retrieves a board and prints its lists
func (a *Agent) CheckBoard(boardID string) error {
	board, err := a.trelloClient.GetBoard(boardID)
	if err != nil {
		return fmt.Errorf("failed to get board: %w", err)
	}

	// Retrieve the lists in the board
	lists, err := board.GetLists(trelloAPI.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get lists: %w", err)
	}

	log.Printf("Agent %s (%s) is checking board: %s\n", a.name, a.role, board.Name)
	for _, l := range lists {
		log.Printf("List: %s\n", l.Name)
	}
	return nil
}

// ListColumns retrieves the Trello board columns (lists) and returns their names
func (a *Agent) ListColumns(boardID string) ([]string, error) {
	lists, err := a.trelloClient.GetBoardLists(boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve board lists: %w", err)
	}

	var listNames []string
	for _, l := range lists {
		listNames = append(listNames, l.Name)
	}
	log.Printf("Agent %s found columns: %v\n", a.name, listNames)

	return listNames, nil
}
