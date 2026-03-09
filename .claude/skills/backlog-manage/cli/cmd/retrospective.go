package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunRetrospective(dir string, args []string) error {
	fs := flag.NewFlagSet("retrospective", flag.ContinueOnError)
	last := fs.Int("last", 10, "Analyze last N audit entries")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	entries, err := store.ReadAll[model.AuditEntry](filepath.Join(dir, "audit-log.jsonl"))
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "no audit entries")
		return nil
	}

	// Take last N entries
	if *last > 0 && *last < len(entries) {
		entries = entries[len(entries)-*last:]
	}

	result := analyze(entries)

	if *jsonOut {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	printRetroText(result)
	return nil
}

func analyze(entries []model.AuditEntry) model.RetroResult {
	failCounts := make(map[string]int)
	unpatchedCounts := make(map[string]int)
	var scores []model.AuditScore

	for _, e := range entries {
		scores = append(scores, e.Score)
		for _, f := range e.Findings {
			if f.Status == "fail" || f.Status == "warn" {
				failCounts[f.Check]++
				if !f.Patched {
					unpatchedCounts[f.Check]++
				}
			}
		}
	}

	// Filter recurring: 3+ failures
	recurring := make(map[string]int)
	for check, count := range failCounts {
		if count >= 3 {
			recurring[check] = count
		}
	}

	// Calculate all-pass streak from the end
	allPassStreak := 0
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Score.Passed == entries[i].Score.Total {
			allPassStreak++
		} else {
			break
		}
	}

	return model.RetroResult{
		Recurring:     recurring,
		Unpatched:     unpatchedCounts,
		Scores:        scores,
		AllPassStreak: allPassStreak,
		TotalRuns:     len(entries),
	}
}

func printRetroText(r model.RetroResult) {
	fmt.Printf("=== Retrospective (%d runs) ===\n", r.TotalRuns)

	fmt.Printf("\nAll-pass streak: %d\n", r.AllPassStreak)

	if len(r.Recurring) > 0 {
		fmt.Println("\nRecurring failures (3+):")
		for check, count := range r.Recurring {
			fmt.Printf("  %s: %d times\n", check, count)
		}
	} else {
		fmt.Println("\nNo recurring failures")
	}

	if len(r.Unpatched) > 0 {
		fmt.Println("\nUnpatched failures:")
		for check, count := range r.Unpatched {
			fmt.Printf("  %s: %d unpatched\n", check, count)
		}
	}

	fmt.Println("\nScore trend:")
	for i, s := range r.Scores {
		fmt.Printf("  [%d] %d/%d\n", i+1, s.Passed, s.Total)
	}
}
