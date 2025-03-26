package inmemory

import (
	"fmt"
	"sync"

	modelClient "github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/room"
)

// InMemoryRoom is a simple in-memory implementation of the Room interface.
type InMemoryRoom struct {
	mu     sync.Mutex
	agents map[string]participantWrapper
}

type participantWrapper struct {
	info        room.AgentInfo
	participant room.Participant
}

// NewInMemoryRoom creates a new in-memory room instance.
func NewInMemoryRoom() *InMemoryRoom {
	return &InMemoryRoom{
		agents: make(map[string]participantWrapper),
	}
}

// EnterRoom registers an agent (participant) into the room.
func (r *InMemoryRoom) EnterRoom(info room.AgentInfo, participant room.Participant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[info.Name]; exists {
		return fmt.Errorf("agent %s already registered", info.Name)
	}
	r.agents[info.Name] = participantWrapper{
		info:        info,
		participant: participant,
	}
	return nil
}

// LeaveRoom unregisters an agent from the room.
func (r *InMemoryRoom) LeaveRoom(agentName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[agentName]; !exists {
		return fmt.Errorf("agent %s not found", agentName)
	}
	delete(r.agents, agentName)
	return nil
}

// CheckRoom returns all registered agents' information.
func (r *InMemoryRoom) CheckRoom() ([]room.AgentInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var infos []room.AgentInfo
	for _, wrapper := range r.agents {
		infos = append(infos, wrapper.info)
	}
	return infos, nil
}

// Ask sends a question (slice of messages) from one agent to a specific agent.
func (r *InMemoryRoom) Ask(fromAgent, toAgent string, question []modelClient.Message) ([]modelClient.Message, error) {
	r.mu.Lock()
	wrapper, exists := r.agents[toAgent]
	r.mu.Unlock()
	if !exists {
		return nil, fmt.Errorf("agent %s not found", toAgent)
	}
	// Directly call the target participant's Answer method.
	return wrapper.participant.Answer(question)
}

// Shout broadcasts a question (slice of messages) from one agent to all agents.
func (r *InMemoryRoom) Shout(fromAgent string, question []modelClient.Message) (map[string][]modelClient.Message, error) {
	r.mu.Lock()
	agentsCopy := make(map[string]room.Participant)
	for name, wrapper := range r.agents {
		agentsCopy[name] = wrapper.participant
	}
	r.mu.Unlock()
	responses := make(map[string][]modelClient.Message)
	for name, participant := range agentsCopy {
		resp, err := participant.Answer(question)
		if err != nil {
			responses[name] = []modelClient.Message{{Role: "error", Content: fmt.Sprintf("error: %v", err)}}
		} else {
			responses[name] = resp
		}
	}
	return responses, nil
}
