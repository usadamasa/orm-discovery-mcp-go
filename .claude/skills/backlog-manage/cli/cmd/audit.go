package cmd

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/usadamasa/backlog-cli/internal/md"
	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunAudit(dir string, args []string) error {
	// Check for log-entry subcommand
	if len(args) > 0 && args[0] == "log-entry" {
		return runLogEntry(dir, args[1:])
	}

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

	// 5. unlinked_gh_issues
	findings = append(findings, checkUnlinkedGHIssues(dir))

	// 6. memory_duplicates
	findings = append(findings, checkMemoryDuplicates())

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

type execFunc func(args ...string) ([]byte, error)

func defaultGHExec(args ...string) ([]byte, error) {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return nil, err
	}
	return exec.Command(ghPath, args...).Output()
}

func checkUnlinkedGHIssues(dir string) model.AuditFinding {
	return checkUnlinkedGHIssuesWithExec(dir, defaultGHExec)
}

func checkUnlinkedGHIssuesWithExec(dir string, ghExec execFunc) model.AuditFinding {
	output, err := ghExec("issue", "list", "-R", "usadamasa/orm-discovery-mcp-go", "--label", "voc", "--json", "number,title")
	if err != nil {
		return model.AuditFinding{Check: "unlinked_gh_issues", Status: "pass", Detail: "gh unavailable, skip"}
	}

	var ghIssues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(output, &ghIssues); err != nil {
		return model.AuditFinding{Check: "unlinked_gh_issues", Status: "pass", Detail: fmt.Sprintf("gh output parse error, skip: %v", err)}
	}

	if len(ghIssues) == 0 {
		return model.AuditFinding{Check: "unlinked_gh_issues", Status: "pass", Detail: "no voc issues on GitHub"}
	}

	// Read issues.jsonl and collect linked github_issue numbers
	issues, err := store.ReadAll[model.Issue](filepath.Join(dir, "issues.jsonl"))
	if err != nil {
		return model.AuditFinding{Check: "unlinked_gh_issues", Status: "pass", Detail: fmt.Sprintf("issues.jsonl read error, skip: %v", err)}
	}

	linkedNumbers := make(map[int]bool)
	for _, iss := range issues {
		var num int
		if err := json.Unmarshal(iss.GitHubIssue, &num); err == nil && num > 0 {
			linkedNumbers[num] = true
		}
	}

	var unlinked []string
	for _, gh := range ghIssues {
		if !linkedNumbers[gh.Number] {
			unlinked = append(unlinked, fmt.Sprintf("#%d", gh.Number))
		}
	}

	if len(unlinked) > 0 {
		return model.AuditFinding{Check: "unlinked_gh_issues", Status: "warn", Detail: fmt.Sprintf("%d unlinked GH issues: %s", len(unlinked), strings.Join(unlinked, ", "))}
	}
	return model.AuditFinding{Check: "unlinked_gh_issues", Status: "pass", Detail: "all GH voc issues linked"}
}

func checkMemoryDuplicates() model.AuditFinding {
	home, err := os.UserHomeDir()
	if err != nil {
		return model.AuditFinding{Check: "memory_duplicates", Status: "pass", Detail: "cannot determine home dir, skip"}
	}
	memoryPath := filepath.Join(home, ".claude", "projects", "*", "memory", "MEMORY.md")
	matches, err := filepath.Glob(memoryPath)
	if err != nil || len(matches) == 0 {
		return model.AuditFinding{Check: "memory_duplicates", Status: "pass", Detail: "no MEMORY.md files found, skip"}
	}

	var allDuplicates []string
	for _, m := range matches {
		finding := checkMemoryDuplicatesFile(m)
		if finding.Status == "warn" {
			allDuplicates = append(allDuplicates, finding.Detail)
		}
	}

	if len(allDuplicates) > 0 {
		return model.AuditFinding{Check: "memory_duplicates", Status: "warn", Detail: strings.Join(allDuplicates, "; ")}
	}
	return model.AuditFinding{Check: "memory_duplicates", Status: "pass", Detail: "no duplicate sections in MEMORY.md"}
}

func checkMemoryDuplicatesFile(path string) model.AuditFinding {
	f, err := os.Open(path)
	if err != nil {
		return model.AuditFinding{Check: "memory_duplicates", Status: "pass", Detail: fmt.Sprintf("cannot read %s, skip", filepath.Base(path))}
	}
	defer f.Close()

	headers := make(map[string]int)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			header := strings.TrimSpace(line[3:])
			headers[header]++
		}
	}

	var duplicates []string
	for header, count := range headers {
		if count > 1 {
			duplicates = append(duplicates, fmt.Sprintf("%s (%dx)", header, count))
		}
	}

	if len(duplicates) > 0 {
		return model.AuditFinding{Check: "memory_duplicates", Status: "warn", Detail: fmt.Sprintf("duplicate sections: %s", strings.Join(duplicates, ", "))}
	}
	return model.AuditFinding{Check: "memory_duplicates", Status: "pass", Detail: "no duplicates"}
}

func runLogEntry(dir string, args []string) error {
	fs := flag.NewFlagSet("log-entry", flag.ContinueOnError)
	findingsJSON := fs.String("findings", "", "JSON array of findings")
	patchActionsJSON := fs.String("patch-actions", "", "JSON array of patch action descriptions")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *findingsJSON == "" {
		return fmt.Errorf("--findings is required")
	}

	var findings []model.AuditFinding
	if err := json.Unmarshal([]byte(*findingsJSON), &findings); err != nil {
		return fmt.Errorf("parse findings: %w", err)
	}

	var patchActions []string
	if *patchActionsJSON != "" {
		if err := json.Unmarshal([]byte(*patchActionsJSON), &patchActions); err != nil {
			return fmt.Errorf("parse patch-actions: %w", err)
		}
	}

	id := model.GenerateID("audit")
	entry := model.NewAuditEntry(id, findings, patchActions)
	if err := store.Append(filepath.Join(dir, "audit-log.jsonl"), entry); err != nil {
		return err
	}

	fmt.Println(id)
	return nil
}
