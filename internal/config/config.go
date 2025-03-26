package config

import "fmt"

// Config represents the entire YAML configuration.
type Config struct {
	Roles map[string]struct {
		Name          string `yaml:"name" json:"name"`
		Prompt        string `yaml:"prompt" json:"prompt"`
		DefaultAction string `yaml:"defaultAction" json:"defaultAction"`
		Actions       []struct {
			ID     string `yaml:"id" json:"id"`
			Name   string `yaml:"name" json:"name"`
			Mode   string `yaml:"mode" json:"mode"`
			Prompt string `yaml:"prompt,omitempty" json:"prompt,omitempty"`
		} `yaml:"actions" json:"actions"`
	} `yaml:"roles" json:"roles"`

	GlobalModes map[string]string `yaml:"globalModes" json:"globalModes"`

	Workflow struct {
		HighLevelTask string `yaml:"highLevelTask" json:"highLevelTask"`
		Steps         []Step `yaml:"steps" json:"steps"`
	} `yaml:"workflow" json:"workflow"`

	WorkflowControl struct {
		CurrentStep string   `yaml:"currentStep" json:"currentStep"`
		StepsOrder  []string `yaml:"stepsOrder" json:"stepsOrder"`
	} `yaml:"workflowControl" json:"workflowControl"`
}

// Step represents an individual step in the workflow.
type Step struct {
	ID          string      `yaml:"id" json:"id"`
	Name        string      `yaml:"name" json:"name"`
	Actor       string      `yaml:"actor" json:"actor"`
	Action      string      `yaml:"action" json:"action"`
	Description string      `yaml:"description" json:"description"`
	Next        interface{} `yaml:"next,omitempty" json:"next,omitempty"`
	Options     interface{} `yaml:"options,omitempty" json:"options,omitempty"` // New field for decision branches
}

// ConfigProvider is an interface for loading a configuration.
type ConfigProvider interface {
	LoadConfig(path string) (*Config, error)
}

// Global references
var (
	provider     ConfigProvider
	loadedConfig *Config
	ErrNotLoaded = fmt.Errorf("configuration not loaded")
)

// SetProvider sets the configuration provider.
func SetProvider(p ConfigProvider) {
	provider = p
}

// Load uses the current provider to load configuration from the given path.
func Load(path string) error {
	if provider == nil {
		return fmt.Errorf("no config provider set")
	}
	cfg, err := provider.LoadConfig(path)
	if err != nil {
		return err
	}
	loadedConfig = cfg
	return nil
}

func GetLoadedConfig() *Config {
	return loadedConfig
}

func GetRoleInstruction(role string) (string, error) {
	if loadedConfig == nil {
		return "", ErrNotLoaded
	}
	r, ok := loadedConfig.Roles[role]
	if !ok {
		return "", fmt.Errorf("role %q not found", role)
	}
	return r.Prompt, nil
}

// GetRoleMode returns the prompt for a given role and mode.
// It checks the role-specific modes first, then falls back to globalModes.
func GetRoleMode(role, mode string) (string, error) {
	if loadedConfig == nil {
		return "", ErrNotLoaded
	}
	if roleData, found := loadedConfig.Roles[role]; found {
		for _, act := range roleData.Actions {
			if act.Mode == mode {
				if act.Prompt != "" {
					return act.Prompt, nil
				}
				break
			}
		}
	}
	if prompt, ok := loadedConfig.GlobalModes[mode]; ok {
		return prompt, nil
	}
	return "", fmt.Errorf("mode %q not found for role %q and no global mode available", mode, role)
}
