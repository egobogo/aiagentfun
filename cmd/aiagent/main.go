// cmd/ai_agent/main.go
package main

import (
	"log"
	"os"
	"time"

	"github.com/egobogo/aiagents/internal/agent"
	"github.com/egobogo/aiagents/internal/gitrepo"
	chatgpt "github.com/egobogo/aiagents/internal/model"
	"github.com/egobogo/aiagents/internal/roles"
	"github.com/egobogo/aiagents/internal/trello"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file.
	log.Println("Fetching env")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment variables")
	}

	// Initialize core clients.
	log.Println("creating clients")
	trelloClient := trello.NewTrelloClient(os.Getenv("TRELLO_API_KEY"), os.Getenv("TRELLO_API_TOKEN"), os.Getenv("TRELLO_BOARD_ID"))
	gitClient, err := gitrepo.NewGitClient(os.Getenv("GIT_REPO_URL"), os.Getenv("GIT_REPO_PATH"))
	if err != nil {
		log.Fatalf("Error creating Git client: %v", err)
	}
	// Initialize ChatGPT client using the chosen model.
	gptClient := chatgpt.NewChatGPTClient("o3-mini-high")

	// Create the base agent.
	log.Println("creating agents")
	baseAgent := &agent.AIAgent{
		Name:         "engManagerAgent", // For the engineering manager
		TrelloClient: trelloClient,
		GitClient:    gitClient,
		GPTClient:    gptClient,
	}

	// Create specialized agents using predefined role configurations.
	engManagerAgent := agent.NewEngineeringManagerAIAgent(baseAgent, roles.Manager.SystemMessage)
	backendAgent := agent.NewBackendDeveloperAIAgent(baseAgent, roles.Backend.SystemMessage)

	// Main event loop: poll for new tickets and process them.
	log.Println("starting main loop")
	for {
		// 1. Engineering Manager polls Trello for tickets assigned to him.
		tickets, err := engManagerAgent.GetAssignedTickets() // (Method to fetch tickets assigned to "engManagerAgent")
		if err != nil {
			log.Printf("Error fetching assigned tickets: %v", err)
		}

		if len(tickets) > 0 {
			for _, ticket := range tickets {
				// Engineering Manager reads the ticket and writes comments if needed,
				// waiting for manual approval or clarification via Trello comments.
				if err := engManagerAgent.HandleTicket(ticket); err != nil {
					log.Printf("Error handling ticket %s: %v", ticket.ID, err)
					continue
				}

				// Once clarifications are acquired, create a technical ticket.
				techTicket, err := engManagerAgent.CreateTechnicalTicket(ticket)
				if err != nil {
					log.Printf("Error creating technical ticket from %s: %v", ticket.ID, err)
					continue
				}

				// Assign the technical ticket to the Backend Developer agent.
				if err := engManagerAgent.AssignTicketToAgent(techTicket, backendAgent.Name); err != nil {
					log.Printf("Error assigning technical ticket %s to developer: %v", techTicket.ID, err)
				}
			}
		} else {
			log.Println("No tickets assigned to eng manager found")
		}

		// 2. Backend Developer polls for technical tickets assigned to it.
		techTickets, err := backendAgent.GetAssignedTickets() // (Method to fetch tickets assigned to backend agent)
		if err != nil {
			log.Printf("Error fetching technical tickets: %v", err)
		}

		if len(techTickets) > 0 {
			gitUsername := os.Getenv("GIT_USERNAME")
			gitToken := os.Getenv("GIT_TOKEN")

			for _, tkt := range techTickets {
				// Backend Developer asks for at least one clarification from the Engineering Manager.
				if err := backendAgent.RequestClarification(tkt); err != nil {
					log.Printf("Error requesting clarification for ticket %s: %v", tkt.ID, err)
					continue
				}
				// Wait (polling or blocking) until the Engineering Manager responds via Trello comments.
				if err := backendAgent.WaitForClarificationResponse(tkt); err != nil {
					log.Printf("Error waiting for clarification on ticket %s: %v", tkt.ID, err)
					continue
				}
				// Execute the technical assignment: generate code, write tests.
				if err := backendAgent.ExecuteTechnicalAssignment(tkt); err != nil {
					log.Printf("Error executing technical assignment for ticket %s: %v", tkt.ID, err)
					continue
				}
				// Commit the result to Git.
				err := backendAgent.CommitAndPushTicketResult(tkt,
					"Update ticket implementation", // commit message
					"BackendBot",                   // author name
					"backendbot@example.com",       // author email
					gitUsername,                    // git username (from env or config)
					gitToken)
				if err != nil {
					log.Printf("Error committing result for ticket %s: %v", tkt.ID, err)
					continue
				}
				// Finally, mark the ticket as done and reassign it (for example, to "me").
				if err := backendAgent.CloseTicket(tkt, "engManagerAgent"); err != nil {
					log.Printf("Error closing ticket %s: %v", tkt.ID, err)
				}
			}
		}

		// Sleep before polling again.
		time.Sleep(30 * time.Second)
	}
}
