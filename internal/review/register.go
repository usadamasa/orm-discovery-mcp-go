// Package review は review_pr MCP ツールの登録とハンドラーを提供する。
// server.go の O'Reilly ドメインから分離するために独立パッケージとして構成。
package review

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/reviewer"
)

// Args represents the parameters for the review_pr tool.
type Args struct {
	RepoPath   string `json:"repo_path" jsonschema:"Absolute path to the git repository to review"`
	BaseBranch string `json:"base_branch,omitempty" jsonschema:"Base branch for diff comparison (default: main)"`
}

// Result represents the structured output for the review_pr tool.
type Result struct {
	Summary    Summary   `json:"summary"`
	Findings   []Finding `json:"findings"`
	BaseBranch string    `json:"base_branch"`
	TotalFiles int       `json:"total_files"`
	Errors     []string  `json:"errors,omitempty"`
}

// Summary holds counts of findings by severity.
type Summary struct {
	CriticalCount int `json:"critical_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
}

// Finding is a JSON-friendly representation of a Finding for MCP responses.
type Finding struct {
	ID         string         `json:"id"`
	Severity   string         `json:"severity"`
	Category   string         `json:"category"`
	Message    string         `json:"message"`
	CriticName string         `json:"critic_name"`
	FilePath   string         `json:"file_path,omitempty"`
	Suggestion string         `json:"suggestion,omitempty"`
	Confidence float64        `json:"confidence"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// RegisterTools は review_pr ツールを MCP サーバーに登録する。
func RegisterTools(s *mcp.Server) {
	tool := &mcp.Tool{
		Name: "review_pr",
		Description: `Review code changes in a local git repository against the base branch.

Runs multiple critics (MissingTest, InfraChange, LargeDiff) and returns structured findings sorted by severity.

Input: Absolute path to git repository. Optionally specify base branch (default: main).

Output: Summary counts, detailed findings with severity/category/suggestion, and any critic errors.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Review Pull Request",
			ReadOnlyHint:    true,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(false),
		},
	}
	mcp.AddTool(s, tool, handleReviewPR)
}

func handleReviewPR(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args Args,
) (*mcp.CallToolResult, *Result, error) {
	if args.RepoPath == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "repo_path is required"}},
			IsError: true,
		}, nil, nil
	}
	baseBranch := args.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	dp := git.NewGitDiffProvider()
	critics := []critic.Critic{
		critic.NewMissingTestCritic(),
		critic.NewInfraChangeCritic(),
		critic.NewLargeDiffCritic(),
	}
	orch := reviewer.NewOrchestrator(dp, critics...)

	result, err := orch.Run(ctx, args.RepoPath, baseBranch)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("review failed: %v", err)}},
			IsError: true,
		}, nil, nil
	}

	mcpFindings := make([]Finding, len(result.Findings))
	for i, f := range result.Findings {
		mcpFindings[i] = Finding{
			ID:         f.ID,
			Severity:   string(f.Severity),
			Category:   string(f.Category),
			Message:    f.Message,
			CriticName: f.CriticName,
			FilePath:   f.Location.FilePath,
			Suggestion: f.Suggestion,
			Confidence: f.Confidence,
			Metadata:   f.Metadata,
		}
	}

	errStrings := make([]string, 0, len(result.Errors))
	for _, ce := range result.Errors {
		errStrings = append(errStrings, fmt.Sprintf("%s: %v", ce.CriticName, ce.Err))
	}

	return nil, &Result{
		Summary: Summary{
			CriticalCount: result.Summary.CriticalCount,
			WarningCount:  result.Summary.WarningCount,
			InfoCount:     result.Summary.InfoCount,
		},
		Findings:   mcpFindings,
		BaseBranch: result.BaseBranch,
		TotalFiles: result.TotalFiles,
		Errors:     errStrings,
	}, nil
}

func ptrBool(b bool) *bool { return &b }
