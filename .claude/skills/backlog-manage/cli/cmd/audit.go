package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/usadamasa/backlog-cli/internal/md"
	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunAudit(dir string, args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	last := fs.Bool("last", true, "Show only the latest entry")
	failures := fs.Bool("failures", false, "Show only fail/warn findings")
	run := fs.Bool("run", false, "Run health checks and append result to audit-log")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *run {
		findings := runHealthChecks(dir)
		entry := model.NewAuditEntry(model.GenerateID("audit"), findings, nil)
		if err := store.Append(filepath.Join(dir, "audit-log.jsonl"), entry); err != nil {
			return err
		}
		printEntry(entry, false)
		return nil
	}

	entries, err := store.ReadAll[model.AuditEntry](filepath.Join(dir, "audit-log.jsonl"))
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no prior eval")
		return nil
	}

	var selected []model.AuditEntry
	if *last {
		selected = entries[len(entries)-1:]
	} else {
		selected = entries
	}

	for _, e := range selected {
		printEntry(e, *failures)
	}

	return nil
}

func printEntry(e model.AuditEntry, failuresOnly bool) {
	fmt.Printf("=== %s (Score: %d/%d) ===\n", e.ID, e.Score.Passed, e.Score.Total)
	for _, f := range e.Findings {
		if failuresOnly && f.Status != "fail" && f.Status != "warn" {
			continue
		}
		fmt.Printf("  [%s] %s: %s\n", f.Status, f.Check, f.Detail)
	}
}

func runHealthChecks(dir string) []model.AuditFinding {
	var findings []model.AuditFinding

	// 1. jsonl_integrity
	findings = append(findings, checkJSONLIntegrity(dir))

	// 2. stale_ideas
	findings = append(findings, checkStaleIdeas(dir))

	// 3. backup_files
	findings = append(findings, checkBackupFiles(dir))

	// 4. md_summaries
	findings = append(findings, checkMDSummaries(dir))

	// 5. untracked_handoffs
	findings = append(findings, checkUntrackedHandoffs())

	return findings
}

func checkJSONLIntegrity(dir string) model.AuditFinding {
	files := []string{"tasks.jsonl", "ideas.jsonl", "issues.jsonl"}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		switch f {
		case "tasks.jsonl":
			if _, err := store.ReadAll[model.Task](path); err != nil {
				return model.AuditFinding{Check: "jsonl_integrity", Status: "fail", Detail: fmt.Sprintf("%s: %v", f, err)}
			}
		case "ideas.jsonl":
			if _, err := store.ReadAll[model.Idea](path); err != nil {
				return model.AuditFinding{Check: "jsonl_integrity", Status: "fail", Detail: fmt.Sprintf("%s: %v", f, err)}
			}
		case "issues.jsonl":
			if _, err := store.ReadAll[model.Issue](path); err != nil {
				return model.AuditFinding{Check: "jsonl_integrity", Status: "fail", Detail: fmt.Sprintf("%s: %v", f, err)}
			}
		}
	}
	return model.AuditFinding{Check: "jsonl_integrity", Status: "pass", Detail: "all JSONL files valid"}
}

func checkStaleIdeas(dir string) model.AuditFinding {
	ideas, err := store.ReadAll[model.Idea](filepath.Join(dir, "ideas.jsonl"))
	if err != nil {
		return model.AuditFinding{Check: "stale_ideas", Status: "fail", Detail: fmt.Sprintf("read error: %v", err)}
	}

	threshold := time.Now().UTC().Add(-30 * 24 * time.Hour)
	var stale []string
	for _, idea := range ideas {
		if idea.Status != "active" {
			continue
		}
		created, err := time.Parse(time.RFC3339, idea.CreatedAt)
		if err != nil {
			continue
		}
		if created.Before(threshold) {
			stale = append(stale, idea.ID)
		}
	}

	if len(stale) > 0 {
		return model.AuditFinding{Check: "stale_ideas", Status: "warn", Detail: fmt.Sprintf("%d stale ideas (>30 days): %v", len(stale), stale)}
	}
	return model.AuditFinding{Check: "stale_ideas", Status: "pass", Detail: "no stale ideas"}
}

func checkBackupFiles(dir string) model.AuditFinding {
	matches, err := filepath.Glob(filepath.Join(dir, "*.bak"))
	if err != nil {
		return model.AuditFinding{Check: "backup_files", Status: "fail", Detail: fmt.Sprintf("glob error: %v", err)}
	}
	if len(matches) > 0 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = filepath.Base(m)
		}
		return model.AuditFinding{Check: "backup_files", Status: "warn", Detail: fmt.Sprintf("%d backup files: %v", len(matches), names)}
	}
	return model.AuditFinding{Check: "backup_files", Status: "pass", Detail: "no backup files"}
}

func checkMDSummaries(dir string) model.AuditFinding {
	if err := md.Generate(dir); err != nil {
		return model.AuditFinding{Check: "md_summaries", Status: "fail", Detail: fmt.Sprintf("generate error: %v", err)}
	}
	return model.AuditFinding{Check: "md_summaries", Status: "pass", Detail: "MD summaries regenerated"}
}

func checkUntrackedHandoffs() model.AuditFinding {
	home, err := os.UserHomeDir()
	if err != nil {
		return model.AuditFinding{Check: "untracked_handoffs", Status: "pass", Detail: "cannot determine home dir, skipped"}
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", "memory", "SESSION_HANDOFF_*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return model.AuditFinding{Check: "untracked_handoffs", Status: "pass", Detail: "glob error, skipped"}
	}
	if len(matches) > 0 {
		return model.AuditFinding{Check: "untracked_handoffs", Status: "warn", Detail: fmt.Sprintf("%d untracked handoff files", len(matches))}
	}
	return model.AuditFinding{Check: "untracked_handoffs", Status: "pass", Detail: "no untracked handoffs"}
}
