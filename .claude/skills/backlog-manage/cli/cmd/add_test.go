package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Fix time for deterministic IDs
	model.NowFunc = func() time.Time {
		return time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		model.NowFunc = func() time.Time { return time.Now().UTC() }
	})
	return dir
}

func TestRunAddTask(t *testing.T) {
	dir := setupTestDir(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RunAdd(dir, []string{"task", "--title", "Test task", "--description", "A test", "--priority", "p1", "--tags", "tag1,tag2"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("RunAdd task: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))

	if !strings.HasPrefix(output, "task-20260307-") {
		t.Errorf("expected task ID, got: %s", output)
	}

	// Verify JSONL
	tasks, err := store.ReadAll[model.Task](filepath.Join(dir, "tasks.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.Title != "Test task" {
		t.Errorf("title: %s", task.Title)
	}
	if task.Priority != "p1" {
		t.Errorf("priority: %s", task.Priority)
	}
	if len(task.Tags) != 2 || task.Tags[0] != "tag1" {
		t.Errorf("tags: %v", task.Tags)
	}
	if task.Status != "active" {
		t.Errorf("status: %s", task.Status)
	}
	if string(task.SourceRef) != "null" {
		t.Errorf("source_ref: %s", string(task.SourceRef))
	}
}

func TestRunAddIdea(t *testing.T) {
	dir := setupTestDir(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RunAdd(dir, []string{"idea", "--title", "Test idea", "--description", "An idea"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("RunAdd idea: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(output, "idea-20260307-") {
		t.Errorf("expected idea ID, got: %s", output)
	}

	ideas, err := store.ReadAll[model.Idea](filepath.Join(dir, "ideas.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("expected 1 idea, got %d", len(ideas))
	}
	if ideas[0].Title != "Test idea" {
		t.Errorf("title: %s", ideas[0].Title)
	}
}

func TestRunAddIssue(t *testing.T) {
	dir := setupTestDir(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RunAdd(dir, []string{"issue", "--title", "Test issue", "--description", "An issue", "--severity", "high"})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("RunAdd issue: %v", err)
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(output, "issue-20260307-") {
		t.Errorf("expected issue ID, got: %s", output)
	}

	issues, err := store.ReadAll[model.Issue](filepath.Join(dir, "issues.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != "high" {
		t.Errorf("severity: %s", issues[0].Severity)
	}
}

func TestRunAddMissingArgs(t *testing.T) {
	dir := setupTestDir(t)
	err := RunAdd(dir, []string{"task", "--title", "No desc"})
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestRunAddTaskTagsEmptyArray(t *testing.T) {
	dir := setupTestDir(t)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	err := RunAdd(dir, []string{"task", "--title", "No tags", "--description", "test"})
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	// Verify tags is [] not null in JSON
	data, _ := os.ReadFile(filepath.Join(dir, "tasks.jsonl"))
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)
	if string(raw["tags"]) != "[]" {
		t.Errorf("expected tags=[], got %s", string(raw["tags"]))
	}
}
