// internal/trello/trello.go

package trello

import (
	"github.com/adlio/trello"
)

// TrelloClient is a small wrapper around adlio/trello.Client
type TrelloClient struct {
	client *trello.Client
}

// NewTrelloClient creates a new Trello client using the provided API key and token
func NewTrelloClient(apiKey, token string) *TrelloClient {
	return &TrelloClient{
		client: trello.NewClient(apiKey, token),
	}
}

// GetBoard fetches a board by its shortLink or ID
func (tc *TrelloClient) GetBoard(boardID string) (*trello.Board, error) {
	return tc.client.GetBoard(boardID, trello.Defaults())
}

// GetBoardLists returns all Trello lists (columns) from a given board
func (tc *TrelloClient) GetBoardLists(boardID string) ([]*trello.List, error) {
	board, err := tc.client.GetBoard(boardID, trello.Defaults())
	if err != nil {
		return nil, err
	}
	lists, err := board.GetLists(trello.Defaults())
	if err != nil {
		return nil, err
	}
	return lists, nil
}

// (Add more methods as needed, e.g. GetLists, CreateCard, etc.)
