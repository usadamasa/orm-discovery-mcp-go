package critic

import (
	"context"
	"fmt"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// LargeDiffCritic detects pull requests with excessively large diffs.
type LargeDiffCritic struct {
	MaxChangedLines int
	MaxChangedFiles int
}

// NewLargeDiffCritic creates a new LargeDiffCritic with default thresholds.
func NewLargeDiffCritic() *LargeDiffCritic {
	return &LargeDiffCritic{
		MaxChangedLines: 500,
		MaxChangedFiles: 20,
	}
}

// Name returns the name of this critic.
func (c *LargeDiffCritic) Name() string {
	return "LargeDiffCritic"
}

// Review checks whether the diff exceeds configured thresholds for changed
// lines and changed files. Returns up to two Warning findings.
func (c *LargeDiffCritic) Review(_ context.Context, input ReviewInput) ([]model.Finding, error) {
	if input.Diff == nil {
		return nil, nil
	}

	var findings []model.Finding

	totalLines := input.Diff.TotalAdditions + input.Diff.TotalDeletions
	if totalLines > c.MaxChangedLines {
		f := model.NewFinding(
			model.SeverityWarning,
			model.CategoryLargeDiff,
			fmt.Sprintf("diff has %d changed lines, exceeding threshold of %d", totalLines, c.MaxChangedLines),
			model.Location{},
		)
		f.CriticName = c.Name()
		f.Suggestion = "consider splitting into smaller, focused pull requests"
		f.Metadata = map[string]any{
			"actual":    totalLines,
			"threshold": c.MaxChangedLines,
		}
		findings = append(findings, f)
	}

	totalFiles := len(input.Diff.Files)
	if totalFiles > c.MaxChangedFiles {
		f := model.NewFinding(
			model.SeverityWarning,
			model.CategoryLargeDiff,
			fmt.Sprintf("diff has %d changed files, exceeding threshold of %d", totalFiles, c.MaxChangedFiles),
			model.Location{},
		)
		f.CriticName = c.Name()
		f.Suggestion = "consider splitting into smaller, focused pull requests"
		f.Metadata = map[string]any{
			"actual":    totalFiles,
			"threshold": c.MaxChangedFiles,
		}
		findings = append(findings, f)
	}

	return findings, nil
}
