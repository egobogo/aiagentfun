package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/egobogo/aiagents/internal/context/embedding/openai"
	"github.com/joho/godotenv"
)

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		log.Fatalf("vectors must be the same length: got %d and %d", len(a), len(b))
	}
	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func main() {
	// Load environment variables from .env (if present).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Get the OpenAI API key from environment variables.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}

	// Create an OpenAI embedding provider instance using your integration.
	embProvider := openai.NewOpenAIEmbeddingProvider(apiKey, "text-embedding-ada-002")

	// Define the two example sentences.
	sentence1 := "The architecture is modular and event-driven, ensuring each module can develop independently with clear interfaces. Integrates seamlessly with external tools like Notion and Trello, enhancing adaptability and scalability."
	sentence2 := "Architecture Modular and event-driven architecture to ensure independent development and seamless integration with tools like Notion and Trello"

	// Compute embeddings for both sentences.
	emb1, err := embProvider.ComputeEmbedding(sentence1)
	if err != nil {
		log.Fatalf("Failed to compute embedding for sentence1: %v", err)
	}
	emb2, err := embProvider.ComputeEmbedding(sentence2)
	if err != nil {
		log.Fatalf("Failed to compute embedding for sentence2: %v", err)
	}

	// Calculate cosine similarity and distance.
	similarity := cosineSimilarity(emb1, emb2)
	distance := 1 - similarity

	fmt.Printf("Cosine similarity: %f\n", similarity)
	fmt.Printf("Cosine distance: %f\n", distance)
}
