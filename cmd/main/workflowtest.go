package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/egobogo/aiagents/internal/config"
	"github.com/egobogo/aiagents/internal/config/filesys"
	"github.com/egobogo/aiagents/internal/workflow"
)

func main() {
	// Load configuration from YAML file.
	prov, err := filesys.NewFilesysConfigProvider("cfg/main.cfg.yaml")
	if err != nil {
		log.Fatalf("Could not create config provider: %v", err)
	}
	config.SetProvider(prov)
	if err := config.Load("cfg/main.cfg.yaml"); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create a new workflow manager using the loaded configuration.
	wm := workflow.NewWorkflowManager(config.GetLoadedConfig())
	reader := bufio.NewReader(os.Stdin)

	wm.SetCurrentStep("pm_product_step")

	for {
		current, err := wm.CurrentStep()
		if err != nil {
			log.Fatalf("Error getting current step: %v", err)
		}
		fmt.Printf("\nCurrent step: %s\nDescription: %s\n", current.Name, current.Description)

		// Check if this is the final step.
		if strings.ToLower(current.Action) == "close_ticket" {
			fmt.Println("Workflow complete. Ticket closed.")
			break
		}

		// Get the unified next choices.
		choices, err := wm.NextChoices()
		if err != nil {
			log.Fatalf("Error getting next choices: %v", err)
		}

		// Display the next choices.
		fmt.Println("Next choices:")
		for i, c := range choices {
			fmt.Printf("  %d) %s (Next Step: %s, Action: %s)\n", i+1, c.Option, c.NextStep, c.Action)
		}

		// Prompt the user for a selection.
		fmt.Print("Enter your choice: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(choices) {
			fmt.Println("Invalid input, please try again.")
			continue
		}

		// Advance to the chosen step.
		selectedNext := choices[choice-1].NextStep
		if err := wm.NextStep(selectedNext); err != nil {
			fmt.Printf("Error advancing to step: %v\n", err)
		}
	}
}
