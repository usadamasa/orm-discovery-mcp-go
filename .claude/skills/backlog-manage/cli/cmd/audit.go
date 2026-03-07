package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/usadamasa/backlog-cli/internal/model"
	"github.com/usadamasa/backlog-cli/internal/store"
)

func RunAudit(dir string, args []string) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	last := fs.Bool("last", true, "Show only the latest entry")
	failures := fs.Bool("failures", false, "Show only fail/warn findings")
	if err := fs.Parse(args); err != nil {
		return err
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
		fmt.Printf("=== %s (Score: %d/%d) ===\n", e.ID, e.Score.Passed, e.Score.Total)
		for _, f := range e.Findings {
			if *failures && f.Status != "fail" && f.Status != "warn" {
				continue
			}
			fmt.Printf("  [%s] %s: %s\n", f.Status, f.Check, f.Detail)
		}
	}

	return nil
}
