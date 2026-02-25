package critic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
)

func TestReviewInput_ZeroValue(t *testing.T) {
	var input ReviewInput
	assert.Nil(t, input.Diff)
	assert.Nil(t, input.ClassifiedFiles)
	assert.Empty(t, input.RepoPath)
}

func TestReviewInput_WithFields(t *testing.T) {
	diff := &git.DiffResult{
		BaseBranch: "main",
	}
	files := []git.ClassifiedFile{
		{
			ChangedFile: git.ChangedFile{
				Path:   "foo.go",
				Status: git.FileStatusAdded,
			},
			Category: git.FileCategoryCode,
		},
	}

	input := ReviewInput{
		Diff:            diff,
		ClassifiedFiles: files,
		RepoPath:        "/tmp/repo",
	}

	assert.Equal(t, diff, input.Diff)
	assert.Len(t, input.ClassifiedFiles, 1)
	assert.Equal(t, "/tmp/repo", input.RepoPath)
}
