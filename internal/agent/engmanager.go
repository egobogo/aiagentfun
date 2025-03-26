package agent

import (
	"fmt"
)

type EngineeringManagerAgent struct {
	*BaseAgent
}

func NewEngineeringManagerAgent(base *BaseAgent) *EngineeringManagerAgent {
	engManagerAgent := &EngineeringManagerAgent{
		BaseAgent: base,
	}
	if err := engManagerAgent.createContext(); err != nil {
		fmt.Printf("Failed to create context: %v\n", err)
	}
	return engManagerAgent
}

// createContext builds the agent's context by studying both documentation and repository information.
func (em *EngineeringManagerAgent) createContext() error {
	// 1. Gather documentation info: retrieve the tree representation and list all pages.
	docTree, err := em.DocsClient.PrintTree()
	if err != nil {
		return fmt.Errorf("failed to get documentation tree: %w", err)
	}
	pages, err := em.DocsClient.ListPages()
	if err != nil {
		return fmt.Errorf("failed to list documentation pages: %w", err)
	}
	var pagesInfo string
	for _, p := range pages {
		content, _ := em.DocsClient.ReadPage(p.ID)
		pagesInfo += fmt.Sprintf("Title: %s\nContent: %s\n", p.Title, content)
	}
	combinedDocContent := docTree + "\n" + pagesInfo

	// 2. Summarize the combined documentation content to create memory entries.
	docMemories, err := em.Summarize(combinedDocContent, nil)
	if err != nil {
		return fmt.Errorf("failed to summarize documentation info: %w", err)
	}

	// 3. Gather repository (code) information.
	repoJSON, repoSchema, err := em.GitClient.GatherRepoInfo()
	if err != nil {
		return fmt.Errorf("failed to gather repository info: %w", err)
	}
	repoMemories, err := em.Summarize(repoJSON, repoSchema)
	if err != nil {
		return fmt.Errorf("failed to summarize repository info: %w", err)
	}

	// 4. Combine the new memories from both documentation and repository.
	newMemories := append(docMemories, repoMemories...)

	// 5. For each new memory, search for related old memories.
	collectedOldMemories := em.Context.FilterRelatedMemories(newMemories)

	// 6. Build the updated hot context by merging new and old memories.
	updatedContext, err := em.BuildContext(newMemories, collectedOldMemories)
	if err != nil {
		return fmt.Errorf("failed to build updated context: %w", err)
	}

	// 7. Set the updated hot context.
	if err := em.Context.SetContext(updatedContext); err != nil {
		return fmt.Errorf("failed to set hot context: %w", err)
	}

	// 8. Refresh memories: remove redundant old memories and add the new ones.
	if err := em.RefreshMemories(collectedOldMemories, newMemories); err != nil {
		return fmt.Errorf("failed to refresh memories: %w", err)
	}

	return nil
}
