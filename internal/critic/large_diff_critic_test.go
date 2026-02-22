package critic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// Compile-time check that LargeDiffCritic implements Critic.
var _ Critic = (*LargeDiffCritic)(nil)

func TestLargeDiffCritic_Name(t *testing.T) {
	c := NewLargeDiffCritic()
	assert.Equal(t, "LargeDiffCritic", c.Name())
}

func TestLargeDiffCritic_Defaults(t *testing.T) {
	c := NewLargeDiffCritic()
	assert.Equal(t, 500, c.MaxChangedLines)
	assert.Equal(t, 20, c.MaxChangedFiles)
}

func TestLargeDiffCritic_Review(t *testing.T) {
	tests := []struct {
		name         string
		diff         *git.DiffResult
		wantFindings int
	}{
		{
			name: "lines exceed threshold",
			diff: &git.DiffResult{
				TotalAdditions: 400,
				TotalDeletions: 200,
				Files:          make([]git.FileDiff, 5),
			},
			wantFindings: 1,
		},
		{
			name: "files exceed threshold",
			diff: &git.DiffResult{
				TotalAdditions: 100,
				TotalDeletions: 50,
				Files:          make([]git.FileDiff, 25),
			},
			wantFindings: 1,
		},
		{
			name: "both lines and files exceed threshold",
			diff: &git.DiffResult{
				TotalAdditions: 400,
				TotalDeletions: 200,
				Files:          make([]git.FileDiff, 25),
			},
			wantFindings: 2,
		},
		{
			name: "lines exactly at threshold produces no finding",
			diff: &git.DiffResult{
				TotalAdditions: 300,
				TotalDeletions: 200,
				Files:          make([]git.FileDiff, 5),
			},
			wantFindings: 0,
		},
		{
			name: "files exactly at threshold produces no finding",
			diff: &git.DiffResult{
				TotalAdditions: 50,
				TotalDeletions: 50,
				Files:          make([]git.FileDiff, 20),
			},
			wantFindings: 0,
		},
		{
			name: "below threshold produces no finding",
			diff: &git.DiffResult{
				TotalAdditions: 50,
				TotalDeletions: 30,
				Files:          make([]git.FileDiff, 3),
			},
			wantFindings: 0,
		},
		{
			name:         "empty diff produces no finding",
			diff:         &git.DiffResult{},
			wantFindings: 0,
		},
		{
			name:         "nil diff produces no finding",
			diff:         nil,
			wantFindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewLargeDiffCritic()
			input := ReviewInput{
				Diff: tt.diff,
			}

			findings, err := c.Review(context.Background(), input)
			require.NoError(t, err)

			if tt.wantFindings == 0 {
				assert.Empty(t, findings)
				return
			}

			assert.Len(t, findings, tt.wantFindings)
			for _, f := range findings {
				assert.Equal(t, model.SeverityWarning, f.Severity)
				assert.Equal(t, model.CategoryLargeDiff, f.Category)
				assert.Equal(t, "LargeDiffCritic", f.CriticName)
				assert.NotEmpty(t, f.ID)
				assert.NotEmpty(t, f.Message)
				assert.NotEmpty(t, f.Suggestion)
				assert.NotZero(t, f.CreatedAt)
			}
		})
	}
}

func TestLargeDiffCritic_CustomThresholds(t *testing.T) {
	c := &LargeDiffCritic{
		MaxChangedLines: 100,
		MaxChangedFiles: 5,
	}

	input := ReviewInput{
		Diff: &git.DiffResult{
			TotalAdditions: 80,
			TotalDeletions: 30,
			Files:          make([]git.FileDiff, 3),
		},
	}

	findings, err := c.Review(context.Background(), input)
	require.NoError(t, err)
	assert.Len(t, findings, 1, "custom threshold 100 should catch 110 lines")

	f := findings[0]
	assert.Contains(t, f.Message, "110")
	assert.Contains(t, f.Message, "100")
}

func TestLargeDiffCritic_FindingFields(t *testing.T) {
	c := NewLargeDiffCritic()
	input := ReviewInput{
		Diff: &git.DiffResult{
			TotalAdditions: 400,
			TotalDeletions: 200,
			Files:          make([]git.FileDiff, 25),
		},
	}

	findings, err := c.Review(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, findings, 2)

	// Check lines finding
	linesF := findings[0]
	assert.Equal(t, model.SeverityWarning, linesF.Severity)
	assert.Equal(t, model.CategoryLargeDiff, linesF.Category)
	assert.Equal(t, "LargeDiffCritic", linesF.CriticName)
	assert.Contains(t, linesF.Message, "600")
	assert.Contains(t, linesF.Message, "500")
	assert.NotEmpty(t, linesF.Suggestion)
	assert.Equal(t, 600, linesF.Metadata["actual"])
	assert.Equal(t, 500, linesF.Metadata["threshold"])

	// Check files finding
	filesF := findings[1]
	assert.Equal(t, model.SeverityWarning, filesF.Severity)
	assert.Equal(t, model.CategoryLargeDiff, filesF.Category)
	assert.Equal(t, "LargeDiffCritic", filesF.CriticName)
	assert.Contains(t, filesF.Message, "25")
	assert.Contains(t, filesF.Message, "20")
	assert.NotEmpty(t, filesF.Suggestion)
	assert.Equal(t, 25, filesF.Metadata["actual"])
	assert.Equal(t, 20, filesF.Metadata["threshold"])
}
