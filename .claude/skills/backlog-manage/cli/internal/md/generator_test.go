package md

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func TestGenerateAllFiles(t *testing.T) {
	dir := t.TempDir()

	// Seed data
	store.Append(filepath.Join(dir, "tasks.jsonl"), model.NewTask("task-20260307-0001", "High prio task", "d", "p1", []string{"tag1"}))
	store.Append(filepath.Join(dir, "tasks.jsonl"), model.NewTask("task-20260307-0002", "Normal task", "d", "p2", []string{}))
	store.Append(filepath.Join(dir, "tasks.done.jsonl"), model.NewTask("task-20260307-0003", "Done task", "d", "p2", []string{}))
	store.Append(filepath.Join(dir, "ideas.jsonl"), model.NewIdea("idea-20260307-0001", "Cool idea", "d", []string{"ai"}))
	store.Append(filepath.Join(dir, "issues.jsonl"), model.NewIssue("issue-20260307-0001", "Big bug", "d", "high", []string{"auth"}))
	store.Append(filepath.Join(dir, "issues.jsonl"), model.NewIssue("issue-20260307-0002", "Small bug", "d", "low", []string{}))

	if err := Generate(dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check README.md
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile README: %v", err)
	}
	readmeStr := string(readme)
	if !strings.Contains(readmeStr, "# Backlog Summary") {
		t.Error("README missing header")
	}
	if !strings.Contains(readmeStr, "| Tasks | 2 | 1 |") {
		t.Errorf("README wrong task count:\n%s", readmeStr)
	}
	if !strings.Contains(readmeStr, "| Ideas | 1 | 0 |") {
		t.Errorf("README wrong idea count:\n%s", readmeStr)
	}
	if !strings.Contains(readmeStr, "| Issues | 2 | 0 |") {
		t.Errorf("README wrong issue count:\n%s", readmeStr)
	}

	// Check TASKS.md
	tasksmd, _ := os.ReadFile(filepath.Join(dir, "TASKS.md"))
	tasksStr := string(tasksmd)
	if !strings.Contains(tasksStr, "## P1") {
		t.Error("TASKS missing P1 section")
	}
	if !strings.Contains(tasksStr, "task-20260307-0001") {
		t.Error("TASKS missing p1 task")
	}
	if !strings.Contains(tasksStr, "## P2") {
		t.Error("TASKS missing P2 section")
	}
	if !strings.Contains(tasksStr, "tag1") {
		t.Error("TASKS missing tag")
	}

	// Check IDEAS.md
	ideasmd, _ := os.ReadFile(filepath.Join(dir, "IDEAS.md"))
	ideasStr := string(ideasmd)
	if !strings.Contains(ideasStr, "idea-20260307-0001") {
		t.Error("IDEAS missing idea")
	}
	if !strings.Contains(ideasStr, "ai") {
		t.Error("IDEAS missing tag")
	}

	// Check ISSUES.md
	issuesmd, _ := os.ReadFile(filepath.Join(dir, "ISSUES.md"))
	issuesStr := string(issuesmd)
	if !strings.Contains(issuesStr, "## High") {
		t.Error("ISSUES missing High section")
	}
	if !strings.Contains(issuesStr, "issue-20260307-0001") {
		t.Error("ISSUES missing high issue")
	}
	if !strings.Contains(issuesStr, "## Low") {
		t.Error("ISSUES missing Low section")
	}
	if !strings.Contains(issuesStr, "issue-20260307-0002") {
		t.Error("ISSUES missing low issue")
	}
}

func TestGenerateEmpty(t *testing.T) {
	dir := t.TempDir()

	if err := Generate(dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "| Tasks | 0 | 0 |") {
		t.Errorf("README should show 0 counts:\n%s", string(readme))
	}

	tasksmd, _ := os.ReadFile(filepath.Join(dir, "TASKS.md"))
	if !strings.Contains(string(tasksmd), "| - | - | - | - |") {
		t.Error("TASKS should show empty placeholder")
	}
}

func TestGenerateNoTmpLeftOver(t *testing.T) {
	dir := t.TempDir()

	Generate(dir)

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("tmp file left over: %s", e.Name())
		}
	}
}

func TestGenerateTimestamp(t *testing.T) {
	dir := t.TempDir()

	Generate(dir)

	readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(readme), "> Generated:") {
		t.Error("README missing Generated timestamp")
	}
}

func TestGenerateSkipsWriteWhenContentUnchanged(t *testing.T) {
	dir := t.TempDir()

	// First generation
	if err := Generate(dir); err != nil {
		t.Fatalf("first Generate: %v", err)
	}

	// Record modtime of all generated files
	files := []string{"README.md", "TASKS.md", "IDEAS.md", "ISSUES.md"}
	modTimes := make(map[string]int64)
	for _, f := range files {
		info, err := os.Stat(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("stat %s: %v", f, err)
		}
		modTimes[f] = info.ModTime().UnixNano()
	}

	// Second generation (JSONL unchanged, only timestamp differs)
	if err := Generate(dir); err != nil {
		t.Fatalf("second Generate: %v", err)
	}

	// Verify files were NOT rewritten
	for _, f := range files {
		info, err := os.Stat(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("stat %s after second gen: %v", f, err)
		}
		if info.ModTime().UnixNano() != modTimes[f] {
			t.Errorf("%s was rewritten despite unchanged content", f)
		}
	}
}

func TestStripTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with timestamp",
			input: "> Generated: 2026-03-07 12:00\n\n# Backlog Summary\n",
			want:  "\n\n# Backlog Summary\n",
		},
		{
			name:  "without timestamp",
			input: "# Backlog Summary\n",
			want:  "# Backlog Summary\n",
		},
		{
			name:  "timestamp only",
			input: "> Generated: 2026-03-07 12:00",
			want:  "> Generated: 2026-03-07 12:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripTimestamp(tt.input)
			if got != tt.want {
				t.Errorf("stripTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
