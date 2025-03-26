package modelClient

// Message represents a single message in a conversation.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
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

// ChatRequest represents the payload sent to the OpenAI API.
// Note: the official Responses API uses "input" (not "messages") to pass the conversation.
type ChatRequest struct {
	Model       string      `json:"model"`
	Input       []Message   `json:"input"`
	Temperature float64     `json:"temperature,omitempty"`
	Text        *TextFormat `json:"text,omitempty"`
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
}
