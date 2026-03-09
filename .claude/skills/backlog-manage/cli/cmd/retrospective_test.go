package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func TestRunRetrospectiveNoData(t *testing.T) {
	dir := setupTestDir(t)

	output := captureStderr(t, func() {
		if err := RunRetrospective(dir, []string{}); err != nil {
			t.Fatalf("RunRetrospective: %v", err)
		}
	})

	if !strings.Contains(output, "no audit") {
		t.Errorf("expected 'no audit' message, got: %s", output)
	}
}

func TestRunRetrospectiveTextOutput(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	// Create 5 entries with some recurring failures
	entries := []model.AuditEntry{
		makeAuditEntry("eval-r01", 4, 5, []model.AuditFinding{
			{Check: "jsonl_integrity", Status: "pass", Detail: "ok"},
			{Check: "stale_ideas", Status: "fail", Detail: "stale", Patched: false},
			{Check: "backup_files", Status: "pass", Detail: "ok"},
			{Check: "md_summaries", Status: "pass", Detail: "ok"},
			{Check: "untracked_handoffs", Status: "pass", Detail: "ok"},
		}),
		makeAuditEntry("eval-r02", 3, 5, []model.AuditFinding{
			{Check: "jsonl_integrity", Status: "pass", Detail: "ok"},
			{Check: "stale_ideas", Status: "fail", Detail: "stale", Patched: true},
			{Check: "backup_files", Status: "fail", Detail: "found bak", Patched: false},
			{Check: "md_summaries", Status: "pass", Detail: "ok"},
			{Check: "untracked_handoffs", Status: "pass", Detail: "ok"},
		}),
		makeAuditEntry("eval-r03", 3, 5, []model.AuditFinding{
			{Check: "jsonl_integrity", Status: "pass", Detail: "ok"},
			{Check: "stale_ideas", Status: "fail", Detail: "stale", Patched: false},
			{Check: "backup_files", Status: "pass", Detail: "ok"},
			{Check: "md_summaries", Status: "pass", Detail: "ok"},
			{Check: "untracked_handoffs", Status: "warn", Detail: "1 handoff"},
		}),
		makeAuditEntry("eval-r04", 5, 5, []model.AuditFinding{
			{Check: "jsonl_integrity", Status: "pass", Detail: "ok"},
			{Check: "stale_ideas", Status: "pass", Detail: "ok"},
			{Check: "backup_files", Status: "pass", Detail: "ok"},
			{Check: "md_summaries", Status: "pass", Detail: "ok"},
			{Check: "untracked_handoffs", Status: "pass", Detail: "ok"},
		}),
		makeAuditEntry("eval-r05", 5, 5, []model.AuditFinding{
			{Check: "jsonl_integrity", Status: "pass", Detail: "ok"},
			{Check: "stale_ideas", Status: "pass", Detail: "ok"},
			{Check: "backup_files", Status: "pass", Detail: "ok"},
			{Check: "md_summaries", Status: "pass", Detail: "ok"},
			{Check: "untracked_handoffs", Status: "pass", Detail: "ok"},
		}),
	}

	for _, e := range entries {
		if err := store.Append(logPath, e); err != nil {
			t.Fatalf("store.Append: %v", err)
		}
	}

	output := captureStdout(t, func() {
		if err := RunRetrospective(dir, []string{"--last", "5"}); err != nil {
			t.Fatalf("RunRetrospective: %v", err)
		}
	})

	// stale_ideas failed 3 times -> recurring
	if !strings.Contains(output, "stale_ideas") {
		t.Errorf("expected 'stale_ideas' in recurring failures: %s", output)
	}
	// All-pass streak should be 2 (last 2 entries)
	if !strings.Contains(output, "2") {
		t.Errorf("expected all-pass streak of 2 in output: %s", output)
	}
}

func TestRunRetrospectiveJSONOutput(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	entries := []model.AuditEntry{
		makeAuditEntry("eval-j01", 4, 5, []model.AuditFinding{
			{Check: "check_a", Status: "fail", Detail: "x", Patched: false},
			{Check: "check_b", Status: "pass", Detail: "ok"},
			{Check: "check_c", Status: "pass", Detail: "ok"},
			{Check: "check_d", Status: "pass", Detail: "ok"},
			{Check: "check_e", Status: "pass", Detail: "ok"},
		}),
		makeAuditEntry("eval-j02", 4, 5, []model.AuditFinding{
			{Check: "check_a", Status: "fail", Detail: "x", Patched: false},
			{Check: "check_b", Status: "pass", Detail: "ok"},
			{Check: "check_c", Status: "pass", Detail: "ok"},
			{Check: "check_d", Status: "pass", Detail: "ok"},
			{Check: "check_e", Status: "pass", Detail: "ok"},
		}),
		makeAuditEntry("eval-j03", 4, 5, []model.AuditFinding{
			{Check: "check_a", Status: "fail", Detail: "x", Patched: false},
			{Check: "check_b", Status: "pass", Detail: "ok"},
			{Check: "check_c", Status: "pass", Detail: "ok"},
			{Check: "check_d", Status: "pass", Detail: "ok"},
			{Check: "check_e", Status: "pass", Detail: "ok"},
		}),
	}

	for _, e := range entries {
		if err := store.Append(logPath, e); err != nil {
			t.Fatalf("store.Append: %v", err)
		}
	}

	output := captureStdout(t, func() {
		if err := RunRetrospective(dir, []string{"--json"}); err != nil {
			t.Fatalf("RunRetrospective --json: %v", err)
		}
	})

	var result model.RetroResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %s", err, output)
	}

	// check_a failed 3 times -> recurring (threshold is 3)
	if count, ok := result.Recurring["check_a"]; !ok || count != 3 {
		t.Errorf("expected check_a recurring=3, got %v", result.Recurring)
	}

	// check_a unpatched 3 times
	if count, ok := result.Unpatched["check_a"]; !ok || count != 3 {
		t.Errorf("expected check_a unpatched=3, got %v", result.Unpatched)
	}

	// 3 score entries
	if len(result.Scores) != 3 {
		t.Errorf("expected 3 scores, got %d", len(result.Scores))
	}

	// All-pass streak: 0 (none are all-pass)
	if result.AllPassStreak != 0 {
		t.Errorf("expected all-pass streak 0, got %d", result.AllPassStreak)
	}

	if result.TotalRuns != 3 {
		t.Errorf("expected total_runs=3, got %d", result.TotalRuns)
	}
}

func TestRunRetrospectiveLastDefault(t *testing.T) {
	dir := setupTestDir(t)
	logPath := filepath.Join(dir, "audit-log.jsonl")

	// Create 12 entries, only last 10 should be analyzed by default
	for i := 0; i < 12; i++ {
		e := makeAuditEntry("eval-d"+string(rune('a'+i)), 5, 5, []model.AuditFinding{
			{Check: "c1", Status: "pass", Detail: "ok"},
		})
		if err := store.Append(logPath, e); err != nil {
			t.Fatalf("store.Append: %v", err)
		}
	}

	output := captureStdout(t, func() {
		if err := RunRetrospective(dir, []string{"--json"}); err != nil {
			t.Fatalf("RunRetrospective: %v", err)
		}
	})

	var result model.RetroResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if result.TotalRuns != 10 {
		t.Errorf("expected 10 runs (default --last), got %d", result.TotalRuns)
	}
}
