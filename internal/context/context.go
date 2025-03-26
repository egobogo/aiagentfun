package context

import "time"

// MemoryEntry represents a unit of knowledge.
type MemoryEntry struct {
	ID         string    `json:"ID"`                   // Unique ID of the memory.
	Category   string    `json:"category"`             // e.g. "Architecture", "Performance", etc.
	Content    string    `json:"content"`              // The actual knowledge detail or summary.
	Timestamp  time.Time `json:"timestamp"`            // When this entry was added.
	Importance int       `json:"importance,omitempty"` // Relative importance score.
	Embedding  []float64 `json:"embedding,omitempty"`  // Embedding for similarity search.
}

// EasyMemory is a simplified memory structure.
type EasyMemory struct {
	Category   string `json:"category"`   // e.g. "Architecture", "Performance", etc.
	Content    string `json:"content"`    // The actual knowledge detail or summary.
	Importance int    `json:"importance"` // Relative importance score.
}

// ContextStorage defines operations for storing and managing conversation context.
type ContextStorage interface {
	Remember(me EasyMemory) error
	Forget(ID string) error
	SetContext(summary string) error
	GetContext() string
	GetMemories() (string, error)
	SearchMemories(query string) []MemoryEntry
	FilterRelatedMemories(newMems []EasyMemory) []MemoryEntry
	MemoryExists(id string) bool
}
