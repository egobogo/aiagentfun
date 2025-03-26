package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/egobogo/aiagents/internal/docs/notion"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file.
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Optionally override with command-line flags.
	tokenFlag := flag.String("token", "", "Notion integration token")
	rootPageFlag := flag.String("root", "", "Root page ID")
	flag.Parse()

	// Load from environment if flags not provided.
	token := *tokenFlag
	if token == "" {
		token = os.Getenv("NOTION_TOKEN")
	}
	rootPageID := *rootPageFlag
	if rootPageID == "" {
		rootPageID = os.Getenv("NOTION_PARENT_PAGE")
	}

	if token == "" || rootPageID == "" {
		log.Fatal("Please provide both a token and a root page ID using environment variables or flags.")
	}

	// Create a NotionClient instance.
	nc := notion.NewNotionClient(token, rootPageID)

	// 1. Create a new page in the root.
	fmt.Println("Creating a new page in the root...")
	newPage, err := nc.CreatePage("Test Page "+time.Now().Format("20060102150405"), "Initial content", "")
	if err != nil {
		log.Fatalf("CreatePage failed: %v", err)
	}
	fmt.Printf("Created page: ID: %s, Title: %s, URL: %s\n", newPage.ID, newPage.Title, newPage.URL)

	// 2. Create a subpage under the new page.
	fmt.Println("Creating a subpage under the new page...")
	subPage, err := nc.CreatePage("Sub Page "+time.Now().Format("150405"), "Subpage content", newPage.ID)
	if err != nil {
		log.Fatalf("CreatePage (subpage) failed: %v", err)
	}
	fmt.Printf("Created subpage: ID: %s, Title: %s, URL: %s\n", subPage.ID, subPage.Title, subPage.URL)

	// 3. Read the new page.
	fmt.Println("Reading the new page...")
	readPage, err := nc.ReadPage(newPage.ID)
	if err != nil {
		log.Fatalf("ReadPage failed: %v", err)
	}
	fmt.Printf("Read page: ID: %s, Title: %s, URL: %s\n", readPage.ID, readPage.Title, readPage.URL)

	// 4. Update the page by replacing its content.
	fmt.Println("Updating the page with replacement...")
	err = nc.UpdatePage(newPage.ID, "Replaced content", true)
	if err != nil {
		log.Fatalf("UpdatePage (replace) failed: %v", err)
	}
	fmt.Println("Replaced page content successfully.")

	// 5. Append additional content (without replacement).
	fmt.Println("Appending additional content to the page...")
	err = nc.UpdatePage(newPage.ID, "Appended content", false)
	if err != nil {
		log.Fatalf("UpdatePage (append) failed: %v", err)
	}
	fmt.Println("Appended content successfully.")

	// 6. Search for pages containing "Test".
	fmt.Println("Searching for pages with query 'Test'...")
	searchResults, err := nc.SearchPages("Test")
	if err != nil {
		log.Fatalf("SearchPages failed: %v", err)
	}
	fmt.Println("Search results:")
	for _, p := range searchResults {
		fmt.Printf("ID: %s, Title: %s, URL: %s\n", p.ID, p.Title, p.URL)
	}

	// 7. List all pages recursively.
	fmt.Println("Listing all pages recursively:")
	allPages, err := nc.ListPages()
	if err != nil {
		log.Fatalf("ListPages failed: %v", err)
	}
	for i, p := range allPages {
		fmt.Printf("%d: ID: %s, Title: %s, Path: %s, URL: %s\n", i+1, p.ID, p.Title, p.Path, p.URL)
	}

	// 8. List immediate subpages of the new page.
	fmt.Printf("Listing immediate subpages of page ID %s:\n", newPage.ID)
	subPages, err := nc.ListSubPages(newPage.ID)
	if err != nil {
		log.Fatalf("ListSubPages failed: %v", err)
	}
	for i, p := range subPages {
		fmt.Printf("%d: ID: %s, Title: %s, URL: %s\n", i+1, p.ID, p.Title, p.URL)
	}

	// 9. Get the hierarchical tree as a string.
	fmt.Println("Printing the page tree:")
	tree, err := nc.PrintTree()
	if err != nil {
		log.Fatalf("PrintTree failed: %v", err)
	}
	fmt.Println(tree)

	// 10. Cleanup: Delete the created subpage and page.
	fmt.Printf("Deleting subpage with ID %s...\n", subPage.ID)
	err = nc.DeletePage(subPage.ID)
	if err != nil {
		log.Fatalf("DeletePage (subpage) failed: %v", err)
	}
	fmt.Printf("Deleting page with ID %s...\n", newPage.ID)
	err = nc.DeletePage(newPage.ID)
	if err != nil {
		log.Fatalf("DeletePage (page) failed: %v", err)
	}
	fmt.Println("Cleanup complete.")
}
