package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	out, _ := io.ReadAll(r)
	return string(out)
}

func makeAuditEntry(id string, passed, total int, findings []model.AuditFinding) model.AuditEntry {
	return model.AuditEntry{
		ID:       id,
		RunAt:    "2026-03-07T12:00:00Z",
		Score:    model.AuditScore{Passed: passed, Total: total},
		Findings: findings,
	}
}

func TestRunAuditLast(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	entry1 := makeAuditEntry("eval-001", 3, 4, []model.AuditFinding{
		{Check: "B1", Status: "pass", Detail: "ok"},
		{Check: "B2", Status: "fail", Detail: "DFS not used"},
		{Check: "B3", Status: "pass", Detail: "ok"},
		{Check: "B4", Status: "pass", Detail: "ok"},
	})
	entry2 := makeAuditEntry("eval-002", 4, 4, []model.AuditFinding{
		{Check: "B1", Status: "pass", Detail: "ok"},
		{Check: "B2", Status: "pass", Detail: "ok"},
		{Check: "B3", Status: "warn", Detail: "no citation"},
		{Check: "B4", Status: "pass", Detail: "ok"},
	})

	if err := store.Append(logPath, entry1); err != nil {
		t.Fatalf("store.Append: %v", err)
	}
	if err := store.Append(logPath, entry2); err != nil {
		t.Fatalf("store.Append: %v", err)
	}

	output := captureStdout(t, func() {
		if err := RunAudit(dir, []string{}); err != nil {
			t.Fatalf("RunAudit: %v", err)
		}
	})

	// --last (default) should show only the latest entry
	if !strings.Contains(output, "eval-002") {
		t.Errorf("expected eval-002 in output:\n%s", output)
	}
	if strings.Contains(output, "eval-001") {
		t.Errorf("should not show eval-001 with --last:\n%s", output)
	}
	if !strings.Contains(output, "[pass] B1: ok") {
		t.Errorf("missing finding line:\n%s", output)
	}
	if !strings.Contains(output, "[warn] B3: no citation") {
		t.Errorf("missing warn finding:\n%s", output)
	}
}

func TestRunAuditFailuresOnly(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	entry := makeAuditEntry("eval-003", 2, 4, []model.AuditFinding{
		{Check: "B1", Status: "pass", Detail: "ok"},
		{Check: "B2", Status: "fail", Detail: "DFS not used"},
		{Check: "B3", Status: "warn", Detail: "no citation"},
		{Check: "B4", Status: "pass", Detail: "ok"},
	})
	if err := store.Append(logPath, entry); err != nil {
		t.Fatalf("store.Append: %v", err)
	}

	output := captureStdout(t, func() {
		if err := RunAudit(dir, []string{"--failures"}); err != nil {
			t.Fatalf("RunAudit: %v", err)
		}
	})

	if !strings.Contains(output, "[fail] B2: DFS not used") {
		t.Errorf("missing fail finding:\n%s", output)
	}
	if !strings.Contains(output, "[warn] B3: no citation") {
		t.Errorf("missing warn finding:\n%s", output)
	}
	if strings.Contains(output, "[pass] B1") {
		t.Errorf("should not show pass findings with --failures:\n%s", output)
	}
}

func TestRunAuditNoLogFile(t *testing.T) {
	dir := setupTestDir(t)

	output := captureStderr(t, func() {
		if err := RunAudit(dir, []string{}); err != nil {
			t.Fatalf("RunAudit: %v", err)
		}
	})

	if !strings.Contains(output, "no prior eval") {
		t.Errorf("expected 'no prior eval' on stderr:\n%s", output)
	}
}

func TestRunAuditScoreLine(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	entry := makeAuditEntry("eval-004", 3, 4, []model.AuditFinding{
		{Check: "B1", Status: "pass", Detail: "ok"},
	})
	if err := store.Append(logPath, entry); err != nil {
		t.Fatalf("store.Append: %v", err)
	}

	output := captureStdout(t, func() {
		if err := RunAudit(dir, []string{}); err != nil {
			t.Fatalf("RunAudit: %v", err)
		}
	})

	if !strings.Contains(output, "Score: 3/4") {
		t.Errorf("expected score line:\n%s", output)
	}
}

func TestRunAuditRunClean(t *testing.T) {
	dir := setupTestDir(t)
	// Create valid JSONL files
	task := model.NewTask(model.GenerateID("task"), "test task", "desc", "p2", nil)
	if err := store.Append(filepath.Join(dir, "tasks.jsonl"), task); err != nil {
		t.Fatalf("store.Append tasks: %v", err)
	}
	idea := model.NewIdea(model.GenerateID("idea"), "test idea", "desc", nil)
	if err := store.Append(filepath.Join(dir, "ideas.jsonl"), idea); err != nil {
		t.Fatalf("store.Append ideas: %v", err)
	}
	issue := model.NewIssue(model.GenerateID("issue"), "test issue", "desc", "low", nil)
	if err := store.Append(filepath.Join(dir, "issues.jsonl"), issue); err != nil {
		t.Fatalf("store.Append issues: %v", err)
	}

	output := captureStdout(t, func() {
		if err := RunAudit(dir, []string{"--run"}); err != nil {
			t.Fatalf("RunAudit --run: %v", err)
		}
	})

	// Should create audit-log.jsonl with an entry
	entries, err := store.ReadAll[model.AuditEntry](filepath.Join(dir, "audit-log.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll audit-log: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
	}

	// All checks except untracked_handoffs should pass in a clean state
	// (untracked_handoffs depends on actual home directory state)
	for _, f := range entries[0].Findings {
		if f.Check == "untracked_handoffs" {
			continue
		}
		if f.Status != "pass" {
			t.Errorf("expected pass for %s, got %s: %s", f.Check, f.Status, f.Detail)
		}
	}

	if !strings.Contains(output, "jsonl_integrity") {
		t.Errorf("expected jsonl_integrity in output:\n%s", output)
	}
}

func TestRunAuditRunStaleIdea(t *testing.T) {
	dir := setupTestDir(t)
	// Create valid JSONL files
	task := model.NewTask(model.GenerateID("task"), "test task", "desc", "p2", nil)
	if err := store.Append(filepath.Join(dir, "tasks.jsonl"), task); err != nil {
		t.Fatalf("store.Append tasks: %v", err)
	}
	issue := model.NewIssue(model.GenerateID("issue"), "test issue", "desc", "low", nil)
	if err := store.Append(filepath.Join(dir, "issues.jsonl"), issue); err != nil {
		t.Fatalf("store.Append issues: %v", err)
	}

	// Create idea with created_at 31 days ago
	oldTime := time.Now().UTC().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	staleIdea := model.Idea{
		ID:          "idea-20260101-0001",
		Type:        "idea",
		Title:       "stale idea",
		Description: "old",
		Status:      "active",
		Tags:        []string{},
		Source:      "manual",
		SourceRef:   json.RawMessage("null"),
		PromotedTo:  json.RawMessage("null"),
		CreatedAt:   oldTime,
		CreatedBy:   "manual",
		DoneAt:      json.RawMessage("null"),
	}
	if err := store.Append(filepath.Join(dir, "ideas.jsonl"), staleIdea); err != nil {
		t.Fatalf("store.Append ideas: %v", err)
	}

	captureStdout(t, func() {
		if err := RunAudit(dir, []string{"--run"}); err != nil {
			t.Fatalf("RunAudit --run: %v", err)
		}
	})

	entries, err := store.ReadAll[model.AuditEntry](filepath.Join(dir, "audit-log.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll audit-log: %v", err)
	}

	found := false
	for _, f := range entries[0].Findings {
		if f.Check == "stale_ideas" {
			found = true
			if f.Status != "warn" {
				t.Errorf("expected warn for stale_ideas, got %s: %s", f.Status, f.Detail)
			}
		}
	}
	if !found {
		t.Error("stale_ideas check not found in findings")
	}
}

func TestRunAuditRunBackupFile(t *testing.T) {
	dir := setupTestDir(t)
	// Create valid JSONL files
	task := model.NewTask(model.GenerateID("task"), "test task", "desc", "p2", nil)
	if err := store.Append(filepath.Join(dir, "tasks.jsonl"), task); err != nil {
		t.Fatalf("store.Append tasks: %v", err)
	}
	idea := model.NewIdea(model.GenerateID("idea"), "test idea", "desc", nil)
	if err := store.Append(filepath.Join(dir, "ideas.jsonl"), idea); err != nil {
		t.Fatalf("store.Append ideas: %v", err)
	}
	issue := model.NewIssue(model.GenerateID("issue"), "test issue", "desc", "low", nil)
	if err := store.Append(filepath.Join(dir, "issues.jsonl"), issue); err != nil {
		t.Fatalf("store.Append issues: %v", err)
	}

	// Create a .bak file
	bakPath := filepath.Join(dir, "tasks.jsonl.bak")
	if err := os.WriteFile(bakPath, []byte("backup"), 0644); err != nil {
		t.Fatalf("WriteFile bak: %v", err)
	}

	captureStdout(t, func() {
		if err := RunAudit(dir, []string{"--run"}); err != nil {
			t.Fatalf("RunAudit --run: %v", err)
		}
	})

	entries, err := store.ReadAll[model.AuditEntry](filepath.Join(dir, "audit-log.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll audit-log: %v", err)
	}

	found := false
	for _, f := range entries[0].Findings {
		if f.Check == "backup_files" {
			found = true
			if f.Status != "warn" {
				t.Errorf("expected warn for backup_files, got %s: %s", f.Status, f.Detail)
			}
		}
	}
	if !found {
		t.Error("backup_files check not found in findings")
	}
}

func TestRunAuditAll(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	entry1 := makeAuditEntry("eval-010", 2, 3, []model.AuditFinding{
		{Check: "A1", Status: "pass", Detail: "ok"},
	})
	entry2 := makeAuditEntry("eval-011", 3, 3, []model.AuditFinding{
		{Check: "A1", Status: "pass", Detail: "ok"},
	})
	if err := store.Append(logPath, entry1); err != nil {
		t.Fatalf("store.Append: %v", err)
	}
	if err := store.Append(logPath, entry2); err != nil {
		t.Fatalf("store.Append: %v", err)
	}

	output := captureStdout(t, func() {
		if err := RunAudit(dir, []string{"--last=false"}); err != nil {
			t.Fatalf("RunAudit: %v", err)
		}
	})

	if !strings.Contains(output, "eval-010") {
		t.Errorf("expected eval-010 in output:\n%s", output)
	}
	if !strings.Contains(output, "eval-011") {
		t.Errorf("expected eval-011 in output:\n%s", output)
	}
}
