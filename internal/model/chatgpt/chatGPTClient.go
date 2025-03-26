package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	model "github.com/egobogo/aiagents/internal/model"
)

// ChatGPTClient implements the ModelClient interface using the OpenAI Chat API.
type ChatGPTClient struct {
	APIKey      string
	Model       string
	Temperature float64
}

// NewChatGPTClient creates a new ChatGPTClient.
func NewChatGPTClient(apiKey, model string) *ChatGPTClient {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &ChatGPTClient{
		APIKey:      apiKey,
		Model:       model,
		Temperature: 0.7,
	}
}

// Chat sends a prompt and returns the response as a string.
func (c *ChatGPTClient) Chat(prompt string) (string, error) {
	reqBody := model.ChatRequest{
		Model:       c.Model,
		Input:       []model.Message{{Role: "user", Content: prompt}},
		Temperature: c.Temperature,
		Text:        nil,
	}
	return c.ChatAdvanced(reqBody)
}

// ChatAdvanced sends a ChatRequest and returns the raw response.
func (c *ChatGPTClient) ChatAdvanced(request model.ChatRequest) (string, error) {
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ChatRequest: %w", err)
	}

	// The endpoint remains the same as in your working example.
	url := "https://api.openai.com/v1/responses"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	log.Printf("API Request:\ncurl %s \\\n  -H \"Content-Type: application/json\" \\\n  -H \"Authorization: Bearer %s\" \\\n  -d '%s'\n", url, c.APIKey, string(bodyBytes))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire response body.
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Pretty-print the raw JSON response.
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBytes, "", "  "); err != nil {
		log.Printf("Failed to pretty-print response: %v", err)
	} else {
		log.Printf("Chat response (pretty):\n%s", prettyJSON.String())
	}

	var respData struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBytes, &respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(respData.Output) > 0 {
		if len(respData.Output[0].Content) > 0 {
			return respData.Output[0].Content[0].Text, nil
		}
	}
	return "", fmt.Errorf("no output returned in response")

}

// ChatAdvancedParsed sends a ChatRequest and unmarshals the response into target.
func (c *ChatGPTClient) ChatAdvancedParsed(request model.ChatRequest, target interface{}) error {
	raw, err := c.ChatAdvanced(request)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), target)
}

// SetModel sets the model.
func (c *ChatGPTClient) SetModel(model string) {
	c.Model = model
}

// SetTemperature sets the temperature.
func (c *ChatGPTClient) SetTemperature(temp float64) {
	c.Temperature = temp
}

// GetTemperature returns the temperature.
func (c *ChatGPTClient) GetTemperature() float64 {
	return c.Temperature
}

// GetModel returns the model.
func (c *ChatGPTClient) GetModel() string {
	return c.Model
}
