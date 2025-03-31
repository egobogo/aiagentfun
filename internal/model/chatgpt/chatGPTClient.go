package chatgpt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/model/chatgpt/vectorstorage"
)

// ChatGPTClient implements the ModelClient interface using the OpenAI Chat API.
type ChatGPTClient struct {
	APIKey        string
	Model         string
	Temperature   float64
	VectorStorage *vectorstorage.Client // optional vector storage client
}

// NewChatGPTClient creates a new ChatGPTClient.
func NewChatGPTClient(apiKey, model string, vsClient *vectorstorage.Client) *ChatGPTClient {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &ChatGPTClient{
		APIKey:        apiKey,
		Model:         model,
		Temperature:   0.7,
		VectorStorage: vsClient,
	}
}

// PollUploadedFile polls the file endpoint until the file is available.
func (c *ChatGPTClient) pollUploadedFile(fileID string) (model.File, error) {
	timeout := time.Now().Add(60 * time.Second)
	for {
		fileObj, err := c.GetFile(fileID)
		if err == nil && fileObj.ID != "" {
			// Assuming that if the file object is returned and has an ID,
			// it is processed and ready.
			return fileObj, nil
		}
		if time.Now().After(timeout) {
			return model.File{}, fmt.Errorf("timeout waiting for file %s to be available", fileID)
		}
		time.Sleep(2 * time.Second)
	}
}

// writeDebugLog appends a log entry with a timestamp to "chatgpt_debug.log".
func writeDebugLog(content string) {
	logFile := "chatgpt_debug.log"
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer f.Close()
	timestamp := time.Now().Format(time.RFC3339)
	entry := fmt.Sprintf("[%s] %s\n", timestamp, content)
	if _, err := f.WriteString(entry); err != nil {
		fmt.Printf("Error writing log entry: %v\n", err)
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

func (c *ChatGPTClient) ChatAdvanced(request model.ChatRequest) (string, error) {
	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ChatRequest: %w", err)
	}

	url := "https://api.openai.com/v1/responses"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	writeDebugLog(fmt.Sprintf("API Request:\ncurl %s \\\n  -H \"Content-Type: application/json\" \\\n  -H \"Authorization: Bearer %s\" \\\n  -d '%s'",
		url, c.APIKey, string(bodyBytes)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Pretty-print the raw JSON response for debugging.
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, respBytes, "", "  "); err != nil {
		log.Printf("Failed to pretty-print response: %v", err)
	} else {
		log.Printf("Chat response (pretty):\n%s", prettyJSON.String())
	}

	// Define a temporary structure that includes the "type" field for each output.
	var respData struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}

	if err := json.Unmarshal(respBytes, &respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Iterate over the output blocks and return the text from the first block of type "message".
	for _, out := range respData.Output {
		if out.Type == "message" && len(out.Content) > 0 {
			return out.Content[0].Text, nil
		}
	}

	return "", fmt.Errorf("no message output returned in response")
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

// UploadFile uploads a file using the files API endpoint.
func (c *ChatGPTClient) UploadFile(filePath string, purpose string) (model.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return model.File{}, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return model.File{}, fmt.Errorf("failed to copy file content: %w", err)
	}
	if err := writer.WriteField("purpose", purpose); err != nil {
		return model.File{}, fmt.Errorf("failed to write purpose field: %w", err)
	}
	writer.Close()

	url := "https://api.openai.com/v1/files"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to read response: %w", err)
	}
	var fileObj model.File
	if err := json.Unmarshal(respBytes, &fileObj); err != nil {
		return model.File{}, fmt.Errorf("failed to unmarshal file object: %w", err)
	}
	// Poll until the file is available.
	processedFile, err := c.pollUploadedFile(fileObj.ID)
	if err != nil {
		return model.File{}, err
	}
	return processedFile, nil
}

// GetFile retrieves metadata for a file given its ID.
func (c *ChatGPTClient) GetFile(fileID string) (model.File, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/files/%s", fileID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to read response: %w", err)
	}

	var fileObj model.File
	if err := json.Unmarshal(respBytes, &fileObj); err != nil {
		return model.File{}, fmt.Errorf("failed to unmarshal file metadata: %w", err)
	}
	return fileObj, nil
}

// DeleteAllFiles deletes all files uploaded via the files API. This is useful for cleanup during tests.
func (c *ChatGPTClient) DeleteAllFiles() error {
	url := "https://api.openai.com/v1/files"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create list files request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read list files response: %w", err)
	}
	var listResponse struct {
		Data []model.File `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &listResponse); err != nil {
		return fmt.Errorf("failed to unmarshal list response: %w", err)
	}

	for _, file := range listResponse.Data {
		delURL := fmt.Sprintf("https://api.openai.com/v1/files/%s", file.ID)
		delReq, err := http.NewRequest("DELETE", delURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create delete request for file %s: %w", file.ID, err)
		}
		delReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
		delResp, err := client.Do(delReq)
		if err != nil {
			return fmt.Errorf("failed to delete file %s: %w", file.ID, err)
		}
		delResp.Body.Close()
	}
	return nil
}
