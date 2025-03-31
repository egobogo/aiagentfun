// internal/context/inmemory/inmemory.go
package inmemory

import (
	"fmt"
	"sync"
	"time"

	"github.com/egobogo/aiagents/internal/context"
	"github.com/google/uuid"

	"github.com/egobogo/aiagents/internal/context/embedding"  // EmbeddingProvider interface
	"github.com/egobogo/aiagents/internal/context/similarity" // SimilaritySearcher interface
)

// MemoryManager is a concrete implementation of ContextStorage.
type InMemoryContextStorage struct {
	mu          sync.RWMutex
	hotContext  string
	coldStorage map[string]context.MemoryEntry

	embProvider embedding.EmbeddingProvider   // Dependency to compute embeddings.
	simSearcher similarity.SimilaritySearcher // Dependency to index and search embeddings.
}

// NewInMemoryContextStorage constructs a new instance of InMemoryContextStorage with the provided
// EmbeddingProvider and SimilaritySearcher.
func NewInMemoryContextStorage(embProvider embedding.EmbeddingProvider, simSearcher similarity.SimilaritySearcher) *InMemoryContextStorage {
	return &InMemoryContextStorage{
		coldStorage: make(map[string]context.MemoryEntry),
		hotContext:  "",
		embProvider: embProvider,
		simSearcher: simSearcher,
	}
}

// MemoryExists returns true if a memory with the given ID is present in coldStorage.
func (s *InMemoryContextStorage) MemoryExists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.coldStorage[id]
	return exists
}

// FilterRelatedMemories iterates over the provided new memories, searches for related existing memories,
// and returns a deduplicated slice of related MemoryEntry.
func (s *InMemoryContextStorage) FilterRelatedMemories(newMems []context.EasyMemory) []context.MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resultsMap := make(map[string]context.MemoryEntry)
	for _, nm := range newMems {
		// Search for related memories based on the content of the new memory.
		related := s.SearchMemories(nm.Content)
		for _, mem := range related {
			// If this memory is not already in the results, add it.
			if _, exists := resultsMap[mem.ID]; !exists {
				resultsMap[mem.ID] = mem
			}
		}
	}
	// Convert the map to a slice.
	results := make([]context.MemoryEntry, 0, len(resultsMap))
	for _, mem := range resultsMap {
		results = append(results, mem)
	}
	return results
}

// Remember adds a new memory record based on an EasyMemory input.
// It computes the embedding via the injected EmbeddingProvider,
// assigns a unique ID and current timestamp, stores it in cold storage,
// and indexes it via the injected SimilaritySearcher.
func (s *InMemoryContextStorage) Remember(easyMem context.EasyMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a new MemoryEntry.
	entry := context.MemoryEntry{
		ID:         uuid.New().String(),
		Category:   easyMem.Category,
		Content:    easyMem.Content,
		Importance: easyMem.Importance,
		Timestamp:  time.Now(),
	}

	// Compute the embedding.
	embedding, err := s.embProvider.ComputeEmbedding(easyMem.Content)
	if err != nil {
		return fmt.Errorf("failed to compute embedding: %w", err)
	}
	entry.Embedding = embedding

	// Store in cold storage.
	s.coldStorage[entry.ID] = entry

	// Index the new memory.
	if err := s.simSearcher.IndexMemory(entry); err != nil {
		return fmt.Errorf("failed to index memory: %w", err)
	}

	return nil
}

// SetContext updates the hot context summary.
func (m *InMemoryContextStorage) SetContext(context string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hotContext = context
	return nil
}

// GetContext retrieves the current hot context summary.
func (m *InMemoryContextStorage) GetContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hotContext
}

// GetMemories returns the entire cold storage as a pretty-printed JSON string.
func (m *InMemoryContextStorage) GetMemories() []context.MemoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var memorySlice []context.MemoryEntry

	for _, mem := range m.coldStorage {
		memorySlice = append(memorySlice, mem)
	}
	return memorySlice
}

// SearchMemories computes an embedding for the query text and uses the injected SimilaritySearcher
// to retrieve similar memories.
func (s *InMemoryContextStorage) SearchMemories(query string) []context.MemoryEntry {
	emb, err := s.embProvider.ComputeEmbedding(query)
	if err != nil {
		return nil
	}
	results, err := s.simSearcher.Search(emb, 10, 0.1)
	if err != nil {
		return nil
	}
	// Remove embeddings from each memory.
	for i := range results {
		results[i].Embedding = nil
	}
	return results
}

// Forget removes the memory with the given ID from cold storage.
func (s *InMemoryContextStorage) Forget(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.coldStorage[id]; !exists {
		return fmt.Errorf("memory with ID %s not found", id)
	}

	// Remove from the internal map.
	delete(s.coldStorage, id)

	return nil
}
