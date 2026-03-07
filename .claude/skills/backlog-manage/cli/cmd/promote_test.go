package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func TestRunPromoteToTask(t *testing.T) {
	dir := setupTestDir(t)

	idea := model.NewIdea("idea-20260307-aaaa", "Good idea", "promote me", []string{"tag1"})
	store.Append(filepath.Join(dir, "ideas.jsonl"), idea)

	output := captureStdout(t, func() {
		err := RunPromote(dir, []string{"--id", "idea-20260307-aaaa", "--to", "task", "--priority", "p1"})
		if err != nil {
			t.Fatalf("RunPromote: %v", err)
		}
	})

	newID := strings.TrimSpace(output)
	if !strings.HasPrefix(newID, "task-20260307-") {
		t.Errorf("expected task ID, got: %s", newID)
	}

	// Idea should be gone from active
	activeIdeas, _ := store.ReadAll[model.Idea](filepath.Join(dir, "ideas.jsonl"))
	if len(activeIdeas) != 0 {
		t.Errorf("expected 0 active ideas, got %d", len(activeIdeas))
	}

	// Idea should be in done with promoted status
	doneIdeas, _ := store.ReadAll[map[string]json.RawMessage](filepath.Join(dir, "ideas.done.jsonl"))
	if len(doneIdeas) != 1 {
		t.Fatalf("expected 1 done idea, got %d", len(doneIdeas))
	}
	if string(doneIdeas[0]["status"]) != `"promoted"` {
		t.Errorf("expected status=promoted, got %s", string(doneIdeas[0]["status"]))
	}
	promotedTo := strings.Trim(string(doneIdeas[0]["promoted_to"]), `"`)
	if promotedTo != newID {
		t.Errorf("promoted_to=%s, expected %s", promotedTo, newID)
	}

	// New task should exist
	tasks, _ := store.ReadAll[model.Task](filepath.Join(dir, "tasks.jsonl"))
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Good idea" {
		t.Errorf("title: %s", tasks[0].Title)
	}
	if tasks[0].Priority != "p1" {
		t.Errorf("priority: %s", tasks[0].Priority)
	}
	if tasks[0].Source != "idea" {
		t.Errorf("source: %s", tasks[0].Source)
	}
	sourceRef := strings.Trim(string(tasks[0].SourceRef), `"`)
	if sourceRef != "idea-20260307-aaaa" {
		t.Errorf("source_ref: %s", sourceRef)
	}
	if len(tasks[0].Tags) != 1 || tasks[0].Tags[0] != "tag1" {
		t.Errorf("tags should inherit: %v", tasks[0].Tags)
	}
}

func TestRunPromoteToIssue(t *testing.T) {
	dir := setupTestDir(t)

	idea := model.NewIdea("idea-20260307-bbbb", "Bug idea", "this is a bug", []string{})
	store.Append(filepath.Join(dir, "ideas.jsonl"), idea)

	output := captureStdout(t, func() {
		err := RunPromote(dir, []string{"--id", "idea-20260307-bbbb", "--to", "issue", "--severity", "high"})
		if err != nil {
			t.Fatalf("RunPromote: %v", err)
		}
	})

	newID := strings.TrimSpace(output)
	if !strings.HasPrefix(newID, "issue-20260307-") {
		t.Errorf("expected issue ID, got: %s", newID)
	}

	issues, _ := store.ReadAll[model.Issue](filepath.Join(dir, "issues.jsonl"))
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != "high" {
		t.Errorf("severity: %s", issues[0].Severity)
	}
	if issues[0].Source != "idea" {
		t.Errorf("source: %s", issues[0].Source)
	}
}

func TestRunPromoteNotFound(t *testing.T) {
	dir := setupTestDir(t)

	err := RunPromote(dir, []string{"--id", "idea-20260307-9999", "--to", "task"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestRunPromoteMissingArgs(t *testing.T) {
	dir := setupTestDir(t)

	err := RunPromote(dir, []string{"--id", "idea-20260307-aaaa"})
	if err == nil {
		t.Fatal("expected error for missing --to")
	}
}
