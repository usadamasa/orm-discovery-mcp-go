package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository with an initial commit on main.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test User",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test User",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	runGit("init", "-b", "main")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")
	runGit("commit", "--allow-empty", "-m", "initial commit")

	return dir
}

// runGitInRepo is a test helper to execute git commands in a repo directory.
func runGitInRepo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, out)
}

func TestGetDiff_AddedFile(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a feature branch from main
	runGitInRepo(t, dir, "checkout", "-b", "feature")

	// Add a new file
	err := os.WriteFile(filepath.Join(dir, "newfile.txt"), []byte("hello\nworld\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "newfile.txt")
	runGitInRepo(t, dir, "commit", "-m", "add newfile")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "main"})
	require.NoError(t, err)

	require.Len(t, result.Files, 1)
	assert.Equal(t, "newfile.txt", result.Files[0].Path)
	assert.Equal(t, "added", result.Files[0].Status)
	assert.Equal(t, 2, result.Files[0].Additions)
	assert.Equal(t, 0, result.Files[0].Deletions)
	assert.False(t, result.Files[0].IsBinary)
	assert.NotEmpty(t, result.Files[0].Patch)
	assert.Equal(t, 2, result.TotalAdditions)
	assert.Equal(t, 0, result.TotalDeletions)
	assert.Equal(t, "main", result.BaseBranch)
}

func TestGetDiff_ModifiedFile(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a file on main
	err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("line1\nline2\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "existing.txt")
	runGitInRepo(t, dir, "commit", "-m", "add existing file")

	// Create feature branch and modify
	runGitInRepo(t, dir, "checkout", "-b", "feature")
	err = os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("line1\nmodified\nline3\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "existing.txt")
	runGitInRepo(t, dir, "commit", "-m", "modify existing file")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "main"})
	require.NoError(t, err)

	require.Len(t, result.Files, 1)
	assert.Equal(t, "existing.txt", result.Files[0].Path)
	assert.Equal(t, "modified", result.Files[0].Status)
	assert.Greater(t, result.Files[0].Additions, 0)
	assert.Greater(t, result.Files[0].Deletions, 0)
	assert.NotEmpty(t, result.Files[0].Patch)
}

func TestGetDiff_DeletedFile(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a file on main
	err := os.WriteFile(filepath.Join(dir, "toremove.txt"), []byte("content\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "toremove.txt")
	runGitInRepo(t, dir, "commit", "-m", "add file to remove")

	// Create feature branch and delete
	runGitInRepo(t, dir, "checkout", "-b", "feature")
	err = os.Remove(filepath.Join(dir, "toremove.txt"))
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "toremove.txt")
	runGitInRepo(t, dir, "commit", "-m", "delete file")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "main"})
	require.NoError(t, err)

	require.Len(t, result.Files, 1)
	assert.Equal(t, "toremove.txt", result.Files[0].Path)
	assert.Equal(t, "deleted", result.Files[0].Status)
	assert.Equal(t, 0, result.Files[0].Additions)
	assert.Equal(t, 1, result.Files[0].Deletions)
}

func TestGetDiff_StagedChanges(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a file and stage it (but do not commit)
	err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged content\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "staged.txt")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{Staged: true})
	require.NoError(t, err)

	require.Len(t, result.Files, 1)
	assert.Equal(t, "staged.txt", result.Files[0].Path)
	assert.Equal(t, "added", result.Files[0].Status)
	assert.Equal(t, 1, result.Files[0].Additions)
	assert.NotEmpty(t, result.Files[0].Patch)
}

func TestGetDiff_DefaultBaseBranch(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a feature branch from main
	runGitInRepo(t, dir, "checkout", "-b", "feature")

	err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "file.txt")
	runGitInRepo(t, dir, "commit", "-m", "add file")

	provider := NewGitDiffProvider()
	// Empty BaseBranch should default to "main"
	result, err := provider.GetDiff(dir, DiffOptions{})
	require.NoError(t, err)

	assert.Equal(t, "main", result.BaseBranch)
	require.Len(t, result.Files, 1)
	assert.Equal(t, "file.txt", result.Files[0].Path)
}

func TestGetDiff_CustomBaseBranch(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a develop branch
	runGitInRepo(t, dir, "checkout", "-b", "develop")
	err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "base.txt")
	runGitInRepo(t, dir, "commit", "-m", "add base file")

	// Create feature branch from develop
	runGitInRepo(t, dir, "checkout", "-b", "feature")
	err = os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "feature.txt")
	runGitInRepo(t, dir, "commit", "-m", "add feature file")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "develop"})
	require.NoError(t, err)

	assert.Equal(t, "develop", result.BaseBranch)
	require.Len(t, result.Files, 1)
	assert.Equal(t, "feature.txt", result.Files[0].Path)
}

func TestParseDiffStat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]fileStat
	}{
		{
			name:  "single file",
			input: "10\t2\tpath/to/file.go\n",
			expected: map[string]fileStat{
				"path/to/file.go": {additions: 10, deletions: 2, isBinary: false},
			},
		},
		{
			name:  "multiple files",
			input: "5\t3\tfile1.go\n20\t0\tfile2.go\n",
			expected: map[string]fileStat{
				"file1.go": {additions: 5, deletions: 3, isBinary: false},
				"file2.go": {additions: 20, deletions: 0, isBinary: false},
			},
		},
		{
			name:  "binary file",
			input: "-\t-\timage.png\n",
			expected: map[string]fileStat{
				"image.png": {additions: 0, deletions: 0, isBinary: true},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: map[string]fileStat{},
		},
		{
			name:  "renamed file with arrow",
			input: "3\t1\toldname.go => newname.go\n",
			expected: map[string]fileStat{
				"oldname.go => newname.go": {additions: 3, deletions: 1, isBinary: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDiffStat(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
			for path, expectedStat := range tt.expected {
				actualStat, ok := result[path]
				require.True(t, ok, "expected path %q not found in result", path)
				assert.Equal(t, expectedStat.additions, actualStat.additions)
				assert.Equal(t, expectedStat.deletions, actualStat.deletions)
				assert.Equal(t, expectedStat.isBinary, actualStat.isBinary)
			}
		})
	}
}

func TestParseDiffNameStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []nameStatusEntry
	}{
		{
			name:  "added file",
			input: "A\tnewfile.go\n",
			expected: []nameStatusEntry{
				{status: "added", path: "newfile.go"},
			},
		},
		{
			name:  "modified file",
			input: "M\texisting.go\n",
			expected: []nameStatusEntry{
				{status: "modified", path: "existing.go"},
			},
		},
		{
			name:  "deleted file",
			input: "D\tremoved.go\n",
			expected: []nameStatusEntry{
				{status: "deleted", path: "removed.go"},
			},
		},
		{
			name:  "renamed file",
			input: "R100\told.go\tnew.go\n",
			expected: []nameStatusEntry{
				{status: "renamed", path: "new.go", oldPath: "old.go"},
			},
		},
		{
			name:  "multiple entries",
			input: "A\tfile1.go\nM\tfile2.go\nD\tfile3.go\n",
			expected: []nameStatusEntry{
				{status: "added", path: "file1.go"},
				{status: "modified", path: "file2.go"},
				{status: "deleted", path: "file3.go"},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDiffNameStatus(tt.input)
			assert.Equal(t, len(tt.expected), len(result))
			for i, expected := range tt.expected {
				assert.Equal(t, expected.status, result[i].status)
				assert.Equal(t, expected.path, result[i].path)
				assert.Equal(t, expected.oldPath, result[i].oldPath)
			}
		})
	}
}

func TestGetDiff_InvalidRepoPath(t *testing.T) {
	provider := NewGitDiffProvider()
	_, err := provider.GetDiff("/nonexistent/path", DiffOptions{})
	assert.Error(t, err)
}

func TestGetDiff_ContextLines(t *testing.T) {
	dir := setupTestRepo(t)

	// Create a file on main with multiple lines
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\n"
	err := os.WriteFile(filepath.Join(dir, "multiline.txt"), []byte(content), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "multiline.txt")
	runGitInRepo(t, dir, "commit", "-m", "add multiline file")

	// Modify one line in the middle
	runGitInRepo(t, dir, "checkout", "-b", "feature")
	modified := "line1\nline2\nline3\nline4\nMODIFIED\nline6\nline7\nline8\nline9\nline10\n"
	err = os.WriteFile(filepath.Join(dir, "multiline.txt"), []byte(modified), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", "multiline.txt")
	runGitInRepo(t, dir, "commit", "-m", "modify middle line")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "main", Context: 1})
	require.NoError(t, err)

	require.Len(t, result.Files, 1)
	assert.NotEmpty(t, result.Files[0].Patch)
	// With context=1, the patch should have fewer context lines than default
	assert.Contains(t, result.Files[0].Patch, "MODIFIED")
}

func TestGetDiff_MultipleFiles(t *testing.T) {
	dir := setupTestRepo(t)

	runGitInRepo(t, dir, "checkout", "-b", "feature")

	// Add multiple files
	err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("content1\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("content2\nline2\n"), 0644)
	require.NoError(t, err)
	runGitInRepo(t, dir, "add", ".")
	runGitInRepo(t, dir, "commit", "-m", "add multiple files")

	provider := NewGitDiffProvider()
	result, err := provider.GetDiff(dir, DiffOptions{BaseBranch: "main"})
	require.NoError(t, err)

	assert.Len(t, result.Files, 2)
	assert.Equal(t, 3, result.TotalAdditions) // 1 + 2
	assert.Equal(t, 0, result.TotalDeletions)

	// Each file should have its own patch
	for _, f := range result.Files {
		assert.NotEmpty(t, f.Patch, "file %s should have a patch", f.Path)
	}
}
