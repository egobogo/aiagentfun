package similarity

import "github.com/egobogo/aiagents/internal/context"

// SimilaritySearcher defines an interface for indexing memory entries and searching them by embedding similarity.
type SimilaritySearcher interface {
	// IndexMemory adds a memory entry to the search index.
	IndexMemory(mem context.MemoryEntry) error
	// Search takes a query embedding and returns matching memory entries whose similarity is above threshold.
	Search(query []float64, k int, threshold float64) ([]context.MemoryEntry, error)
}
