package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
