// File: cmd/manual_workflow.go
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
	// Load configuration from the absolute or relative path (adjust as needed)
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
	for {
		current, err := wm.CurrentStep()
		if err != nil {
			log.Fatalf("Error getting current step: %v", err)
		}
		fmt.Printf("\nCurrent Step: %s\nDescription: %s\n", current.Name, current.Description)

		// If the current action indicates completion, exit.
		if strings.ToLower(current.Action) == "close_ticket" {
			fmt.Println("Workflow complete. Ticket closed.")
			break
		}

		// Display next choices.
		choices, err := wm.NextChoices()
		if err != nil {
			log.Fatalf("Error getting next choices: %v", err)
		}
		fmt.Println("Next choices:")
		for i, choice := range choices {
			fmt.Printf("  %d) %s (Next Step: %s, Action: %s)\n", i+1, choice.Option, choice.NextStep, choice.Action)
		}

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
		if err := wm.NextStep(choices[choice-1].NextStep); err != nil {
			log.Fatalf("Error advancing to step: %v", err)
		}
	}
}
