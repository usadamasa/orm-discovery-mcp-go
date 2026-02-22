// Package reviewer provides orchestration for running multiple Critics
// against code changes and aggregating their findings.
package reviewer

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// Orchestrator coordinates multiple Critics to review code changes.
type Orchestrator struct {
	diffProvider git.DiffProvider
	critics      []critic.Critic
}

// ReviewResult holds the aggregated output from all Critics.
type ReviewResult struct {
	Findings   []model.Finding `json:"findings"`
	Summary    ReviewSummary   `json:"summary"`
	BaseBranch string          `json:"base_branch"`
	TotalFiles int             `json:"total_files"`
	Errors     []CriticError   `json:"errors,omitempty"`
}

// ReviewSummary holds counts of findings by severity.
type ReviewSummary struct {
	CriticalCount int `json:"critical_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
}

// CriticError records an error from a single Critic without stopping others.
type CriticError struct {
	CriticName string `json:"critic_name"`
	Err        error  `json:"error"`
}

// NewOrchestrator creates a new Orchestrator with the given DiffProvider and Critics.
func NewOrchestrator(dp git.DiffProvider, critics ...critic.Critic) *Orchestrator {
	return &Orchestrator{
		diffProvider: dp,
		critics:      critics,
	}
}

// Run executes all Critics against the diff for the given repo and base branch.
// Individual Critic errors are recorded in ReviewResult.Errors without stopping
// other Critics. Returns an error only if the diff cannot be obtained.
func (o *Orchestrator) Run(ctx context.Context, repoPath, baseBranch string) (*ReviewResult, error) {
	diff, err := o.diffProvider.GetDiff(repoPath, git.DiffOptions{BaseBranch: baseBranch})
	if err != nil {
		return nil, fmt.Errorf("get diff: %w", err)
	}

	changedFiles := git.ExtractChangedFiles(diff)
	classifiedFiles := git.ClassifyChangedFiles(changedFiles)

	input := critic.ReviewInput{
		Diff:            diff,
		ClassifiedFiles: classifiedFiles,
		RepoPath:        repoPath,
	}

	var allFindings []model.Finding
	var criticErrors []CriticError

	for _, c := range o.critics {
		if ctx.Err() != nil {
			break
		}
		findings, reviewErr := c.Review(ctx, input)
		if reviewErr != nil {
			criticErrors = append(criticErrors, CriticError{
				CriticName: c.Name(),
				Err:        reviewErr,
			})
			continue
		}
		allFindings = append(allFindings, findings...)
	}

	sorted := sortFindings(allFindings)

	return &ReviewResult{
		Findings:   sorted,
		Summary:    buildSummary(sorted),
		BaseBranch: diff.BaseBranch,
		TotalFiles: len(diff.Files),
		Errors:     criticErrors,
	}, nil
}

// buildSummary computes a ReviewSummary from a slice of Findings.
func buildSummary(findings []model.Finding) ReviewSummary {
	var s ReviewSummary
	for _, f := range findings {
		switch f.Severity {
		case model.SeverityCritical:
			s.CriticalCount++
		case model.SeverityWarning:
			s.WarningCount++
		case model.SeverityInfo:
			s.InfoCount++
		}
	}
	return s
}

// severityOrder returns a numeric rank for sorting (lower = higher priority).
func severityOrder(s model.Severity) int {
	switch s {
	case model.SeverityCritical:
		return 0
	case model.SeverityWarning:
		return 1
	case model.SeverityInfo:
		return 2
	default:
		return 3
	}
}

// sortFindings returns a copy sorted by severity (critical first), then category.
func sortFindings(findings []model.Finding) []model.Finding {
	if len(findings) == 0 {
		return nil
	}
	sorted := make([]model.Finding, len(findings))
	copy(sorted, findings)
	slices.SortStableFunc(sorted, func(a, b model.Finding) int {
		if d := severityOrder(a.Severity) - severityOrder(b.Severity); d != 0 {
			return d
		}
		return strings.Compare(string(a.Category), string(b.Category))
	})
	return sorted
}
