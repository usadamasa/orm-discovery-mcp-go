// Package critic provides interfaces and implementations for code review analysis.
// Each Critic examines code changes and generates structured findings.
package critic

import (
	"context"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// ReviewInput holds all data needed by a Critic to perform review.
// RepoPath is optional; if empty, critics cannot perform filesystem checks.
type ReviewInput struct {
	Diff            *git.DiffResult
	ClassifiedFiles []git.ClassifiedFile
	RepoPath        string
}

// Critic analyzes code changes and generates review findings.
// Name returns a unique, non-empty identifier for this critic.
// Review returns (nil, nil) for no findings, or (findings, nil) on success.
type Critic interface {
	Name() string
	Review(ctx context.Context, input ReviewInput) ([]model.Finding, error)
}
