package test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egobogo/aiagents/internal/config"
	"github.com/egobogo/aiagents/internal/config/filesys"
	"github.com/egobogo/aiagents/internal/workflow"
)

func TestWorkflowManager(t *testing.T) {
	// Determine the directory of this test file.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	// Compute the project root by going one directory up from the "test" folder.
	projectRoot := filepath.Join(filepath.Dir(filename), "..")
	// Build the absolute path to the configuration file.
	configPath := filepath.Join(projectRoot, "cfg", "main.cfg.yaml")
	t.Logf("Using config file: %s", configPath)

	// Load configuration from the computed absolute path.
	prov, err := filesys.NewFilesysConfigProvider(configPath)
	if err != nil {
		t.Fatalf("Could not create config provider: %v", err)
	}
	config.SetProvider(prov)
	if err := config.Load(configPath); err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a new workflow manager using the loaded configuration.
	wm := workflow.NewWorkflowManager(config.GetLoadedConfig())

	// Set a known starting step.
	if err := wm.SetCurrentStep("pm_product_step"); err != nil {
		t.Fatalf("Failed to set current step: %v", err)
	}

	// Retrieve the current step.
	current, err := wm.CurrentStep()
	if err != nil {
		t.Fatalf("Error getting current step: %v", err)
	}
	t.Logf("Current step: %s, Description: %s", current.Name, current.Description)

	// Get the next choices.
	choices, err := wm.NextChoices()
	if err != nil {
		t.Fatalf("Error getting next choices: %v", err)
	}
	if len(choices) == 0 {
		t.Fatalf("Expected at least one next choice, got zero")
	}
	t.Log("Next choices:")
	for i, c := range choices {
		t.Logf("Choice %d: %s, Next Step: %s, Action: %s", i+1, c.Option, c.NextStep, c.Action)
	}

	// Simulate advancing to the first next step.
	selectedNext := choices[0].NextStep
	if err := wm.NextStep(selectedNext); err != nil {
		t.Fatalf("Error advancing to step: %v", err)
	}
	newCurrent, err := wm.CurrentStep()
	if err != nil {
		t.Fatalf("Error getting new current step: %v", err)
	}
	t.Logf("Advanced to new current step: %s", newCurrent.Name)
}
