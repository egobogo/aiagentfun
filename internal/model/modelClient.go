package model

// Message represents a single message in a conversation.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// FilePurpose defines the allowed purposes for uploaded files.
type FilePurpose string

type SearchContextSize string

const (
	SearchContextSizeLow    SearchContextSize = "low"
	SearchContextSizeMedium SearchContextSize = "medium"
	SearchContextSizeHigh   SearchContextSize = "high"
)

const (
	FilePurposeAssistants FilePurpose = "assistants" // Used in the Assistants API
	FilePurposeBatch      FilePurpose = "batch"      // Used in the Batch API
	FilePurposeFineTune   FilePurpose = "fine-tune"  // Used for fine-tuning
	FilePurposeVision     FilePurpose = "vision"     // Images used for vision fine-tuning
	FilePurposeUserData   FilePurpose = "user_data"  // Flexible file type for any purpose
	FilePurposeEvals      FilePurpose = "evals"      // Used for eval data sets
)

// File represents metadata for an uploaded file.
type File struct {
	ID        string      `json:"id"`
	Object    string      `json:"object"`
	Bytes     int         `json:"bytes"`
	CreatedAt int64       `json:"created_at"`
	ExpiresAt int64       `json:"expires_at"`
	Filename  string      `json:"filename"`
	Purpose   FilePurpose `json:"purpose"`
}

// VectorStore represents a vector storage object.
type VectorStore struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FileAttachment represents a tuple containing a file ID and its corresponding vector store ID.
type FileAttachment struct {
	FileID        string
	VectorStoreID string
}

// TextFormat contains detailed output format instructions.
type TextFormat struct {
	Format FormatOptions `json:"format"`
}

// FormatOptions defines the schema for the desired output.
type FormatOptions struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Schema      interface{} `json:"schema,omitempty"`
	Strict      bool        `json:"strict"`
}

// WebSearch represents the configuration for the web search preview tool.
type WebSearch struct {
	Type         string                 `json:"type"`                          // Should always be "web_search_preview"
	UserLocation map[string]interface{} `json:"user_location,omitempty"`       // e.g., {"type": "approximate", "country": "GB", "city": "London", "region": "London"}
	ContextSize  SearchContextSize      `json:"search_context_size,omitempty"` // e.g., "low", "medium", or "high"
}

// ChatRequest represents the payload sent to the OpenAI API.
// Note: the official Responses API uses "input" (not "messages") to pass the conversation.
type ChatRequest struct {
	Model       string        `json:"model"`
	Input       []Message     `json:"input"`
	Temperature float64       `json:"temperature,omitempty"`
	Text        *TextFormat   `json:"text,omitempty"`
	Tools       []interface{} `json:"tools,omitempty"`
}

// ModelClient is an abstract, model-agnostic interface for interacting with a language model.
type ModelClient interface {
	Chat(prompt string) (string, error)
	ChatAdvanced(request ChatRequest) (string, error)
	ChatAdvancedParsed(req ChatRequest, target interface{}) error
	SetModel(model string)
	SetTemperature(temp float64)
	GetModel() string
	GetTemperature() float64
	UploadFile(filePath string, purpose string) (File, error)
	GetFile(fileID string) (File, error)
	DeleteAllFiles() error
}
