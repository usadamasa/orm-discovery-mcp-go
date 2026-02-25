package reviewer

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/critic"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
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

func TestBuildSummary_Zero(t *testing.T) {
	s := buildSummary(nil)
	assert.Equal(t, 0, s.CriticalCount)
	assert.Equal(t, 0, s.WarningCount)
	assert.Equal(t, 0, s.InfoCount)
}

func TestBuildSummary_MixedFindings(t *testing.T) {
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

func TestSortFindings_SameSeveritySortedByCategory(t *testing.T) {
	findings := []model.Finding{
		model.NewFinding(model.SeverityWarning, model.CategoryMissingTest, "warn1", model.Location{}),
		model.NewFinding(model.SeverityWarning, model.CategoryInfraChange, "warn2", model.Location{}),
	}

	sorted := sortFindings(findings)

	require.Len(t, sorted, 2)
	// infra_change < missing_test alphabetically
	assert.Equal(t, model.CategoryInfraChange, sorted[0].Category)
	assert.Equal(t, model.CategoryMissingTest, sorted[1].Category)
}

func TestSortFindings_Empty(t *testing.T) {
	sorted := sortFindings(nil)
	assert.Empty(t, sorted)
}

// failingCritic is a test helper that always returns an error.
type failingCritic struct {
	name string
	err  error
}

func (f *failingCritic) Name() string { return f.name }
func (f *failingCritic) Review(_ context.Context, _ critic.ReviewInput) ([]model.Finding, error) {
	return nil, f.err
}

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
	// handler.go → MissingTestCritic, Dockerfile → InfraChangeCritic
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
	fc := &failingCritic{name: "FailCritic", err: fmt.Errorf("critic broke")}
	o := NewOrchestrator(dp, fc, critic.NewInfraChangeCritic())

	result, err := o.Run(context.Background(), "/tmp/repo", "main")

	require.NoError(t, err)
	require.NotNil(t, result)
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
	for i := 1; i < len(result.Findings); i++ {
		prev := severityOrder(result.Findings[i-1].Severity)
		curr := severityOrder(result.Findings[i].Severity)
		assert.LessOrEqual(t, prev, curr, "findings should be sorted by severity")
	}
}
