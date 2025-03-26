package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/egobogo/aiagents/internal/docs"
)

// NotionClient is a concrete implementation of docs.DocumentationClient using the Notion API in a wiki style.
type NotionClient struct {
	Token      string // Notion integration token (secret)
	ParentPage string // The parent page ID for the wiki (the root wiki page)
	BaseURL    string // e.g., "https://api.notion.com/v1"
	APIVersion string // e.g., "2022-06-28"
	HTTPClient *http.Client
}

// NewNotionClient creates a new NotionClient instance.
func NewNotionClient(token, parentPage string) *NotionClient {
	return &NotionClient{
		Token:      token,
		ParentPage: parentPage,
		BaseURL:    "https://api.notion.com/v1",
		APIVersion: "2022-06-28",
		HTTPClient: &http.Client{},
	}
}

// CreatePage creates a new wiki page as a child of the specified parent page.
// If parentPageID is an empty string, the page is created under the root.
func (nc *NotionClient) CreatePage(title string, content string, parentPageID string) (docs.Page, error) {
	if parentPageID == "" {
		parentPageID = nc.ParentPage
	}

	payload := map[string]interface{}{
		"parent": map[string]string{
			"page_id": parentPageID,
		},
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{"text": map[string]string{"content": title}},
				},
			},
		},
		"children": []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]string{"content": content}},
					},
				},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return docs.Page{}, fmt.Errorf("failed to marshal payload: %w", err)
	}
	req, err := http.NewRequest("POST", nc.BaseURL+"/pages", bytes.NewBuffer(data))
	if err != nil {
		return docs.Page{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	req.Header.Add("Content-Type", "application/json")

	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return docs.Page{}, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return docs.Page{}, fmt.Errorf("failed to create page, status: %d, body: %s", resp.StatusCode, string(body))
	}
	var result struct {
		ID         string `json:"id"`
		Properties struct {
			Title struct {
				Title []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"title"`
			} `json:"title"`
		} `json:"properties"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return docs.Page{}, fmt.Errorf("failed to decode response: %w", err)
	}
	page := docs.Page{
		ID:      result.ID,
		Title:   result.Properties.Title.Title[0].Text.Content,
		Content: content,
		URL:     result.URL,
	}
	return page, nil
}

// UpdatePage updates the content of a page.
// If replace is true, it erases all non-child_page content before appending the new content.
// Otherwise, it simply appends the new content.
func (nc *NotionClient) UpdatePage(pageID string, content string, replace bool) error {
	if replace {
		// Erase existing content (but keep child_page blocks).
		if err := nc.ClearPageContent(pageID); err != nil {
			return fmt.Errorf("failed to clear existing content: %w", err)
		}
	}
	appendPayload := map[string]interface{}{
		"children": []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{"type": "text", "text": map[string]string{"content": content}},
					},
				},
			},
		},
	}
	data, err := json.Marshal(appendPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal append payload: %w", err)
	}
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/blocks/%s/children", nc.BaseURL, pageID), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create append request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	req.Header.Add("Content-Type", "application/json")
	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to append new block: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to append new block, status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ClearPageContent erases all content blocks of a page except for child_page blocks.
// It retrieves all child blocks and archives those that are not of type "child_page".
func (nc *NotionClient) ClearPageContent(pageID string) error {
	url := fmt.Sprintf("%s/blocks/%s/children", nc.BaseURL, pageID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to list blocks: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to list blocks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to list blocks, status: %d, body: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Results []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"results"`
		HasMore    bool   `json:"has_more"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode blocks: %w", err)
	}
	// Archive each block that is not a child_page.
	for _, block := range result.Results {
		if block.Type != "child_page" {
			patchPayload := map[string]interface{}{
				"archived": true,
			}
			patchData, err := json.Marshal(patchPayload)
			if err != nil {
				return fmt.Errorf("failed to marshal patch payload: %w", err)
			}
			patchURL := fmt.Sprintf("%s/blocks/%s", nc.BaseURL, block.ID)
			patchReq, err := http.NewRequest("PATCH", patchURL, bytes.NewBuffer(patchData))
			if err != nil {
				return fmt.Errorf("failed to create patch request: %w", err)
			}
			patchReq.Header.Add("Authorization", "Bearer "+nc.Token)
			patchReq.Header.Add("Notion-Version", nc.APIVersion)
			patchReq.Header.Add("Content-Type", "application/json")
			patchResp, err := nc.HTTPClient.Do(patchReq)
			if err != nil {
				return fmt.Errorf("failed to patch block: %w", err)
			}
			patchResp.Body.Close()
			if patchResp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(patchResp.Body)
				return fmt.Errorf("failed to patch block, status: %d, body: %s", patchResp.StatusCode, string(body))
			}
		}
	}
	return nil
}

// ReadPage retrieves a wiki page by its ID and assembles its full content
// by collecting the text of its immediate children (and their children, except for child pages).
func (nc *NotionClient) ReadPage(pageID string) (docs.Page, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/pages/%s", nc.BaseURL, pageID), nil)
	if err != nil {
		return docs.Page{}, fmt.Errorf("failed to create read request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return docs.Page{}, fmt.Errorf("failed to read page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return docs.Page{}, fmt.Errorf("failed to read page, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     string `json:"id"`
		Parent struct {
			Type   string `json:"type"`
			PageID string `json:"page_id,omitempty"`
		} `json:"parent"`
		Properties struct {
			Title struct {
				Title []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"title"`
			} `json:"title"`
		} `json:"properties"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return docs.Page{}, fmt.Errorf("failed to decode page: %w", err)
	}

	// Collect content from child blocks, using a processed map to avoid duplicate blocks.
	var collected []string
	processed := make(map[string]bool)
	if err := nc.collectBlockContent(pageID, &collected, processed); err != nil {
		return docs.Page{}, fmt.Errorf("failed to collect page content: %w", err)
	}
	fullContent := strings.Join(collected, "\n")
	page := docs.Page{
		ID:       result.ID,
		Title:    result.Properties.Title.Title[0].Text.Content,
		URL:      result.URL,
		ParentID: result.Parent.PageID,
		Content:  fullContent,
	}
	return page, nil
}

// DeletePage archives (deletes) a page by setting its "archived" property to true.
func (nc *NotionClient) DeletePage(pageID string) error {
	payload := map[string]interface{}{
		"archived": true,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete payload: %w", err)
	}
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s/pages/%s", nc.BaseURL, pageID), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	req.Header.Add("Content-Type", "application/json")
	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete page, status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListSubPages returns the immediate child pages of a given parent page
// by filtering the results from the SearchPages method.
func (nc *NotionClient) ListSubPages(parentPageID string) ([]docs.Page, error) {
	allPages, err := nc.SearchPages("")
	if err != nil {
		return nil, fmt.Errorf("failed to search pages: %w", err)
	}
	var subPages []docs.Page
	for _, p := range allPages {
		if p.ParentID == parentPageID {
			subPages = append(subPages, p)
		}
	}
	return subPages, nil
}

// ListPages recursively lists every page in the wiki hierarchy starting from the root page.
// It retrieves all pages via the Search API, then builds the full hierarchy by recursively
// finding and appending each child page (using the ParentID field) to the result.
func (nc *NotionClient) ListPages() ([]docs.Page, error) {
	allPages, err := nc.SearchPages("")
	if err != nil {
		return nil, fmt.Errorf("failed to search pages: %w", err)
	}
	root, err := nc.ReadPage(nc.ParentPage)
	if err != nil {
		return nil, fmt.Errorf("failed to read root page: %w", err)
	}
	root.Path = root.Title
	var result []docs.Page
	result = append(result, root)
	var addChildren func(parent docs.Page)
	addChildren = func(parent docs.Page) {
		for _, p := range allPages {
			if p.ParentID == parent.ID {
				p.Path = parent.Path + "/" + p.Title
				result = append(result, p)
				addChildren(p)
			}
		}
	}
	addChildren(root)
	return result, nil
}

// SearchPages uses Notion's official search endpoint to find wiki pages matching the query.
// This implementation supports pagination to retrieve all pages.
func (nc *NotionClient) SearchPages(query string) ([]docs.Page, error) {
	var pages []docs.Page
	var startCursor interface{} = nil

	for {
		payload := map[string]interface{}{
			"query": query,
			"filter": map[string]interface{}{
				"value":    "page",
				"property": "object",
			},
		}
		if startCursor != nil {
			payload["start_cursor"] = startCursor
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search payload: %w", err)
		}
		req, err := http.NewRequest("POST", nc.BaseURL+"/search", bytes.NewBuffer(data))
		if err != nil {
			return nil, fmt.Errorf("failed to create search request: %w", err)
		}
		req.Header.Add("Authorization", "Bearer "+nc.Token)
		req.Header.Add("Notion-Version", nc.APIVersion)
		req.Header.Add("Content-Type", "application/json")
		resp, err := nc.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to perform search request: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("search request failed, status: %d, body: %s", resp.StatusCode, string(body))
		}

		var searchResult struct {
			Results []struct {
				ID     string `json:"id"`
				Parent struct {
					Type   string `json:"type"`
					PageID string `json:"page_id,omitempty"`
				} `json:"parent"`
				Properties struct {
					Title struct {
						Title []struct {
							Text struct {
								Content string `json:"content"`
							} `json:"text"`
						} `json:"title"`
					} `json:"title"`
				} `json:"properties"`
				URL string `json:"url"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
			return nil, fmt.Errorf("failed to decode search results: %w", err)
		}
		for _, res := range searchResult.Results {
			if len(res.Properties.Title.Title) > 0 {
				page := docs.Page{
					ID:       res.ID,
					Title:    res.Properties.Title.Title[0].Text.Content,
					URL:      res.URL,
					ParentID: res.Parent.PageID,
				}
				pages = append(pages, page)
			}
		}
		if !searchResult.HasMore {
			break
		}
		startCursor = searchResult.NextCursor
	}
	return pages, nil
}

// PrintTree returns a string representation of the page hierarchy in a tree-like format.
// It builds a mapping of parentID -> children and then recursively assembles the tree string.
func (nc *NotionClient) PrintTree() (string, error) {
	pages, err := nc.ListPages()
	if err != nil {
		return "", fmt.Errorf("failed to list pages: %w", err)
	}

	// Build a map of parentID to its children.
	childrenMap := make(map[string][]docs.Page)
	var root docs.Page
	for _, p := range pages {
		if p.ID == nc.ParentPage {
			root = p
		}
		childrenMap[p.ParentID] = append(childrenMap[p.ParentID], p)
	}

	var builder strings.Builder

	// Recursive function to build the tree string.
	var buildTree func(parentID string, prefix string)
	buildTree = func(parentID string, prefix string) {
		children := childrenMap[parentID]
		for i, child := range children {
			var connector string
			if i == len(children)-1 {
				connector = "└── "
			} else {
				connector = "├── "
			}
			builder.WriteString(fmt.Sprintf("%s%s%s (ID: %s, URL: %s)\n", prefix, connector, child.Title, child.ID, child.URL))
			newPrefix := prefix
			if i == len(children)-1 {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			buildTree(child.ID, newPrefix)
		}
	}

	// Build tree starting from the root.
	builder.WriteString(fmt.Sprintf("%s (ID: %s, URL: %s)\n", root.Title, root.ID, root.URL))
	buildTree(root.ID, "")
	return builder.String(), nil
}

// readBlockContent recursively fetches the content for a given block ID,
// including any nested child blocks.
func (nc *NotionClient) readBlockContent(blockID string) (string, error) {
	var contentBuilder strings.Builder
	url := fmt.Sprintf("%s/blocks/%s/children", nc.BaseURL, blockID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for block children: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+nc.Token)
	req.Header.Add("Notion-Version", nc.APIVersion)
	resp, err := nc.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get block children: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get block children, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var blocksResult struct {
		Results []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			HasChildren bool   `json:"has_children"`
			Paragraph   struct {
				RichText []struct {
					Text struct {
						Content string `json:"content"`
					} `json:"text"`
				} `json:"rich_text"`
			} `json:"paragraph"`
		} `json:"results"`
		HasMore    bool   `json:"has_more"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&blocksResult); err != nil {
		return "", fmt.Errorf("failed to decode block children: %w", err)
	}

	for _, block := range blocksResult.Results {
		if block.Type == "paragraph" {
			for _, rt := range block.Paragraph.RichText {
				contentBuilder.WriteString(rt.Text.Content)
				contentBuilder.WriteString("\n")
			}
		}
		// If the block has children, recursively fetch and append their content.
		if block.HasChildren {
			childContent, err := nc.readBlockContent(block.ID)
			if err != nil {
				return "", fmt.Errorf("failed to read child block content: %w", err)
			}
			contentBuilder.WriteString(childContent)
		}
	}
	return contentBuilder.String(), nil
}

// readBlockContentRecursively fetches the content for a given block ID,
// including all nested children, handling bullet list items,
// and avoids duplicate processing using the processed and addedContent maps.
// It also retries on transient errors (e.g., 502 Bad Gateway) up to maxRetries.
func (nc *NotionClient) readBlockContentRecursively(blockID string, processed map[string]bool, addedContent map[string]bool) (string, error) {
	var contentBuilder strings.Builder
	var startCursor *string = nil

	const maxRetries = 3
	baseDelay := time.Second

	for {
		// Build the URL with pagination if needed.
		url := fmt.Sprintf("%s/blocks/%s/children", nc.BaseURL, blockID)
		if startCursor != nil {
			url = fmt.Sprintf("%s?start_cursor=%s", url, *startCursor)
		}

		var body []byte
		var respStatus int
		var err error

		// Retry loop for transient errors.
		retryCount := 0
	retryRequest:
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request for block children: %w", err)
		}
		req.Header.Add("Authorization", "Bearer "+nc.Token)
		req.Header.Add("Notion-Version", nc.APIVersion)

		resp, err := nc.HTTPClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to get block children: %w", err)
		}
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		respStatus = resp.StatusCode

		if respStatus != http.StatusOK {
			// If 502 and we haven't retried maxRetries times, wait and retry.
			if respStatus == http.StatusBadGateway && retryCount < maxRetries {
				retryCount++
				time.Sleep(baseDelay * time.Duration(retryCount))
				goto retryRequest
			}
			return "", fmt.Errorf("failed to get block children, status: %d, body: %s", respStatus, string(body))
		}

		var blocksResult struct {
			Results []struct {
				ID          string `json:"id"`
				Type        string `json:"type"`
				HasChildren bool   `json:"has_children"`
				// For paragraph blocks.
				Paragraph struct {
					RichText []struct {
						Text struct {
							Content string `json:"content"`
						} `json:"text"`
					} `json:"rich_text"`
				} `json:"paragraph"`
				// For bullet list items.
				BulletedListItem struct {
					RichText []struct {
						Text struct {
							Content string `json:"content"`
						} `json:"text"`
					} `json:"rich_text"`
				} `json:"bulleted_list_item"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := json.Unmarshal(body, &blocksResult); err != nil {
			return "", fmt.Errorf("failed to decode block children: %w", err)
		}

		for _, block := range blocksResult.Results {
			// Skip if already processed.
			if processed[block.ID] {
				continue
			}
			processed[block.ID] = true

			// Function to add a line if not already added.
			addLine := func(line string) {
				line = strings.TrimSpace(line)
				if line != "" && !addedContent[line] {
					contentBuilder.WriteString(line)
					contentBuilder.WriteString("\n")
					addedContent[line] = true
				}
			}

			// Process content based on block type.
			switch block.Type {
			case "paragraph":
				for _, rt := range block.Paragraph.RichText {
					addLine(rt.Text.Content)
				}
			case "bulleted_list_item":
				for _, rt := range block.BulletedListItem.RichText {
					addLine("- " + rt.Text.Content)
				}
			}

			// Recursively fetch nested children if available.
			if block.HasChildren {
				childContent, err := nc.readBlockContentRecursively(block.ID, processed, addedContent)
				if err != nil {
					return "", fmt.Errorf("failed to read nested block content: %w", err)
				}
				for _, line := range strings.Split(childContent, "\n") {
					addLine(line)
				}
			}
		}

		if !blocksResult.HasMore {
			break
		}
		startCursor = &blocksResult.NextCursor
	}
	return contentBuilder.String(), nil
}

// collectBlockContent traverses the children of a given block ID (using pagination)
// and collects their text content in the order encountered.
// Blocks of type "child_page" are skipped to avoid duplication (their content will be read separately).
func (nc *NotionClient) collectBlockContent(blockID string, collected *[]string, processed map[string]bool) error {
	var startCursor *string = nil
	for {
		url := fmt.Sprintf("%s/blocks/%s/children", nc.BaseURL, blockID)
		if startCursor != nil {
			url = fmt.Sprintf("%s?start_cursor=%s", url, *startCursor)
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for block children: %w", err)
		}
		req.Header.Add("Authorization", "Bearer "+nc.Token)
		req.Header.Add("Notion-Version", nc.APIVersion)

		resp, err := nc.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to get block children: %w", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get block children, status: %d, body: %s", resp.StatusCode, string(body))
		}

		var blocksResult struct {
			Results []struct {
				ID          string `json:"id"`
				Type        string `json:"type"`
				HasChildren bool   `json:"has_children"`
				// For paragraph blocks.
				Paragraph struct {
					RichText []struct {
						Text struct {
							Content string `json:"content"`
						} `json:"text"`
					} `json:"rich_text"`
				} `json:"paragraph"`
				// For bullet list items.
				BulletedListItem struct {
					RichText []struct {
						Text struct {
							Content string `json:"content"`
						} `json:"text"`
					} `json:"rich_text"`
				} `json:"bulleted_list_item"`
				// For child pages.
				ChildPage struct {
					Title string `json:"title"`
				} `json:"child_page"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := json.Unmarshal(body, &blocksResult); err != nil {
			return fmt.Errorf("failed to decode block children: %w", err)
		}

		for _, block := range blocksResult.Results {
			if processed[block.ID] {
				continue
			}
			processed[block.ID] = true

			switch block.Type {
			case "paragraph":
				var parts []string
				for _, rt := range block.Paragraph.RichText {
					parts = append(parts, rt.Text.Content)
				}
				line := strings.Join(parts, " ")
				if line != "" {
					*collected = append(*collected, line)
				}
			case "bulleted_list_item":
				var parts []string
				for _, rt := range block.BulletedListItem.RichText {
					parts = append(parts, rt.Text.Content)
				}
				line := "- " + strings.Join(parts, " ")
				if line != "" {
					*collected = append(*collected, line)
				}
			case "child_page":
				// Skip traversing child pages to avoid duplication.
				// Optionally, you could append a placeholder like the child page title:
				// if block.ChildPage.Title != "" {
				//     *collected = append(*collected, fmt.Sprintf("[Child Page: %s]", block.ChildPage.Title))
				// }
			default:
				// For any other block type, you could decide how to handle it.
			}

			// Only traverse children if the block is not a child_page.
			if block.HasChildren && block.Type != "child_page" {
				if err := nc.collectBlockContent(block.ID, collected, processed); err != nil {
					return err
				}
			}
		}

		if !blocksResult.HasMore {
			break
		}
		startCursor = &blocksResult.NextCursor
	}
	return nil
}
