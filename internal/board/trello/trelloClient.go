// internal/board/trello/trelloClient/trelloClient.go
package trelloClient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/adlio/trello"
	bc "github.com/egobogo/aiagents/internal/board"
)

// -------------------------
// Concrete TrelloBoardClient
// -------------------------

// TrelloClient implements the bc.BoardClient interface using the adlio/trello library.
type TrelloClient struct {
	Client  *trello.Client
	BoardID string
	APIKey  string
	Token   string
}

// NewTrelloClient constructs a new TrelloClient.
func NewTrelloClient(apiKey, token, boardID string) *TrelloClient {
	client := trello.NewClient(apiKey, token)
	return &TrelloClient{
		Client:  client,
		BoardID: boardID,
		APIKey:  apiKey,
		Token:   token,
	}
}

func (tc *TrelloClient) GetName() string {
	b, err := tc.Client.GetBoard(tc.BoardID, trello.Defaults())
	if err != nil {
		return ""
	}
	return b.Name
}

func (tc *TrelloClient) GetURL() string {
	b, err := tc.Client.GetBoard(tc.BoardID, trello.Defaults())
	if err != nil {
		return ""
	}
	return b.ShortURL
}

func (tc *TrelloClient) GetMembers() ([]bc.Member, error) {
	b, err := tc.Client.GetBoard(tc.BoardID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}
	members, err := b.GetMembers(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get board members: %w", err)
	}
	var result []bc.Member
	for _, m := range members {
		result = append(result, bc.Member{
			ID:   m.ID,
			Name: m.FullName,
		})
	}
	return result, nil
}

func (tc *TrelloClient) GetLists() ([]bc.List, error) {
	b, err := tc.Client.GetBoard(tc.BoardID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}
	lists, err := b.GetLists(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get lists: %w", err)
	}
	var result []bc.List
	for _, l := range lists {
		result = append(result, &TrelloList{
			ID:   l.ID,
			Name: l.Name,
		})
	}
	return result, nil
}

// CreateCard creates a new card on the board given a name, description, and target list name.
func (tc *TrelloClient) CreateCard(name, description, listName string) (bc.Card, error) {
	// Retrieve board lists.
	lists, err := tc.GetLists()
	if err != nil {
		return nil, fmt.Errorf("failed to get lists: %w", err)
	}

	var targetListID string
	var targetList bc.List
	for _, l := range lists {
		if strings.EqualFold(l.GetName(), listName) {
			targetListID = l.GetID()
			targetList = l
			break
		}
	}
	if targetListID == "" {
		return nil, fmt.Errorf("list %s not found", listName)
	}

	newCard := trello.Card{
		Name: name,
		Desc: description,
	}
	args := trello.Arguments{"idList": targetListID}
	if err := tc.Client.CreateCard(&newCard, args); err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	// Construct a concrete TrelloCard that implements bc.Card.
	tcCard := &TrelloCard{
		ID:          newCard.ID,
		CardName:    name,
		Description: description,
		URL:         newCard.ShortURL,
		List:        targetList,
		BoardClient: tc,
		Client:      tc.Client,
	}
	return tcCard, nil
}

func (tc *TrelloClient) GetCards() ([]bc.Card, error) {
	b, err := tc.Client.GetBoard(tc.BoardID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get board: %w", err)
	}
	cards, err := b.GetCards(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	var result []bc.Card
	for _, c := range cards {
		tcCard := &TrelloCard{
			ID:          c.ID,
			CardName:    c.Name,
			Description: c.Desc,
			URL:         c.ShortURL,
			// List, BoardClient and Client should be set appropriately if needed.
		}
		result = append(result, tcCard)
	}
	return result, nil
}

func (tc *TrelloClient) GetCardsAssignedTo(userName string) ([]bc.Card, error) {
	allCards, err := tc.GetCards()
	if err != nil {
		return nil, err
	}
	var result []bc.Card
	for _, card := range allCards {
		members, err := card.GetAssignedMembers()
		if err != nil {
			continue
		}
		for _, m := range members {
			if strings.EqualFold(m.Name, userName) {
				result = append(result, card)
				break
			}
		}
	}
	return result, nil
}

func (tc *TrelloClient) GetCardsFromList(listName string) ([]bc.Card, error) {
	allCards, err := tc.GetCards()
	if err != nil {
		return nil, err
	}
	var result []bc.Card
	for _, card := range allCards {
		list, err := card.GetList()
		if err != nil {
			continue
		}
		if strings.EqualFold(list.GetName(), listName) {
			result = append(result, card)
		}
	}
	return result, nil
}

// -------------------------
// Concrete TrelloList Implementation
// -------------------------

type TrelloList struct {
	ID   string
	Name string
}

func (tl *TrelloList) GetName() string {
	return tl.Name
}

func (tl *TrelloList) GetID() string {
	return tl.ID
}

// -------------------------
// Concrete TrelloCard Implementation
// -------------------------

type TrelloCard struct {
	ID          string
	CardName    string
	Description string
	URL         string
	// The list the card belongs to.
	List bc.List
	// References to the underlying Trello client and board client.
	BoardClient *TrelloClient
	Client      *trello.Client
}

func (tc *TrelloCard) GetName() string {
	return tc.CardName
}

func (tc *TrelloCard) ChangeName(newName string) error {
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	args := trello.Arguments{"name": newName}
	if err := tCard.Update(args); err != nil {
		return err
	}
	tc.CardName = newName
	return nil
}

func (tc *TrelloCard) GetURL() string {
	return tc.URL
}

func (tc *TrelloCard) GetList() (bc.List, error) {
	if tc.List == nil {
		return nil, fmt.Errorf("list not set for card")
	}
	return tc.List, nil
}

func (tc *TrelloCard) Move(newListName string) error {
	lists, err := tc.BoardClient.GetLists()
	if err != nil {
		return err
	}
	var targetID string
	for _, l := range lists {
		if strings.EqualFold(l.GetName(), newListName) {
			targetID = l.GetID()
			break
		}
	}
	if targetID == "" {
		return fmt.Errorf("list %s not found", newListName)
	}
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	args := trello.Arguments{"idList": targetID}
	return tCard.Update(args)
}

func (tc *TrelloCard) GetAssignedMembers() ([]bc.Member, error) {
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	var members []bc.Member
	for _, mID := range tCard.IDMembers {
		member, err := tc.Client.GetMember(mID, trello.Defaults())
		if err != nil {
			continue
		}
		members = append(members, bc.Member{
			ID:   member.ID,
			Name: member.FullName,
		})
	}
	return members, nil
}

func (tc *TrelloCard) AssignTo(userName string) error {
	b, err := tc.Client.GetBoard(tc.BoardClient.BoardID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get board: %w", err)
	}
	members, err := b.GetMembers(trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get board members: %w", err)
	}
	var targetID string
	for _, m := range members {
		if strings.EqualFold(m.Username, userName) || strings.EqualFold(m.FullName, userName) {
			targetID = m.ID
			break
		}
	}
	if targetID == "" {
		return fmt.Errorf("member %s not found", userName)
	}
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	args := trello.Arguments{"idMembers": targetID}
	return tCard.Update(args)
}

func (tc *TrelloCard) UnassignFrom(userName string) error {
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	current := tCard.IDMembers
	b, err := tc.Client.GetBoard(tc.BoardClient.BoardID, trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get board: %w", err)
	}
	members, err := b.GetMembers(trello.Defaults())
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}
	var targetID string
	for _, m := range members {
		if strings.EqualFold(m.Username, userName) || strings.EqualFold(m.FullName, userName) {
			targetID = m.ID
			break
		}
	}
	if targetID == "" {
		return fmt.Errorf("member %s not found", userName)
	}
	var newMembers []string
	for _, id := range current {
		if id != targetID {
			newMembers = append(newMembers, id)
		}
	}
	args := trello.Arguments{"idMembers": strings.Join(newMembers, ",")}
	return tCard.Update(args)
}

func (tc *TrelloCard) ReadComments() ([]bc.Comment, error) {
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	actions, err := tCard.GetActions(map[string]string{"filter": "commentCard"})
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	var comments []bc.Comment
	for _, a := range actions {
		// Use a.Data.Text instead of indexing a.Data.
		text := a.Data.Text
		if text == "" {
			continue
		}
		comments = append(comments, bc.Comment{
			Text: text,
		})
	}
	return comments, nil
}

func (tc *TrelloCard) WriteComment(comment string) error {
	endpoint := fmt.Sprintf("https://api.trello.com/1/cards/%s/actions/comments", tc.ID)
	values := url.Values{}
	values.Set("text", comment)
	values.Set("key", tc.BoardClient.APIKey)
	values.Set("token", tc.BoardClient.Token)

	resp, err := http.PostForm(endpoint, values)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to post comment, status: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (tc *TrelloCard) GetAttachments() ([]bc.Attachment, error) {
	tCard, err := tc.Client.GetCard(tc.ID, trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	atts, err := tCard.GetAttachments(trello.Defaults())
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	var result []bc.Attachment
	for _, a := range atts {
		result = append(result, bc.Attachment{
			ID:   a.ID,
			Name: a.Name,
			URL:  a.URL,
		})
	}
	return result, nil
}

func (tc *TrelloCard) AddAttachment(attachment bc.Attachment) error {
	endpoint := fmt.Sprintf("https://api.trello.com/1/cards/%s/attachments", tc.ID)
	query := fmt.Sprintf("url=%s&name=%s&key=%s&token=%s",
		attachment.URL, attachment.Name, tc.BoardClient.APIKey, tc.BoardClient.Token)
	url := endpoint + "?" + query
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to add attachment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to add attachment, status: %d", resp.StatusCode)
	}
	return nil
}
