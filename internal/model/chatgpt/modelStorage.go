// internal/model/modelStorage.go
package chatgpt

// ModelInfo holds details about a language model.
type ModelInfo struct {
	Name string
	// PricePerToken is the cost per 1M tokens.
	PricePerToken float64
	// Strengths describes what the model is particularly good at.
	Strengths string
	// DefaultTemperature is a recommended temperature for this model.
	DefaultTemperature float64
}

var Cheap = ModelInfo{
	Name:               "gpt-4o-mini",
	PricePerToken:      0.60,
	Strengths:          "Cost-effective for general tasks with moderate complexity.",
	DefaultTemperature: 0.8,
}

var ExpensiveCoding = ModelInfo{
	Name:               "o3-mini",
	PricePerToken:      4.40,
	Strengths:          "Excels in complex reasoning and coding, ideal for advanced technical tasks.",
	DefaultTemperature: 0.8,
}
