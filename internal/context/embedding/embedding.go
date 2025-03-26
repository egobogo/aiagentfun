package embedding

// EmbeddingProvider defines an interface for computing embeddings from text.
type EmbeddingProvider interface {
	ComputeEmbedding(text string) ([]float64, error)
}
