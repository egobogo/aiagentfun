package hnsw

import (
	"errors"
	"math"
	"sync"

	"github.com/coder/hnsw"
	"github.com/egobogo/aiagents/internal/context"
)

// HNSWSimilaritySearcher implements a similarity searcher using the coder/hnsw generic graph.
type HNSWSimilaritySearcher struct {
	graph  *hnsw.Graph[string]            // Underlying HNSW graph.
	dim    int                            // Dimensionality of embeddings.
	memMap map[string]context.MemoryEntry // Map from memory ID to MemoryEntry.
	mu     sync.Mutex
}

// New creates a new HNSWSimilaritySearcher with the given embedding dimension.
func New(dim int) (*HNSWSimilaritySearcher, error) {
	// Create a new generic graph for string keys.
	g := hnsw.NewGraph[string]()
	return &HNSWSimilaritySearcher{
		graph:  g,
		dim:    dim,
		memMap: make(map[string]context.MemoryEntry),
	}, nil
}

// IndexMemory adds a memory entry to the HNSW graph.
// It expects that mem.Embedding has length equal to the dimension.
func (s *HNSWSimilaritySearcher) IndexMemory(mem context.MemoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(mem.Embedding) != s.dim {
		return errors.New("embedding dimension mismatch")
	}

	// Convert the embedding to []float32 and create a node.
	node := hnsw.MakeNode(mem.ID, float32Slice(mem.Embedding))
	// Add the node to the graph (Add returns no value).
	s.graph.Add(node)
	// Save the memory entry in our map.
	s.memMap[mem.ID] = mem

	return nil
}

// Search performs a similarity search for the query embedding, returning up to k matching memories
// with cosine similarity above the threshold.
// We compute cosine similarity as: similarity = 1.0 - cosineSimilarity(query, node.Value)
func (s *HNSWSimilaritySearcher) Search(query []float64, k int, threshold float64) ([]context.MemoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(query) != s.dim {
		return nil, errors.New("query embedding dimension mismatch")
	}

	// Convert query to []float32.
	q := float32Slice(query)
	neighbors := s.graph.Search(q, k)

	var matches []context.MemoryEntry
	for _, node := range neighbors {
		// Compute cosine similarity between the query and the node's vector stored in Value.
		sim := 1.0 - cosineSimilarity(q, node.Value)
		if sim >= threshold {
			if mem, ok := s.memMap[node.Key]; ok {
				matches = append(matches, mem)
			}
		}
	}
	return matches, nil
}

// float32Slice converts a slice of float64 to []float32.
func float32Slice(input []float64) []float32 {
	out := make([]float32, len(input))
	for i, v := range input {
		out[i] = float32(v)
	}
	return out
}

// cosineSimilarity computes the cosine similarity between two []float32 vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
