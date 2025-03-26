package room

import modelClient "github.com/egobogo/aiagents/internal/model"

// AgentInfo holds basic identifying information for a participant.
type AgentInfo struct {
	Name string // Unique identifier.
	Role string // e.g. "Developer", "Manager", etc.
}

// Participant is the interface that an agent must implement to participate in the room.
type Participant interface {
	// Answer takes a slice of messages (the question and its context) and returns a slice of answer messages.
	Answer(question []modelClient.Message) ([]modelClient.Message, error)
}

// Room defines an abstraction for inter-agent communication.
type Room interface {
	// EnterRoom registers an agent into the room.
	EnterRoom(info AgentInfo, participant Participant) error
	// LeaveRoom unregisters an agent from the room.
	LeaveRoom(agentName string) error
	// CheckRoom returns a list of all registered agents.
	CheckRoom() ([]AgentInfo, error)
	// Ask sends a question (as a slice of messages) from one agent to a specific agent.
	Ask(fromAgent, toAgent string, question []modelClient.Message) ([]modelClient.Message, error)
	// Shout broadcasts a question (as a slice of messages) from one agent to all agents.
	Shout(fromAgent string, question []modelClient.Message) (map[string][]modelClient.Message, error)
}
