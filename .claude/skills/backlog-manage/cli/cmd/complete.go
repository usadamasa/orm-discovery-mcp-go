package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunComplete(dir string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: backlog-cli complete <id>")
	}
	id := args[0]

	activeFile, doneFile, err := resolveFiles(dir, id)
	if err != nil {
		return err
	}

	return store.MoveToCompleted(activeFile, doneFile, id, func(raw json.RawMessage) (json.RawMessage, error) {
		return markCompleted(raw, id)
	})
}

func resolveFiles(dir, id string) (active, done string, err error) {
	switch {
	case strings.HasPrefix(id, "task-"):
		return dir + "/tasks.jsonl", dir + "/tasks.done.jsonl", nil
	case strings.HasPrefix(id, "idea-"):
		return dir + "/ideas.jsonl", dir + "/ideas.done.jsonl", nil
	case strings.HasPrefix(id, "issue-"):
		return dir + "/issues.jsonl", dir + "/issues.done.jsonl", nil
	default:
		return "", "", fmt.Errorf("unknown ID prefix: %s", id)
	}
}

func markCompleted(raw json.RawMessage, id string) (json.RawMessage, error) {
	now := marshalString(model.NowFunc().Format("2006-01-02T15:04:05Z07:00"))

	if strings.HasPrefix(id, "issue-") {
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		item["status"] = json.RawMessage(`"resolved"`)
		item["resolved_at"] = now
		return json.Marshal(item)
	}

	// task or idea
	var item map[string]json.RawMessage
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	item["status"] = json.RawMessage(`"done"`)
	item["done_at"] = now
	return json.Marshal(item)
}
