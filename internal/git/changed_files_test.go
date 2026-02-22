package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractChangedFiles(t *testing.T) {
	result := &DiffResult{
		BaseBranch: "main",
		Files: []FileDiff{
			{
				Path:      "newfile.go",
				Status:    "added",
				Additions: 10,
				Deletions: 0,
				IsBinary:  false,
				Patch:     "diff --git a/newfile.go b/newfile.go\n...",
			},
			{
				Path:      "existing.go",
				Status:    "modified",
				Additions: 5,
				Deletions: 3,
				IsBinary:  false,
				Patch:     "diff --git a/existing.go b/existing.go\n...",
			},
			{
				Path:      "removed.go",
				Status:    "deleted",
				Additions: 0,
				Deletions: 15,
				IsBinary:  false,
				Patch:     "diff --git a/removed.go b/removed.go\n...",
			},
			{
				Path:      "new_name.go",
				OldPath:   "old_name.go",
				Status:    "renamed",
				Additions: 2,
				Deletions: 1,
				IsBinary:  false,
				Patch:     "diff --git a/old_name.go b/new_name.go\n...",
			},
		},
		TotalAdditions: 17,
		TotalDeletions: 19,
	}

	files := ExtractChangedFiles(result)

	assert.Len(t, files, 4)

	// added
	assert.Equal(t, "newfile.go", files[0].Path)
	assert.Equal(t, FileStatusAdded, files[0].Status)
	assert.Equal(t, 10, files[0].Additions)
	assert.Equal(t, 0, files[0].Deletions)
	assert.False(t, files[0].IsBinary)

	// modified
	assert.Equal(t, "existing.go", files[1].Path)
	assert.Equal(t, FileStatusModified, files[1].Status)
	assert.Equal(t, 5, files[1].Additions)
	assert.Equal(t, 3, files[1].Deletions)

	// deleted
	assert.Equal(t, "removed.go", files[2].Path)
	assert.Equal(t, FileStatusDeleted, files[2].Status)
	assert.Equal(t, 0, files[2].Additions)
	assert.Equal(t, 15, files[2].Deletions)

	// renamed
	assert.Equal(t, "new_name.go", files[3].Path)
	assert.Equal(t, "old_name.go", files[3].OldPath)
	assert.Equal(t, FileStatusRenamed, files[3].Status)
	assert.Equal(t, 2, files[3].Additions)
	assert.Equal(t, 1, files[3].Deletions)
}

func TestExtractChangedFiles_EmptyResult(t *testing.T) {
	result := &DiffResult{
		BaseBranch: "main",
		Files:      nil,
	}

	files := ExtractChangedFiles(result)

	assert.Empty(t, files)
}

func TestExtractChangedFiles_PatchOmitted(t *testing.T) {
	result := &DiffResult{
		Files: []FileDiff{
			{
				Path:      "file.go",
				Status:    "modified",
				Additions: 3,
				Deletions: 1,
				Patch:     "diff --git a/file.go b/file.go\n+added line\n-removed line",
			},
		},
	}

	files := ExtractChangedFiles(result)

	assert.Len(t, files, 1)
	// ChangedFile does not have a Patch field; this test verifies the struct
	// only carries the lightweight fields from FileDiff.
	assert.Equal(t, "file.go", files[0].Path)
	assert.Equal(t, FileStatusModified, files[0].Status)
	assert.Equal(t, 3, files[0].Additions)
	assert.Equal(t, 1, files[0].Deletions)
}

func TestExtractChangedFiles_BinaryFile(t *testing.T) {
	result := &DiffResult{
		Files: []FileDiff{
			{
				Path:     "image.png",
				Status:   "added",
				IsBinary: true,
			},
		},
	}

	files := ExtractChangedFiles(result)

	assert.Len(t, files, 1)
	assert.Equal(t, "image.png", files[0].Path)
	assert.Equal(t, FileStatusAdded, files[0].Status)
	assert.True(t, files[0].IsBinary)
	assert.Equal(t, 0, files[0].Additions)
	assert.Equal(t, 0, files[0].Deletions)
}

func TestFileStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status FileStatus
		want   bool
	}{
		{"added", FileStatusAdded, true},
		{"modified", FileStatusModified, true},
		{"deleted", FileStatusDeleted, true},
		{"renamed", FileStatusRenamed, true},
		{"empty string", FileStatus(""), false},
		{"unknown", FileStatus("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestFileStatus_StringValues(t *testing.T) {
	assert.Equal(t, "added", string(FileStatusAdded))
	assert.Equal(t, "modified", string(FileStatusModified))
	assert.Equal(t, "deleted", string(FileStatusDeleted))
	assert.Equal(t, "renamed", string(FileStatusRenamed))
}
