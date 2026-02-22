# B4-1: review_pr MCP Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 既存 3 Critic を統合するオーケストレータと review_pr MCP ツールを実装し、PR レビューの一連フローを1つのツールとして提供する。

**Architecture:** `internal/reviewer/orchestrator.go` に Orchestrator を配置。DiffProvider で diff 取得→ファイル分類→各 Critic 逐次実行→Finding 集約・ソート→MCP レスポンス返却。

**Tech Stack:** Go 1.24, go-sdk/mcp, testify, google/uuid

**Design doc:** `docs/plans/2026-02-22-b4-1-review-pr-design.md`

---

## Task 1: Orchestrator 型定義とコンストラクタ

**Files:**
- Create: `internal/reviewer/orchestrator.go`
- Test: `internal/reviewer/orchestrator_test.go`

**Step 1: Write the failing test**

```go
// internal/reviewer/orchestrator_test.go
package reviewer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
)

// mockDiffProvider implements git.DiffProvider for testing.
type mockDiffProvider struct {
	result *git.DiffResult
	err    error
}

func (m *mockDiffProvider) GetDiff(_ string, _ git.DiffOptions) (*git.DiffResult, error) {
	return m.result, m.err
}

func TestNewOrchestrator(t *testing.T) {
	dp := &mockDiffProvider{}
	o := NewOrchestrator(dp)

	require.NotNil(t, o)
	assert.Empty(t, o.critics)
}

func TestNewOrchestrator_WithCritics(t *testing.T) {
	dp := &mockDiffProvider{}
	c1 := critic.NewMissingTestCritic()
	c2 := critic.NewInfraChangeCritic()
	c3 := critic.NewLargeDiffCritic()

	o := NewOrchestrator(dp, c1, c2, c3)

	require.NotNil(t, o)
	assert.Len(t, o.critics, 3)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/reviewer/ -run TestNewOrchestrator -v`
Expected: FAIL - package does not exist

**Step 3: Write minimal implementation**

```go
// internal/reviewer/orchestrator.go
package reviewer

import (
	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
)

// Orchestrator coordinates multiple Critics to review code changes.
type Orchestrator struct {
	diffProvider git.DiffProvider
	critics      []critic.Critic
}

// NewOrchestrator creates a new Orchestrator with the given DiffProvider and Critics.
func NewOrchestrator(dp git.DiffProvider, critics ...critic.Critic) *Orchestrator {
	return &Orchestrator{
		diffProvider: dp,
		critics:      critics,
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/reviewer/ -run TestNewOrchestrator -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/reviewer/orchestrator.go internal/reviewer/orchestrator_test.go
git commit -m "feat(reviewer): add Orchestrator type and constructor"
```

---

## Task 2: ReviewResult, ReviewSummary, CriticError 型定義

**Files:**
- Modify: `internal/reviewer/orchestrator.go`
- Test: `internal/reviewer/orchestrator_test.go`

**Step 1: Write the failing test**

```go
func TestReviewSummary_Zero(t *testing.T) {
	s := buildSummary(nil)
	assert.Equal(t, 0, s.CriticalCount)
	assert.Equal(t, 0, s.WarningCount)
	assert.Equal(t, 0, s.InfoCount)
}

func TestReviewSummary_MixedFindings(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.SeverityWarning, model.CategoryMissingTest, "msg1", model.Location{}),
		model.NewFinding(model.SeverityCritical, model.CategoryBuildFailure, "msg2", model.Location{}),
		model.NewFinding(model.SeverityWarning, model.CategoryInfraChange, "msg3", model.Location{}),
		model.NewFinding(model.SeverityInfo, model.CategoryStyleIssue, "msg4", model.Location{}),
	}
	s := buildSummary(findings)
	assert.Equal(t, 1, s.CriticalCount)
	assert.Equal(t, 2, s.WarningCount)
	assert.Equal(t, 1, s.InfoCount)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/reviewer/ -run TestReviewSummary -v`
Expected: FAIL - buildSummary not defined

**Step 3: Write minimal implementation**

Add to `internal/reviewer/orchestrator.go`:

```go
import "github.com/usadamasa/orm-discovery-mcp-go/internal/model"

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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/reviewer/ -run TestReviewSummary -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/reviewer/orchestrator.go internal/reviewer/orchestrator_test.go
git commit -m "feat(reviewer): add ReviewResult, ReviewSummary, CriticError types"
```

---

## Task 3: sortFindings ヘルパー

**Files:**
- Modify: `internal/reviewer/orchestrator.go`
- Test: `internal/reviewer/orchestrator_test.go`

**Step 1: Write the failing test**

```go
func TestSortFindings(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.SeverityInfo, model.CategoryStyleIssue, "info", model.Location{}),
		model.NewFinding(model.SeverityWarning, model.CategoryMissingTest, "warn1", model.Location{}),
		model.NewFinding(model.SeverityCritical, model.CategoryBuildFailure, "crit", model.Location{}),
		model.NewFinding(model.SeverityWarning, model.CategoryInfraChange, "warn2", model.Location{}),
	}

	sorted := sortFindings(findings)

	require.Len(t, sorted, 4)
	assert.Equal(t, model.SeverityCritical, sorted[0].Severity)
	assert.Equal(t, model.SeverityWarning, sorted[1].Severity)
	assert.Equal(t, model.SeverityWarning, sorted[2].Severity)
	assert.Equal(t, model.SeverityInfo, sorted[3].Severity)
}

func TestSortFindings_Empty(t *testing.T) {
	sorted := sortFindings(nil)
	assert.Empty(t, sorted)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/reviewer/ -run TestSortFindings -v`
Expected: FAIL - sortFindings not defined

**Step 3: Write minimal implementation**

```go
import "slices"

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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/reviewer/ -run TestSortFindings -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/reviewer/orchestrator.go internal/reviewer/orchestrator_test.go
git commit -m "feat(reviewer): add sortFindings helper"
```

---

## Task 4: Orchestrator.Run() メソッド

**Files:**
- Modify: `internal/reviewer/orchestrator.go`
- Test: `internal/reviewer/orchestrator_test.go`

**Step 1: Write the failing tests**

```go
func TestOrchestrator_Run_EmptyDiff(t *testing.T) {
	dp := &mockDiffProvider{
		result: &git.DiffResult{
			Files:      nil,
			BaseBranch: "main",
		},
	}
	o := NewOrchestrator(dp, critic.NewMissingTestCritic())

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.Equal(t, "main", result.BaseBranch)
	assert.Equal(t, 0, result.TotalFiles)
}

func TestOrchestrator_Run_DiffProviderError(t *testing.T) {
	dp := &mockDiffProvider{
		err: fmt.Errorf("git not found"),
	}
	o := NewOrchestrator(dp, critic.NewMissingTestCritic())

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "git not found")
}

func TestOrchestrator_Run_WithFindings(t *testing.T) {
	dp := &mockDiffProvider{
		result: &git.DiffResult{
			Files: []git.FileDiff{
				{Path: "internal/handler.go", Status: "added", Additions: 50},
				{Path: "Dockerfile", Status: "modified", Additions: 5, Deletions: 2},
			},
			TotalAdditions: 55,
			TotalDeletions: 2,
			BaseBranch:     "main",
		},
	}
	o := NewOrchestrator(dp,
		critic.NewMissingTestCritic(),
		critic.NewInfraChangeCritic(),
		critic.NewLargeDiffCritic(),
	)

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "main", result.BaseBranch)
	assert.Equal(t, 2, result.TotalFiles)
	// handler.go → MissingTestCritic finding, Dockerfile → InfraChangeCritic finding
	assert.GreaterOrEqual(t, len(result.Findings), 2)
	assert.Empty(t, result.Errors)
}

func TestOrchestrator_Run_CriticErrorContinues(t *testing.T) {
	dp := &mockDiffProvider{
		result: &git.DiffResult{
			Files:      []git.FileDiff{{Path: "Dockerfile", Status: "added", Additions: 1}},
			BaseBranch: "main",
		},
	}
	failCritic := &failingCritic{name: "FailCritic", err: fmt.Errorf("critic broke")}
	o := NewOrchestrator(dp, failCritic, critic.NewInfraChangeCritic())

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.NoError(t, err)
	require.NotNil(t, result)
	// InfraChangeCritic should still produce findings despite FailCritic error
	assert.NotEmpty(t, result.Findings)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "FailCritic", result.Errors[0].CriticName)
}

func TestOrchestrator_Run_FindingsSortedBySeverity(t *testing.T) {
	dp := &mockDiffProvider{
		result: &git.DiffResult{
			Files: []git.FileDiff{
				{Path: "internal/handler.go", Status: "added", Additions: 50},
				{Path: "Dockerfile", Status: "modified", Additions: 5},
			},
			TotalAdditions: 600, // exceeds LargeDiffCritic threshold
			TotalDeletions: 0,
			BaseBranch:     "main",
		},
	}
	o := NewOrchestrator(dp,
		critic.NewMissingTestCritic(),
		critic.NewInfraChangeCritic(),
		critic.NewLargeDiffCritic(),
	)

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.NoError(t, err)
	// Verify sorted: all findings should be in severity order
	for i := 1; i < len(result.Findings); i++ {
		prev := severityOrder(result.Findings[i-1].Severity)
		curr := severityOrder(result.Findings[i].Severity)
		assert.LessOrEqual(t, prev, curr, "findings should be sorted by severity")
	}
}
```

Also add `failingCritic` test helper:

```go
type failingCritic struct {
	name string
	err  error
}

func (f *failingCritic) Name() string { return f.name }
func (f *failingCritic) Review(_ context.Context, _ critic.ReviewInput) ([]model.Finding, error) {
	return nil, f.err
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/reviewer/ -run TestOrchestrator_Run -v`
Expected: FAIL - Run method not defined

**Step 3: Write minimal implementation**

```go
import (
	"context"
	"fmt"
)

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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/reviewer/ -run TestOrchestrator_Run -v`
Expected: PASS

**Step 5: Run full package tests**

Run: `go test ./internal/reviewer/ -v`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/reviewer/orchestrator.go internal/reviewer/orchestrator_test.go
git commit -m "feat(reviewer): implement Orchestrator.Run() with critic orchestration"
```

---

## Task 5: MCP ツール引数・結果型の定義

**Files:**
- Modify: `tools_args.go`

**Step 1: Add ReviewPR types to tools_args.go**

```go
// ReviewPRArgs represents the parameters for the review_pr tool.
type ReviewPRArgs struct {
	RepoPath   string `json:"repo_path" jsonschema:"Absolute path to the git repository to review"`
	BaseBranch string `json:"base_branch,omitempty" jsonschema:"Base branch for diff comparison (default: main)"`
}

// ReviewPRResult represents the structured output for the review_pr tool.
type ReviewPRResult struct {
	Summary    ReviewPRSummary    `json:"summary"`
	Findings   []ReviewPRFinding  `json:"findings"`
	BaseBranch string             `json:"base_branch"`
	TotalFiles int                `json:"total_files"`
	Errors     []string           `json:"errors,omitempty"`
}

// ReviewPRSummary holds counts of findings by severity.
type ReviewPRSummary struct {
	CriticalCount int `json:"critical_count"`
	WarningCount  int `json:"warning_count"`
	InfoCount     int `json:"info_count"`
}

// ReviewPRFinding is a JSON-friendly representation of a Finding.
type ReviewPRFinding struct {
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
```

**Step 2: Verify build**

Run: `go build ./...`
Expected: Success

**Step 3: Commit**

```bash
git add tools_args.go
git commit -m "feat: add ReviewPR args and result types"
```

---

## Task 6: ReviewPRHandler と review_pr ツール登録

**Files:**
- Modify: `server.go` (registerHandlers + handler method)

**Step 1: Add review_pr tool to registerHandlers()**

In `server.go`, add after the `oreilly_ask_question` tool registration:

```go
// Add review_pr tool
reviewPRTool := &mcp.Tool{
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
mcp.AddTool(s.server, reviewPRTool, s.ReviewPRHandler)
```

**Step 2: Implement ReviewPRHandler**

```go
// ReviewPRHandler handles the review_pr MCP tool.
func (s *Server) ReviewPRHandler(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args ReviewPRArgs,
) (*mcp.CallToolResult, *ReviewPRResult, error) {
	if args.RepoPath == "" {
		return newToolResultError("repo_path is required"), nil, nil
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
		return newToolResultError(fmt.Sprintf("review failed: %v", err)), nil, nil
	}

	// Convert to MCP result
	mcpFindings := make([]ReviewPRFinding, len(result.Findings))
	for i, f := range result.Findings {
		mcpFindings[i] = ReviewPRFinding{
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

	var errStrings []string
	for _, ce := range result.Errors {
		errStrings = append(errStrings, fmt.Sprintf("%s: %v", ce.CriticName, ce.Err))
	}

	return nil, &ReviewPRResult{
		Summary: ReviewPRSummary{
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
```

**Step 3: Add imports to server.go**

```go
import (
	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/reviewer"
)
```

**Step 4: Verify build**

Run: `go build ./...`
Expected: Success

**Step 5: Commit**

```bash
git add server.go
git commit -m "feat: register review_pr MCP tool with handler"
```

---

## Task 7: CI 検証と最終確認

**Files:**
- None (verification only)

**Step 1: Run full CI**

Run: `task ci`
Expected: ALL PASS (format, lint, test, build)

**Step 2: Fix any issues**

If lint or test fails, fix and re-run until `task ci` passes.

**Step 3: Squash and commit if fixes were needed**

```bash
# Only if fixup commits were needed
git add -A
git commit -m "fix: address CI issues in review_pr implementation"
```

---

## Task 8: PR 作成

**Files:**
- None (git/GitHub operations only)

**Step 1: Create feature branch and push**

Note: If working in a worktree, the branch may already exist. Otherwise:

```bash
git checkout -b feature/b4-1-review-pr-tool
git push -u origin feature/b4-1-review-pr-tool
```

**Step 2: Create PR**

```bash
gh pr create --title "feat: implement review_pr MCP tool" --body "$(cat <<'EOF'
## Summary
- Implement `internal/reviewer/orchestrator.go` with Orchestrator type
- Register `review_pr` MCP tool in server.go
- Integrate 3 Critics: MissingTestCritic, InfraChangeCritic, LargeDiffCritic
- Finding aggregation with severity-based sorting
- Individual Critic error isolation (one failure doesn't stop others)

Closes #101

## Test plan
- [ ] `task ci` passes
- [ ] Orchestrator unit tests (empty diff, error handling, finding aggregation, sorting)
- [ ] MCP tool builds and registers correctly

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

**Step 3: Finalize PR**

Run `/finalize-pr` to review and merge the PR.
