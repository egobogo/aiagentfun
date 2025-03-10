// internal/trello/trello.go

package trello

import (
	"github.com/adlio/trello"
)

// TrelloClient is a small wrapper around adlio/trello.Client
type TrelloClient struct {
	client  *trello.Client
	BoardID string
}

// NewTrelloClient creates a new Trello client using the provided API key and token
func NewTrelloClient(apiKey, token, boardID string) *TrelloClient {
	return &TrelloClient{
		client:  trello.NewClient(apiKey, token),
		BoardID: boardID,
	}
}

// GetBoard fetches a board by its shortLink or ID
func (tc *TrelloClient) GetBoard() (*trello.Board, error) {
	return tc.client.GetBoard(tc.BoardID, nil)
}

// GetMember retrieves a Trello member (user) by username, member ID, or email
func (tc *TrelloClient) GetMember(usernameOrID string) (*trello.Member, error) {
	return tc.client.GetMember(usernameOrID, nil)
}

// GetBoardLists returns all Trello lists (columns) from a given board
func (tc *TrelloClient) GetBoardLists() ([]*trello.List, error) {
	board, err := tc.client.GetBoard(tc.BoardID, nil)
	if err != nil {
		return nil, err
	}
	lists, err := board.GetLists(nil)
	if err != nil {
		return nil, err
	}
	return lists, nil
}

// (Add more methods as needed, e.g. GetLists, CreateCard, etc.)
