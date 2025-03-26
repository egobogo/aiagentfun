package board

// Member represents a board member.
type Member struct {
	ID   string
	Name string
}

// Comment represents a comment on a card.
type Comment struct {
	Text   string
	Member *Member
}

// Attachment represents an attachment on a card.
type Attachment struct {
	ID   string
	Name string
	URL  string
}

// Card defines the operations available on a card.
type Card interface {
	// GetName returns the name of the card.
	GetName() string
	// ChangeName sets a new name for the card.
	ChangeName(newName string) error
	// GetURL returns the URL of the card on the board.
	GetURL() string
	// GetList returns the current list (column) that the card is in.
	GetList() (List, error)
	// Move moves the card to another list identified by its name.
	Move(newListName string) error
	// GetAssignedMembers returns all members to whom the card is assigned.
	GetAssignedMembers() ([]Member, error)
	// AssignTo assigns the card to a member by name.
	AssignTo(userName string) error
	// UnassignFrom removes a member assignment from the card.
	UnassignFrom(userName string) error
	// ReadComments retrieves all comments on the card.
	ReadComments() ([]Comment, error)
	// WriteComment writes a comment to the card.
	WriteComment(comment string) error
	// GetAttachments retrieves all attachments on the card.
	GetAttachments() ([]Attachment, error)
	// AddAttachment adds a new attachment to the card.
	AddAttachment(attachment Attachment) error
}

// List defines operations for a board column (list).
type List interface {
	// GetName returns the name of the list.
	GetName() string
	// GetID returns the unique identifier of the list.
	GetID() string
}

// Board defines the board-level operations.
type Board interface {
	// GetName returns the name of the board.
	GetName() string
	// GetURL returns the URL of the board.
	GetURL() string
	// GetMembers retrieves all members of the board.
	GetMembers() ([]Member, error)
	// GetCards retrieves all cards on the board.
	GetCards() ([]Card, error)
	// CreateCard creates a new card on the board.
	CreateCard(name, description, listName string) (Card, error)
	// GetCardsAssignedTo returns all cards assigned to a specific member.
	GetCardsAssignedTo(userName string) ([]Card, error)
	// GetCardsFromList returns all cards in a specific list.
	GetCardsFromList(listName string) ([]Card, error)
	// GetLists retrieves all lists (columns) on the board.
	GetLists() ([]List, error)
}

// BoardClient is the main dependency injection interface for board connectors.
type BoardClient interface {
	Board
}
