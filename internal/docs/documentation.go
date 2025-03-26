package docs

// DocumentationClient defines operations for managing documentation pages.
type DocumentationClient interface {
	// CreatePage creates a new page. If parentPageID is empty, the page is created under the root.
	CreatePage(title string, content string, parentPageID string) (Page, error)

	// UpdatePage updates a page's content. If replace is true, the existing content (excluding child pages) is replaced.
	UpdatePage(pageID string, content string, replace bool) error

	ReadPage(pageID string) (Page, error)
	SearchPages(query string) ([]Page, error)
	ListPages() ([]Page, error)
	// ListSubPages lists child pages (sub-pages) under the given parent page.
	ListSubPages(parentPageID string) ([]Page, error)
	DeletePage(pageID string) error
	PrintTree() (string, error)
}

// Page represents a documentation page.
type Page struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"` // Note: content may be empty in search results.
	URL      string `json:"url"`
	Path     string `json:"path"`
	ParentID string `json:"ParentID"`
}
