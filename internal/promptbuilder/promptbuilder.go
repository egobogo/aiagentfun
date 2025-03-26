package promptbuilder

import modelClient "github.com/egobogo/aiagents/internal/model"

// PromptBuilder defines an interface for constructing a complete ChatRequest.
type PromptBuilder interface {
	Build(role, mode, state, userInput string, desiredOutput interface{}, temperature float64, modelName string) (modelClient.ChatRequest, error)
}
