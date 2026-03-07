package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestRunListAll(t *testing.T) {
	dir := setupTestDir(t)

	store.Append(filepath.Join(dir, "tasks.jsonl"), model.NewTask("task-20260307-0001", "Task1", "d", "p1", []string{}))
	store.Append(filepath.Join(dir, "ideas.jsonl"), model.NewIdea("idea-20260307-0001", "Idea1", "d", []string{}))
	store.Append(filepath.Join(dir, "issues.jsonl"), model.NewIssue("issue-20260307-0001", "Issue1", "d", "high", []string{}))

	output := captureStdout(t, func() {
		if err := RunList(dir, []string{}); err != nil {
			t.Fatalf("RunList: %v", err)
		}
	})

	if !strings.Contains(output, "=== Tasks ===") {
		t.Error("missing Tasks header")
	}
	if !strings.Contains(output, "[p1] task-20260307-0001: Task1 (active)") {
		t.Errorf("missing task line in:\n%s", output)
	}
	if !strings.Contains(output, "=== Ideas ===") {
		t.Error("missing Ideas header")
	}
	if !strings.Contains(output, "idea-20260307-0001: Idea1 (active)") {
		t.Errorf("missing idea line in:\n%s", output)
	}
	if !strings.Contains(output, "=== Issues ===") {
		t.Error("missing Issues header")
	}
	if !strings.Contains(output, "[high] issue-20260307-0001: Issue1 (active)") {
		t.Errorf("missing issue line in:\n%s", output)
	}
}

func TestRunListFilterType(t *testing.T) {
	dir := setupTestDir(t)

	store.Append(filepath.Join(dir, "tasks.jsonl"), model.NewTask("task-20260307-0001", "Task1", "d", "p1", []string{}))
	store.Append(filepath.Join(dir, "ideas.jsonl"), model.NewIdea("idea-20260307-0001", "Idea1", "d", []string{}))

	output := captureStdout(t, func() {
		if err := RunList(dir, []string{"--type", "task"}); err != nil {
			t.Fatalf("RunList: %v", err)
		}
	})

	if !strings.Contains(output, "=== Tasks ===") {
		t.Error("missing Tasks header")
	}
	if strings.Contains(output, "=== Ideas ===") {
		t.Error("should not show Ideas")
	}
	if strings.Contains(output, "=== Issues ===") {
		t.Error("should not show Issues")
	}
}

func TestRunListEmpty(t *testing.T) {
	dir := setupTestDir(t)

	output := captureStdout(t, func() {
		if err := RunList(dir, []string{}); err != nil {
			t.Fatalf("RunList: %v", err)
		}
	})

	if !strings.Contains(output, "(none)") {
		t.Errorf("expected (none) in empty list:\n%s", output)
	}
}
