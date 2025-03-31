// File: test/notion_test.go
package test

import (
	"os"
	"testing"
	"time"

	"github.com/egobogo/aiagents/internal/docs/notion"
	"github.com/joho/godotenv"
)

func TestNotionClient(t *testing.T) {
	// Load environment variables.
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf("No .env file found; using system environment variables")
	}

	token := os.Getenv("NOTION_TOKEN")
	rootPageID := os.Getenv("NOTION_PARENT_PAGE")
	if token == "" || rootPageID == "" {
		t.Fatalf("NOTION_TOKEN or NOTION_PARENT_PAGE not set, skipping test")
	}

	// Create a NotionClient instance.
	nc := notion.NewNotionClient(token, rootPageID)

	// 1. Create a new page in the root.
	t.Log("Creating a new page in the root...")
	newPage, err := nc.CreatePage("Test Page "+time.Now().Format("20060102150405"), "Initial content", "")
	if err != nil {
		t.Fatalf("CreatePage failed: %v", err)
	}
	t.Logf("Created page: ID: %s, Title: %s, URL: %s", newPage.ID, newPage.Title, newPage.URL)

	// 2. Create a subpage under the new page.
	t.Log("Creating a subpage under the new page...")
	subPage, err := nc.CreatePage("Sub Page "+time.Now().Format("150405"), "Subpage content", newPage.ID)
	if err != nil {
		t.Fatalf("CreatePage (subpage) failed: %v", err)
	}
	t.Logf("Created subpage: ID: %s, Title: %s, URL: %s", subPage.ID, subPage.Title, subPage.URL)

	// 3. Read the new page.
	t.Log("Reading the new page...")
	readPage, err := nc.ReadPage(newPage.ID)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	t.Logf("Read page: ID: %s, Title: %s, URL: %s", readPage.ID, readPage.Title, readPage.URL)

	// 4. Update the page by replacing its content.
	t.Log("Updating the page with replacement...")
	err = nc.UpdatePage(newPage.ID, "Replaced content", true)
	if err != nil {
		t.Fatalf("UpdatePage (replace) failed: %v", err)
	}
	t.Log("Replaced page content successfully.")

	// 5. Append additional content.
	t.Log("Appending additional content to the page...")
	err = nc.UpdatePage(newPage.ID, "Appended content", false)
	if err != nil {
		t.Fatalf("UpdatePage (append) failed: %v", err)
	}
	t.Log("Appended content successfully.")

	// 6. Search for pages containing "Test".
	t.Log("Searching for pages with query 'Test'...")
	searchResults, err := nc.SearchPages("Test")
	if err != nil {
		t.Fatalf("SearchPages failed: %v", err)
	}
	t.Log("Search results:")
	for _, p := range searchResults {
		t.Logf("ID: %s, Title: %s, URL: %s", p.ID, p.Title, p.URL)
	}

	// 7. List all pages recursively.
	t.Log("Listing all pages recursively:")
	allPages, err := nc.ListPages()
	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}
	for i, p := range allPages {
		t.Logf("%d: ID: %s, Title: %s, Path: %s, URL: %s", i+1, p.ID, p.Title, p.Path, p.URL)
	}

	// 8. List immediate subpages.
	t.Logf("Listing immediate subpages of page ID %s:", newPage.ID)
	subPages, err := nc.ListSubPages(newPage.ID)
	if err != nil {
		t.Fatalf("ListSubPages failed: %v", err)
	}
	for i, p := range subPages {
		t.Logf("%d: ID: %s, Title: %s, URL: %s", i+1, p.ID, p.Title, p.URL)
	}

	// 9. Print the page tree.
	t.Log("Printing the page tree:")
	tree, err := nc.PrintTree()
	if err != nil {
		t.Fatalf("PrintTree failed: %v", err)
	}
	t.Log(tree)

	// 10. Cleanup: Delete the created subpage and page.
	t.Logf("Deleting subpage with ID %s...", subPage.ID)
	err = nc.DeletePage(subPage.ID)
	if err != nil {
		t.Fatalf("DeletePage (subpage) failed: %v", err)
	}
	t.Logf("Deleting page with ID %s...", newPage.ID)
	err = nc.DeletePage(newPage.ID)
	if err != nil {
		t.Fatalf("DeletePage (page) failed: %v", err)
	}
	t.Log("Cleanup complete.")
}
