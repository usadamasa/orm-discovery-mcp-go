package critic

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// Compile-time check that MissingTestCritic implements Critic.
var _ Critic = (*MissingTestCritic)(nil)

func TestMissingTestCritic_Name(t *testing.T) {
	c := NewMissingTestCritic()
	assert.Equal(t, "MissingTestCritic", c.Name())
}

func TestMissingTestCritic_Review(t *testing.T) {
	tests := []struct {
		name          string
		setupDisk     func(t *testing.T, repoPath string) // create test files on disk
		classified    []git.ClassifiedFile
		wantFindings  int
		wantPaths     []string // expected file paths in findings
		wantNoFinding bool
	}{
		{
			name: "new code file without test generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/foo.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"internal/foo.go"},
		},
		{
			name: "new code file with test in diff generates no finding",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/foo.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/foo_test.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryTest,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "new code file with test on disk generates no finding",
			setupDisk: func(t *testing.T, repoPath string) {
				t.Helper()
				dir := filepath.Join(repoPath, "internal")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "foo_test.go"), []byte("package internal"), 0o644))
			},
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/foo.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "modified code file without test generates warning",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "pkg/bar.go",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantFindings: 1,
			wantPaths:    []string{"pkg/bar.go"},
		},
		{
			name: "modified code file with test in diff generates no finding",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "pkg/bar.go",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryCode,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "pkg/bar_test.go",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryTest,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "modified code file with test on disk generates no finding",
			setupDisk: func(t *testing.T, repoPath string) {
				t.Helper()
				dir := filepath.Join(repoPath, "pkg")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "bar_test.go"), []byte("package pkg"), 0o644))
			},
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "pkg/bar.go",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "deleted file is skipped",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/old.go",
						Status: git.FileStatusDeleted,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "test-only change is skipped",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "internal/foo_test.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryTest,
				},
			},
			wantNoFinding: true,
		},
		{
			name: "infra file is skipped",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "Dockerfile",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryInfra,
				},
			},
			wantNoFinding: true,
		},
		{
			name:          "empty classified files produces no findings",
			classified:    nil,
			wantNoFinding: true,
		},
		{
			name: "multiple code files with partial test coverage",
			classified: []git.ClassifiedFile{
				{
					ChangedFile: git.ChangedFile{
						Path:   "a.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "a_test.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryTest,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "b.go",
						Status: git.FileStatusAdded,
					},
					Category: git.FileCategoryCode,
				},
				{
					ChangedFile: git.ChangedFile{
						Path:   "c.go",
						Status: git.FileStatusModified,
					},
					Category: git.FileCategoryCode,
				},
			},
			wantFindings: 2,
			wantPaths:    []string{"b.go", "c.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := t.TempDir()
			if tt.setupDisk != nil {
				tt.setupDisk(t, repoPath)
			}

			c := NewMissingTestCritic()
			input := ReviewInput{
				Diff:            &git.DiffResult{},
				ClassifiedFiles: tt.classified,
				RepoPath:        repoPath,
			}

			findings, err := c.Review(context.Background(), input)
			require.NoError(t, err)

			if tt.wantNoFinding {
				assert.Empty(t, findings, "expected no findings")
				return
			}

			assert.Len(t, findings, tt.wantFindings)
			for i, f := range findings {
				assert.Equal(t, model.SeverityWarning, f.Severity)
				assert.Equal(t, model.CategoryMissingTest, f.Category)
				assert.Equal(t, "MissingTestCritic", f.CriticName)
				assert.NotEmpty(t, f.ID)
				assert.NotEmpty(t, f.Message)
				assert.NotZero(t, f.CreatedAt)

				if i < len(tt.wantPaths) {
					assert.Equal(t, tt.wantPaths[i], f.Location.FilePath)
				}
			}
		})
	}
}

func TestMissingTestCritic_FindingFields(t *testing.T) {
	repoPath := t.TempDir()
	c := NewMissingTestCritic()
	input := ReviewInput{
		Diff: &git.DiffResult{},
		ClassifiedFiles: []git.ClassifiedFile{
			{
				ChangedFile: git.ChangedFile{
					Path:   "internal/handler.go",
					Status: git.FileStatusAdded,
				},
				Category: git.FileCategoryCode,
			},
		},
		RepoPath: repoPath,
	}

	findings, err := c.Review(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, findings, 1)

	f := findings[0]
	assert.Equal(t, model.SeverityWarning, f.Severity)
	assert.Equal(t, model.CategoryMissingTest, f.Category)
	assert.Equal(t, "MissingTestCritic", f.CriticName)
	assert.Equal(t, "internal/handler.go", f.Location.FilePath)
	assert.Contains(t, f.Message, "handler.go")
	assert.Contains(t, f.Suggestion, "handler_test.go")
	assert.True(t, f.Confidence > 0)
	assert.True(t, f.Confidence <= 1.0)
}

func TestDeriveTestPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"internal/foo.go", "internal/foo_test.go"},
		{"pkg/bar.py", "pkg/bar_test.py"},
		{"lib/baz.ts", "lib/baz_test.ts"},
		{"deep/nested/path/file.go", "deep/nested/path/file_test.go"},
		{"main.go", "main_test.go"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := deriveTestPath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMissingTestCritic_EmptyRepoPath(t *testing.T) {
	c := NewMissingTestCritic()
	input := ReviewInput{
		Diff: &git.DiffResult{},
		ClassifiedFiles: []git.ClassifiedFile{
			{
				ChangedFile: git.ChangedFile{
					Path:   "internal/foo.go",
					Status: git.FileStatusAdded,
				},
				Category: git.FileCategoryCode,
			},
		},
		RepoPath: "", // empty: skip disk check
	}

	findings, err := c.Review(context.Background(), input)
	require.NoError(t, err)
	assert.Len(t, findings, 1, "empty RepoPath should skip disk check and generate finding")
}

func TestMissingTestCritic_StatError(t *testing.T) {
	// Create a directory that prevents stat access.
	repoPath := t.TempDir()
	dir := filepath.Join(repoPath, "internal")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	// Create a test file, then make the directory unreadable.
	testFile := filepath.Join(dir, "foo_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package internal"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	c := NewMissingTestCritic()
	input := ReviewInput{
		Diff: &git.DiffResult{},
		ClassifiedFiles: []git.ClassifiedFile{
			{
				ChangedFile: git.ChangedFile{
					Path:   "internal/foo.go",
					Status: git.FileStatusAdded,
				},
				Category: git.FileCategoryCode,
			},
		},
		RepoPath: repoPath,
	}

	_, err := c.Review(context.Background(), input)
	assert.Error(t, err, "non-ErrNotExist stat error should propagate")
	assert.Contains(t, err.Error(), "checking test file")
}
