// internal/agent/aiagent1.go

package agent

import (
	"fmt"
	"log"

	"github.com/egobogo/aiagents/internal/gitrepo"
	"github.com/egobogo/aiagents/internal/trello"

	trelloAPI "github.com/adlio/trello"
)

type Agent struct {
	name         string
	role         string
	trelloUser   string // The corresponding Trello username or member ID
	trelloClient *trello.TrelloClient
	gitClient    *gitrepo.GitClient
}

// NewAgent creates a new Agent instance with both Trello and Git clients.
func NewAgent(name, role, trelloUser string, trelloClient *trello.TrelloClient, gitClient *gitrepo.GitClient) *Agent {
	return &Agent{
		name:         name,
		role:         role,
		trelloUser:   trelloUser,
		trelloClient: trelloClient,
		gitClient:    gitClient,
	}
}

// Example method that retrieves a board and prints its lists
func (a *Agent) CheckBoard() error {
	board, err := a.trelloClient.GetBoard()
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
func (a *Agent) ListColumns() ([]string, error) {
	lists, err := a.trelloClient.GetBoardLists()
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

// ListMyTickets fetches all cards from the specified board and
// prints out only those assigned to this agent's Trello user.
func (a *Agent) ListMyTickets() error {
	// 1. Get the Trello board
	board, err := a.trelloClient.GetBoard()
	if err != nil {
		return fmt.Errorf("failed to get board: %w", err)
	}

	// 2. Look up the Trello Member for this agent’s user
	member, err := a.trelloClient.GetMember(a.trelloUser)
	if err != nil {
		return fmt.Errorf("failed to find trello user %q: %w", a.trelloUser, err)
	}

	// 3. Get all the cards on the board
	cards, err := board.GetCards(nil)
	if err != nil {
		return fmt.Errorf("failed to get cards: %w", err)
	}

	log.Printf("Agent %s is listing tickets assigned to %s\n", a.name, member.Username)
	found := false

	// 4. Filter for cards assigned to the agent
	for _, card := range cards {
		// card.IDMembers is a slice of member IDs assigned to this card
		for _, assignedMemberID := range card.IDMembers {
			if assignedMemberID == member.ID {
				// Print or collect more details here
				log.Printf("Ticket Title: %s\n", card.Name)
				log.Printf("Ticket Description: %s\n", card.Desc)
				log.Printf("Ticket URL: %s\n", card.URL)

				// Optionally, check for attachments, labels, or checklists
				// ...
				found = true
				break
			}
		}
	}

	if !found {
		log.Println("No cards assigned to this user on the board.")
	}
	return nil
}

// CommitAndPush uses the GitClient to commit all changes with a message and then push them.
func (a *Agent) CommitAndPush(commitMessage, gitUsername, gitToken string) error {
	// Use agent's name as the commit author.
	if err := a.gitClient.CommitChanges(commitMessage, a.name, "aiagent@example.com"); err != nil {
		return err
	}

	if err := a.gitClient.PushChanges(gitUsername, gitToken); err != nil {
		return err
	}

	log.Println("Repository updated successfully.")
	return nil
}

// ListRepositoryFiles reads and prints the content of all files in the repository.
func (a *Agent) ListRepositoryFiles() error {
	files, err := a.gitClient.ReadAllFiles()
	if err != nil {
		return fmt.Errorf("failed to read repository files: %w", err)
	}

	log.Println("Repository files and contents:")
	for path, content := range files {
		log.Printf("File: %s\nContent:\n%s\n", path, content)
	}
	return nil
}
