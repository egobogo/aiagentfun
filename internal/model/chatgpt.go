package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// ChatGPTClient is a simple client to interact with the OpenAI Chat API.
type ChatGPTClient struct {
	APIKey string
	Model  string
}

// Message represents a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request payload for the ChatGPT API.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponseChoice holds one response choice from the API.
type ChatResponseChoice struct {
	Message Message `json:"message"`
}

// ChatResponse is the API response structure.
type ChatResponse struct {
	Choices []ChatResponseChoice `json:"choices"`
}

// NewChatGPTClient creates a new ChatGPTClient with the given model.
// If an empty model is provided, it defaults to "o3-mini-high".
func NewChatGPTClient(model string) *ChatGPTClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if model == "" {
		model = "o3-mini-high"
	}
	return &ChatGPTClient{
		APIKey: apiKey,
		Model:  model,
	}
}

// Chat sends a prompt to the ChatGPT API and returns the response.
func (c *ChatGPTClient) Chat(prompt string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	reqBody := ChatRequest{
		Model: c.Model,
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.7,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseData, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("non-200 status code: %d, response: %s", resp.StatusCode, string(responseData))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no choices in response")
}

// ChatWithMessages sends a slice of messages and returns the response.
func (c *ChatGPTClient) ChatWithMessages(messages []Message) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	reqBody := ChatRequest{
		Model:       c.Model,
		Messages:    messages,
		Temperature: 0.7,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseData, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("non-200 status code: %d, response: %s", resp.StatusCode, string(responseData))
	}
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no choices in response")
}
