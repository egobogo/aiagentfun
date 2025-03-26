package chatgptpromptbuilder

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/egobogo/aiagents/internal/config"
	model "github.com/egobogo/aiagents/internal/model"
	"github.com/invopop/jsonschema"
)

// FormatSchemaForModel uses the invopop/jsonschema Reflector to generate a JSON schema
// from a given Go value. It disables automatic schema IDs by setting Anonymous to true.
func FormatSchemaForModel(schema interface{}) (interface{}, error) {
	r := &jsonschema.Reflector{
		Anonymous:      true, // disable automatic schema IDs
		DoNotReference: true, // do not generate $ref values
	}
	s := r.Reflect(schema)
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema into object: %w", err)
	}
	// Remove the "$schema" field if present.
	delete(obj, "$schema")
	return obj, nil
}

// getSchemaName returns the type name of the provided schema.
// If a pointer is passed, it returns the underlying type name.
func getSchemaName(schema interface{}) string {
	t := reflect.TypeOf(schema)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

// WrapSchemaForArray wraps a given element schema in an object with property "result" as an array.
func WrapSchemaForArray(elementSchema interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"result": map[string]interface{}{
				"type":  "array",
				"items": elementSchema,
			},
		},
		"required":             []string{"result"},
		"additionalProperties": false,
	}
}

// ChatGPTPromptBuilder implements the PromptBuilder interface for ChatGPT.
type ChatGPTPromptBuilder struct{}

// New returns a new instance of ChatGPTPromptBuilder.
func New() *ChatGPTPromptBuilder {
	return &ChatGPTPromptBuilder{}
}

// Build constructs a ChatRequest by assembling messages and output formatting.
// If desiredOutput is provided, it generates a JSON Schema using reflection.
// For slice types, it wraps the schema in an object with property "result".
func (b *ChatGPTPromptBuilder) Build(role, mode, state, userInput string, desiredOutput interface{}, temperature float64, modelName string) (model.ChatRequest, error) {
	// Retrieve the role instruction from configuration.
	roleInstruction, err := config.GetRoleInstruction(role)
	if err != nil {
		return model.ChatRequest{}, fmt.Errorf("failed to get role instruction for %s: %w", role, err)
	}

	projectGoal := "Project: Create AI agent agile project team."
	// Retrieve the mode-specific prompt from configuration or global mode.
	modePrompt, err := config.GetRoleMode(role, mode)
	if err != nil {
		return model.ChatRequest{}, fmt.Errorf("failed to get mode prompt for %s in mode %s: %w", role, mode, err)
	}

	// Create messages with properly structured content.
	systemMsg := model.Message{
		Role: "system",
		Content: []map[string]string{
			{
				"type": "input_text",
				"text": fmt.Sprintf("%s\n%s\nStructured memory:\n%s", projectGoal, roleInstruction, state),
			},
		},
	}

	developerMsg := model.Message{
		Role: "assistant",
		Content: []map[string]string{
			{
				"type": "output_text",
				"text": modePrompt,
			},
		},
	}

	userMsg := model.Message{
		Role: "user",
		Content: []map[string]string{
			{
				"type": "input_text",
				"text": userInput,
			},
		},
	}

	chatReq := model.ChatRequest{
		Model:       modelName,
		Input:       []model.Message{systemMsg, developerMsg, userMsg},
		Temperature: 1.2,
	}

	if desiredOutput != nil {
		typ := reflect.TypeOf(desiredOutput)
		var schemaObj interface{}
		var schemaName string

		if typ.Kind() == reflect.Slice {
			v := reflect.ValueOf(desiredOutput)
			var sample interface{}
			if v.Len() > 0 {
				sample = v.Index(0).Interface()
			} else {
				sample = reflect.New(typ.Elem()).Elem().Interface()
			}
			elementSchema, err := FormatSchemaForModel(sample)
			if err != nil {
				return model.ChatRequest{}, fmt.Errorf("failed to generate schema for slice element: %w", err)
			}
			schemaObj = WrapSchemaForArray(elementSchema)
			schemaName = "ResultWrapper"
		} else {
			obj, err := FormatSchemaForModel(desiredOutput)
			if err != nil {
				return model.ChatRequest{}, fmt.Errorf("failed to generate schema: %w", err)
			}
			schemaObj = obj
			schemaName = getSchemaName(desiredOutput)
			if schemaName == "" {
				schemaName = "output_schema"
			}
		}

		chatReq.Text = &model.TextFormat{
			Format: model.FormatOptions{
				Type:   "json_schema",
				Name:   schemaName,
				Schema: schemaObj,
				Strict: true,
			},
		}
	}
	return chatReq, nil
}
