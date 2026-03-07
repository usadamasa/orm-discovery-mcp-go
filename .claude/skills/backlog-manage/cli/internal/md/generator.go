package md

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

// Generate creates all MD summary files in the given directory.
func Generate(dir string) error {
	tasks, err := store.ReadAll[model.Task](dir + "/tasks.jsonl")
	if err != nil {
		return err
	}
	doneTasks, err := store.ReadAll[model.Task](dir + "/tasks.done.jsonl")
	if err != nil {
		return err
	}
	ideas, err := store.ReadAll[model.Idea](dir + "/ideas.jsonl")
	if err != nil {
		return err
	}
	doneIdeas, err := store.ReadAll[model.Idea](dir + "/ideas.done.jsonl")
	if err != nil {
		return err
	}
	issues, err := store.ReadAll[model.Issue](dir + "/issues.jsonl")
	if err != nil {
		return err
	}
	doneIssues, err := store.ReadAll[model.Issue](dir + "/issues.done.jsonl")
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02 15:04")

	if err := writeAtomic(dir+"/README.md", generateREADME(tasks, doneTasks, ideas, doneIdeas, issues, doneIssues, now)); err != nil {
		return err
	}
	if err := writeAtomic(dir+"/TASKS.md", generateTASKS(tasks, now)); err != nil {
		return err
	}
	if err := writeAtomic(dir+"/IDEAS.md", generateIDEAS(ideas, now)); err != nil {
		return err
	}
	if err := writeAtomic(dir+"/ISSUES.md", generateISSUES(issues, now)); err != nil {
		return err
	}

	return nil
}

func writeAtomic(path, content string) error {
	existing, err := os.ReadFile(path)
	if err == nil && stripTimestamp(string(existing)) == stripTimestamp(content) {
		return nil
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", tmpPath, err)
	}
	return os.Rename(tmpPath, path)
}

func stripTimestamp(s string) string {
	if after, ok := strings.CutPrefix(s, "> Generated:"); ok {
		if idx := strings.Index(after, "\n"); idx >= 0 {
			return after[idx:]
		}
	}
	return s
}

func generateREADME(tasks []model.Task, doneTasks []model.Task, ideas []model.Idea, doneIdeas []model.Idea, issues []model.Issue, doneIssues []model.Issue, now string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "> Generated: %s\n\n", now)
	b.WriteString("# Backlog Summary\n\n")
	b.WriteString("| Type | Active | Done |\n")
	b.WriteString("|------|--------|------|\n")
	fmt.Fprintf(&b, "| Tasks | %d | %d |\n", len(tasks), len(doneTasks))
	fmt.Fprintf(&b, "| Ideas | %d | %d |\n", len(ideas), len(doneIdeas))
	fmt.Fprintf(&b, "| Issues | %d | %d |\n", len(issues), len(doneIssues))
	return b.String()
}

func generateTASKS(tasks []model.Task, now string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "> Generated: %s\n\n", now)
	b.WriteString("# Tasks\n\n")

	for _, prio := range []string{"p1", "p2", "p3"} {
		b.WriteString("## " + strings.ToUpper(prio) + "\n\n")
		b.WriteString("| ID | Title | Tags | Source |\n")
		b.WriteString("|----|-------|------|--------|\n")
		found := false
		for _, t := range tasks {
			if t.Priority == prio {
				tags := strings.Join(t.Tags, ", ")
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", t.ID, t.Title, tags, t.Source)
				found = true
			}
		}
		if !found {
			b.WriteString("| - | - | - | - |\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func generateIDEAS(ideas []model.Idea, now string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "> Generated: %s\n\n", now)
	b.WriteString("# Ideas\n\n")
	b.WriteString("| ID | Title | Tags | Status |\n")
	b.WriteString("|----|-------|------|--------|\n")
	if len(ideas) == 0 {
		b.WriteString("| - | - | - | - |\n")
	}
	for _, i := range ideas {
		tags := strings.Join(i.Tags, ", ")
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", i.ID, i.Title, tags, i.Status)
	}
	return b.String()
}

func generateISSUES(issues []model.Issue, now string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "> Generated: %s\n\n", now)
	b.WriteString("# Issues\n\n")

	for _, sev := range []string{"high", "medium", "low"} {
		title := strings.ToUpper(sev[:1]) + sev[1:]
		b.WriteString("## " + title + "\n\n")
		b.WriteString("| ID | Title | Tags | GH Issue |\n")
		b.WriteString("|----|-------|------|----------|\n")
		found := false
		for _, i := range issues {
			if i.Severity == sev {
				tags := strings.Join(i.Tags, ", ")
				gh := string(i.GitHubIssue)
				if gh == "null" {
					gh = "-"
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", i.ID, i.Title, tags, gh)
				found = true
			}
		}
		if !found {
			b.WriteString("| - | - | - | - |\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}
