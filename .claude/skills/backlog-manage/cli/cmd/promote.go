package cmd

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunPromote(dir string, args []string) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	id := fs.String("id", "", "Idea ID to promote (required)")
	to := fs.String("to", "", "Target type: task or issue (required)")
	priority := fs.String("priority", "p2", "Priority for task (p1/p2/p3)")
	severity := fs.String("severity", "medium", "Severity for issue (high/medium/low)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" || *to == "" {
		return fmt.Errorf("--id and --to are required")
	}
	if *to != "task" && *to != "issue" {
		return fmt.Errorf("--to must be 'task' or 'issue'")
	}

	// Read the idea
	ideasPath := dir + "/ideas.jsonl"
	raw, err := store.FindByID(ideasPath, *id)
	if err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("idea %s not found", *id)
	}

	var idea model.Idea
	if err := json.Unmarshal(raw, &idea); err != nil {
		return fmt.Errorf("unmarshal idea: %w", err)
	}

	var newID string
	sourceRef := marshalString(*id)

	switch *to {
	case "task":
		newID = model.GenerateID("task")
		task := model.NewTask(newID, idea.Title, idea.Description, *priority, idea.Tags)
		task.Source = "idea"
		task.SourceRef = sourceRef
		if err := store.Append(dir+"/tasks.jsonl", task); err != nil {
			return err
		}
	case "issue":
		newID = model.GenerateID("issue")
		issue := model.NewIssue(newID, idea.Title, idea.Description, *severity, idea.Tags)
		issue.Source = "idea"
		issue.SourceRef = sourceRef
		if err := store.Append(dir+"/issues.jsonl", issue); err != nil {
			return err
		}
	}

	// Move idea to done with promoted status
	donePath := dir + "/ideas.done.jsonl"
	err = store.MoveToCompleted(ideasPath, donePath, *id, func(raw json.RawMessage) (json.RawMessage, error) {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		item["status"] = json.RawMessage(`"promoted"`)
		item["promoted_to"] = marshalString(newID)
		item["done_at"] = marshalString(model.NowFunc().Format("2006-01-02T15:04:05Z07:00"))
		return json.Marshal(item)
	})
	if err != nil {
		return err
	}

	fmt.Println(newID)
	return nil
}
