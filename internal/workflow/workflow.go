package workflow

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/egobogo/aiagents/internal/config"
)

// DecisionOption represents a normalized next choice.
type DecisionOption struct {
	Option   string // The display label (from the YAML decision option)
	NextStep string // The ID of the target step
	Name     string // The target step's name
	Action   string // The target step's action
}

// WorkflowManager controls the workflow state.
type WorkflowManager struct {
	Config      *config.Config
	currentStep string   // current step ID
	StepsOrder  []string // ordered list of step IDs
}

// NewWorkflowManager creates a new WorkflowManager using the loaded configuration.
func NewWorkflowManager(cfg *config.Config) *WorkflowManager {
	return &WorkflowManager{
		Config:      cfg,
		currentStep: cfg.WorkflowControl.CurrentStep,
		StepsOrder:  cfg.WorkflowControl.StepsOrder,
	}
}

// CurrentStep returns the current workflow step.
func (wm *WorkflowManager) CurrentStep() (config.Step, error) {
	for _, step := range wm.Config.Workflow.Steps {
		if step.ID == wm.currentStep {
			return step, nil
		}
	}
	return config.Step{}, fmt.Errorf("current step %q not found", wm.currentStep)
}

// NextChoices returns a unified slice of DecisionOption for the current step.
// It handles both decision branches (via Options or Next) and simple next steps.
func (wm *WorkflowManager) NextChoices() ([]DecisionOption, error) {
	current, err := wm.CurrentStep()
	if err != nil {
		return nil, err
	}
	var choices []DecisionOption

	// First, if the step has structured decision options (Options field), use those.
	if current.Options != nil {
		opts, err := getDecisionOptions(current)
		if err != nil {
			return nil, err
		}
		for _, opt := range opts {
			// Find the target step.
			for _, step := range wm.Config.Workflow.Steps {
				if step.ID == opt.NextStep {
					choices = append(choices, DecisionOption{
						Option:   opt.Option,
						NextStep: opt.NextStep,
						Name:     step.Name,
						Action:   step.Action,
					})
					break
				}
			}
		}
	} else if current.Next != nil {
		// Otherwise, if Next is set (it may be a string, a map, or a slice), normalize it.
		switch v := current.Next.(type) {
		case string:
			// Single next step.
			for _, step := range wm.Config.Workflow.Steps {
				if step.ID == v {
					choices = append(choices, DecisionOption{
						Option:   "Continue", // default label
						NextStep: v,
						Name:     step.Name,
						Action:   step.Action,
					})
					break
				}
			}
		case map[interface{}]interface{}:
			decision, ok := v["decision"]
			if !ok {
				return nil, errors.New("invalid branch structure: missing 'decision' key in next field")
			}
			opts, ok := decision.([]interface{})
			if !ok {
				return nil, errors.New("invalid branch structure: 'decision' is not a list in next field")
			}
			for _, rawOpt := range opts {
				nextID, optText, err := extractNextDetails(rawOpt, current.ID)
				if err != nil {
					return nil, err
				}
				for _, step := range wm.Config.Workflow.Steps {
					if step.ID == nextID {
						choices = append(choices, DecisionOption{
							Option:   optText,
							NextStep: nextID,
							Name:     step.Name,
							Action:   step.Action,
						})
						break
					}
				}
			}
		case []interface{}:
			// If Next is a slice of branch definitions.
			for _, branchItem := range v {
				var decision interface{}
				switch b := branchItem.(type) {
				case map[string]interface{}:
					decision = b["decision"]
				case map[interface{}]interface{}:
					decision = b["decision"]
				default:
					return nil, fmt.Errorf("unsupported type in next slice in step %q", current.ID)
				}
				if decision == nil {
					continue
				}
				opts, ok := decision.([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid branch structure: 'decision' is not a list in step %q", current.ID)
				}
				for _, rawOpt := range opts {
					nextID, optText, err := extractNextDetails(rawOpt, current.ID)
					if err != nil {
						return nil, err
					}
					for _, step := range wm.Config.Workflow.Steps {
						if step.ID == nextID {
							choices = append(choices, DecisionOption{
								Option:   optText,
								NextStep: nextID,
								Name:     step.Name,
								Action:   step.Action,
							})
							break
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("unsupported type for next field in step %q", current.ID)
		}
	} else {
		return nil, fmt.Errorf("no next or options field found in step %q", current.ID)
	}

	if len(choices) == 0 {
		return nil, fmt.Errorf("no next choices found for step %q", current.ID)
	}
	return choices, nil
}

// NextStep advances the workflow to the specified next step if it is valid.
func (wm *WorkflowManager) NextStep(nextID string) error {
	choices, err := wm.NextChoices()
	if err != nil {
		return err
	}
	valid := false
	for _, c := range choices {
		if c.NextStep == nextID {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("step %q is not a valid next choice from current step %q", nextID, wm.currentStep)
	}
	wm.currentStep = nextID
	wm.Config.WorkflowControl.CurrentStep = nextID
	return nil
}

// getDecisionOptions normalizes the Options field of a step into a slice of DecisionOption.
// This function assumes that the step has an Options field set.
func getDecisionOptions(s config.Step) ([]DecisionOption, error) {
	// We'll perform a similar conversion as before.
	optsInterface, ok := s.Options.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected type for Options in step %q: %T", s.ID, s.Options)
	}
	var opts []DecisionOption
	for _, raw := range optsInterface {
		var opt DecisionOption
		// Try map[string]interface{}.
		if m, ok := raw.(map[string]interface{}); ok {
			v, ok := m["option"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'option' key in step %q", s.ID)
			}
			opt.Option = v
			if id, ok := m["nextStep"].(string); ok {
				opt.NextStep = id
			} else {
				return nil, fmt.Errorf("missing or invalid 'nextStep' key in step %q", s.ID)
			}
			opts = append(opts, opt)
			continue
		}
		// Try map[interface{}]interface{}.
		if m, ok := raw.(map[interface{}]interface{}); ok {
			v, ok := m["option"].(string)
			if !ok {
				return nil, fmt.Errorf("missing or invalid 'option' key in step %q", s.ID)
			}
			opt.Option = v
			if id, ok := m["nextStep"].(string); ok {
				opt.NextStep = id
			} else {
				return nil, fmt.Errorf("missing or invalid 'nextStep' key in step %q", s.ID)
			}
			opts = append(opts, opt)
			continue
		}
		return nil, fmt.Errorf("cannot convert option value %v (type %T) in step %q", raw, raw, s.ID)
	}
	return opts, nil
}

// extractNextDetails extracts the nextStep ID and option text from a raw option.
// Returns (nextStep, optionText, error)
func extractNextDetails(opt interface{}, stepID string) (string, string, error) {
	// Try map[string]interface{}
	if m, ok := opt.(map[string]interface{}); ok {
		id, ok := m["nextStep"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid option format in step %q: missing or invalid 'nextStep'", stepID)
		}
		text, ok := m["option"].(string)
		if !ok {
			text = "Continue"
		}
		return id, text, nil
	}
	// Try map[interface{}]interface{}
	if m, ok := opt.(map[interface{}]interface{}); ok {
		id, ok := m["nextStep"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid option format in step %q: missing or invalid 'nextStep'", stepID)
		}
		text, ok := m["option"].(string)
		if !ok {
			text = "Continue"
		}
		return id, text, nil
	}
	// Try *yaml.Node.
	if node, ok := opt.(*yaml.Node); ok {
		var m map[string]interface{}
		if err := node.Decode(&m); err != nil {
			return "", "", fmt.Errorf("failed to decode yaml node in step %q: %w", stepID, err)
		}
		id, ok := m["nextStep"].(string)
		if !ok {
			return "", "", fmt.Errorf("invalid option format in step %q: missing or invalid 'nextStep'", stepID)
		}
		text, ok := m["option"].(string)
		if !ok {
			text = "Continue"
		}
		return id, text, nil
	}
	return "", "", fmt.Errorf("invalid option format in step %q: expected map type, got %T", stepID, opt)
}

// SetCurrentStep sets the current step to the given step ID if it exists.
func (wm *WorkflowManager) SetCurrentStep(stepID string) error {
	for _, step := range wm.Config.Workflow.Steps {
		if step.ID == stepID {
			wm.currentStep = stepID
			wm.Config.WorkflowControl.CurrentStep = stepID
			return nil
		}
	}
	return fmt.Errorf("step %q not found in workflow", stepID)
}
