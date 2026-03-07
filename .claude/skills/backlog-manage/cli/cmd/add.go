package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunAdd(dir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: backlog-cli add <task|idea|issue> --title ... --description ...")
	}

	itemType := args[0]
	subArgs := args[1:]

	switch itemType {
	case "task":
		return runAddTask(dir, subArgs)
	case "idea":
		return runAddIdea(dir, subArgs)
	case "issue":
		return runAddIssue(dir, subArgs)
	default:
		return fmt.Errorf("unknown type: %s (expected task, idea, or issue)", itemType)
	}
}

func runAddTask(dir string, args []string) error {
	fs := flag.NewFlagSet("add task", flag.ContinueOnError)
	title := fs.String("title", "", "Task title (required)")
	description := fs.String("description", "", "Task description (required)")
	priority := fs.String("priority", "p2", "Priority (p1/p2/p3)")
	tagsStr := fs.String("tags", "", "Comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" || *description == "" {
		return fmt.Errorf("--title and --description are required")
	}

	tags := parseTags(*tagsStr)
	id := model.GenerateID("task")
	task := model.NewTask(id, *title, *description, *priority, tags)

	if err := store.Append(dir+"/tasks.jsonl", task); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func runAddIdea(dir string, args []string) error {
	fs := flag.NewFlagSet("add idea", flag.ContinueOnError)
	title := fs.String("title", "", "Idea title (required)")
	description := fs.String("description", "", "Idea description (required)")
	tagsStr := fs.String("tags", "", "Comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" || *description == "" {
		return fmt.Errorf("--title and --description are required")
	}

	tags := parseTags(*tagsStr)
	id := model.GenerateID("idea")
	idea := model.NewIdea(id, *title, *description, tags)

	if err := store.Append(dir+"/ideas.jsonl", idea); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func runAddIssue(dir string, args []string) error {
	fs := flag.NewFlagSet("add issue", flag.ContinueOnError)
	title := fs.String("title", "", "Issue title (required)")
	description := fs.String("description", "", "Issue description (required)")
	severity := fs.String("severity", "medium", "Severity (high/medium/low)")
	tagsStr := fs.String("tags", "", "Comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *title == "" || *description == "" {
		return fmt.Errorf("--title and --description are required")
	}

	tags := parseTags(*tagsStr)
	id := model.GenerateID("issue")
	issue := model.NewIssue(id, *title, *description, *severity, tags)

	if err := store.Append(dir+"/issues.jsonl", issue); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func parseTags(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// marshalString returns a json.RawMessage for a string value.
func marshalString(s string) json.RawMessage {
	data, _ := json.Marshal(s)
	return data
}
