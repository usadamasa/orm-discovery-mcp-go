package cmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func TestRunCompleteTask(t *testing.T) {
	dir := setupTestDir(t)

	// Seed a task
	task := model.NewTask("task-20260307-aaaa", "Test", "desc", "p2", []string{})
	if err := store.Append(filepath.Join(dir, "tasks.jsonl"), task); err != nil {
		t.Fatal(err)
	}

	err := RunComplete(dir, []string{"task-20260307-aaaa"})
	if err != nil {
		t.Fatalf("RunComplete: %v", err)
	}

	// Active should be empty
	active, _ := store.ReadAll[model.Task](filepath.Join(dir, "tasks.jsonl"))
	if len(active) != 0 {
		t.Errorf("expected 0 active, got %d", len(active))
	}

	// Done should have 1 entry with status=done
	done, _ := store.ReadAll[map[string]json.RawMessage](filepath.Join(dir, "tasks.done.jsonl"))
	if len(done) != 1 {
		t.Fatalf("expected 1 done, got %d", len(done))
	}
	if string(done[0]["status"]) != `"done"` {
		t.Errorf("expected status=done, got %s", string(done[0]["status"]))
	}
	if string(done[0]["done_at"]) == "null" {
		t.Error("expected done_at to be set")
	}
}

func TestRunCompleteIssue(t *testing.T) {
	dir := setupTestDir(t)

	issue := model.NewIssue("issue-20260307-bbbb", "Bug", "fix", "high", []string{})
	if err := store.Append(filepath.Join(dir, "issues.jsonl"), issue); err != nil {
		t.Fatal(err)
	}

	err := RunComplete(dir, []string{"issue-20260307-bbbb"})
	if err != nil {
		t.Fatalf("RunComplete: %v", err)
	}

	active, _ := store.ReadAll[model.Issue](filepath.Join(dir, "issues.jsonl"))
	if len(active) != 0 {
		t.Errorf("expected 0 active, got %d", len(active))
	}

	done, _ := store.ReadAll[map[string]json.RawMessage](filepath.Join(dir, "issues.done.jsonl"))
	if len(done) != 1 {
		t.Fatalf("expected 1 done, got %d", len(done))
	}
	if string(done[0]["status"]) != `"resolved"` {
		t.Errorf("expected status=resolved, got %s", string(done[0]["status"]))
	}
	if string(done[0]["resolved_at"]) == "null" {
		t.Error("expected resolved_at to be set")
	}
}

func TestRunCompleteNotFound(t *testing.T) {
	dir := setupTestDir(t)

	task := model.NewTask("task-20260307-cccc", "Test", "desc", "p2", []string{})
	store.Append(filepath.Join(dir, "tasks.jsonl"), task)

	err := RunComplete(dir, []string{"task-20260307-9999"})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestRunCompleteUnknownPrefix(t *testing.T) {
	dir := setupTestDir(t)
	err := RunComplete(dir, []string{"unknown-123"})
	if err == nil {
		t.Fatal("expected error for unknown prefix")
	}
}
