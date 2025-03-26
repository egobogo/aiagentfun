// File: internal/context/embedding/openai/embedding.go
package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// EmbeddingProvider defines the interface for computing embeddings.
type EmbeddingProvider interface {
	ComputeEmbedding(text string) ([]float64, error)
}

// OpenAIEmbeddingProvider implements EmbeddingProvider using direct HTTP calls to OpenAI's API.
type OpenAIEmbeddingProvider struct {
	apiKey    string
	modelName string
	endpoint  string
}

// NewOpenAIEmbeddingProvider creates a new OpenAIEmbeddingProvider instance.
func NewOpenAIEmbeddingProvider(apiKey, modelName string) *OpenAIEmbeddingProvider {
	return &OpenAIEmbeddingProvider{
		apiKey:    apiKey,
		modelName: modelName,
		// OpenAI embeddings endpoint.
		endpoint: "https://api.openai.com/v1/embeddings",
	}
}

// embeddingRequest represents the JSON payload sent to the OpenAI API.
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingData represents one result in the API response.
type embeddingData struct {
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
	Object    string    `json:"object"`
}

// embeddingResponse represents the full response from the OpenAI API.
type embeddingResponse struct {
	Data   []embeddingData `json:"data"`
	Model  string          `json:"model"`
	Object string          `json:"object"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// ComputeEmbedding calls the OpenAI API and returns the embedding vector for the provided text.
func (p *OpenAIEmbeddingProvider) ComputeEmbedding(text string) ([]float64, error) {
	reqBody := embeddingRequest{
		Model: p.modelName,
		Input: []string{text},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(bodyBytes, &embResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal API response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// We requested a single input so we return the first embedding.
	return embResp.Data[0].Embedding, nil
}
