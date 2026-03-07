package cmd

import (
	"flag"
	"fmt"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunList(dir string, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	typeFilter := fs.String("type", "", "Filter by type (task|idea|issue)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	showTasks := *typeFilter == "" || *typeFilter == "task"
	showIdeas := *typeFilter == "" || *typeFilter == "idea"
	showIssues := *typeFilter == "" || *typeFilter == "issue"

	if showTasks {
		fmt.Println("=== Tasks ===")
		tasks, err := store.ReadAll[model.Task](dir + "/tasks.jsonl")
		if err != nil {
			return err
		}
		if len(tasks) == 0 {
			fmt.Println("  (none)")
		}
		for _, t := range tasks {
			fmt.Printf("  [%s] %s: %s (%s)\n", t.Priority, t.ID, t.Title, t.Status)
		}
	}

	if showIdeas {
		fmt.Println("=== Ideas ===")
		ideas, err := store.ReadAll[model.Idea](dir + "/ideas.jsonl")
		if err != nil {
			return err
		}
		if len(ideas) == 0 {
			fmt.Println("  (none)")
		}
		for _, i := range ideas {
			fmt.Printf("  %s: %s (%s)\n", i.ID, i.Title, i.Status)
		}
	}

	if showIssues {
		fmt.Println("=== Issues ===")
		issues, err := store.ReadAll[model.Issue](dir + "/issues.jsonl")
		if err != nil {
			return err
		}
		if len(issues) == 0 {
			fmt.Println("  (none)")
		}
		for _, i := range issues {
			fmt.Printf("  [%s] %s: %s (%s)\n", i.Severity, i.ID, i.Title, i.Status)
		}
	}

	return nil
}
