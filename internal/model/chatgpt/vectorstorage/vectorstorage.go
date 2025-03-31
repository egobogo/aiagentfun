package vectorstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/egobogo/aiagents/internal/model"
)

type Client struct {
	APIKey string
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
	}
}

// CreateStorage creates a new vector store with the given name.
func (c *Client) CreateStorage(name string) (model.VectorStore, error) {
	payload := map[string]string{"name": name}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return model.VectorStore{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	url := "https://api.openai.com/v1/vector_stores"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return model.VectorStore{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return model.VectorStore{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return model.VectorStore{}, fmt.Errorf("failed to read response: %w", err)
	}
	var vs model.VectorStore
	if err := json.Unmarshal(respBytes, &vs); err != nil {
		return model.VectorStore{}, fmt.Errorf("failed to unmarshal vector store: %w", err)
	}
	return vs, nil
}

// DeleteStorage deletes a vector store identified by its ID.
func (c *Client) DeleteStorage(vectorStoreID string) error {
	url := fmt.Sprintf("https://api.openai.com/v1/vector_stores/%s", vectorStoreID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBytes, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete vector store, status: %d, response: %s", resp.StatusCode, string(respBytes))
	}
	return nil
}

// AttachFile attaches an already uploaded file (by file ID) to a vector store.
func (c *Client) AttachFile(vectorStoreID, fileID string) (model.File, error) {
	payload := map[string]string{"file_id": fileID}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	url := fmt.Sprintf("https://api.openai.com/v1/vector_stores/%s/files", vectorStoreID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return model.File{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("OpenAI-Beta", "assistants=v2")

	client := &http.Client{}
	resp, err := client.Do(req)
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

	// Poll until the file appears in the vector store's file list.
	timeout := time.Now().Add(60 * time.Second)
	for {
		files, err := c.ListFiles(vectorStoreID)
		if err != nil {
			return model.File{}, fmt.Errorf("failed to list files: %w", err)
		}
		found := false
		for _, f := range files {
			if f.ID == fileID {
				found = true
				break
			}
		}
		if found {
			break
		}
		if time.Now().After(timeout) {
			return model.File{}, fmt.Errorf("timeout waiting for file %s to be attached", fileID)
		}
		time.Sleep(2 * time.Second)
	}

	return fileObj, nil
}

// ListStorages returns all vector stores.
func (c *Client) ListStorages() ([]model.VectorStore, error) {
	url := "https://api.openai.com/v1/vector_stores"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var listResponse struct {
		Object  string              `json:"object"`
		Data    []model.VectorStore `json:"data"`
		FirstID string              `json:"first_id"`
		LastID  string              `json:"last_id"`
		HasMore bool                `json:"has_more"`
	}
	if err := json.Unmarshal(respBytes, &listResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
	}
	return listResponse.Data, nil
}

// ListFiles returns all files attached to the specified vector store.
func (c *Client) ListFiles(vectorStoreID string) ([]model.File, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/vector_stores/%s/files", vectorStoreID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	var listResponse struct {
		Object  string       `json:"object"`
		Data    []model.File `json:"data"`
		FirstID string       `json:"first_id"`
		LastID  string       `json:"last_id"`
		HasMore bool         `json:"has_more"`
	}
	if err := json.Unmarshal(respBytes, &listResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list files response: %w", err)
	}
	return listResponse.Data, nil
}

// DeleteFile deletes a file from a vector store.
func (c *Client) DeleteFile(vectorStoreID, fileID string) (model.File, error) {
	url := fmt.Sprintf("https://api.openai.com/v1/vector_stores/%s/files/%s", vectorStoreID, fileID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to create DELETE request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Beta", "assistants=v2")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to send DELETE request: %w", err)
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return model.File{}, fmt.Errorf("failed to read response: %w", err)
	}
	var fileObj model.File
	if err := json.Unmarshal(respBytes, &fileObj); err != nil {
		return model.File{}, fmt.Errorf("failed to unmarshal delete response: %w", err)
	}
	return fileObj, nil
}
