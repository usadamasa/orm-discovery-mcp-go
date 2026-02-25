package critic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/git"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/model"
)

// MissingTestCritic detects code files without corresponding test files.
type MissingTestCritic struct{}

// NewMissingTestCritic creates a new MissingTestCritic.
func NewMissingTestCritic() *MissingTestCritic {
	return &MissingTestCritic{}
}

// Name returns the name of this critic.
func (c *MissingTestCritic) Name() string {
	return "MissingTestCritic"
}

// Review checks for code files missing corresponding test files.
// Only Added or Modified code files are checked. The review first looks for
// the test file in the diff, then checks the filesystem at RepoPath if provided.
// If RepoPath is empty, only diff-based checking is performed.
func (c *MissingTestCritic) Review(_ context.Context, input ReviewInput) ([]model.Finding, error) {
	if len(input.ClassifiedFiles) == 0 {
		return nil, nil
	}

	// Collect test file paths present in the diff.
	testFilesInDiff := make(map[string]bool)
	for _, f := range input.ClassifiedFiles {
		if f.Category == git.FileCategoryTest {
			testFilesInDiff[f.Path] = true
		}
	}

	var findings []model.Finding
	for _, f := range input.ClassifiedFiles {
		if f.Category != git.FileCategoryCode {
			continue
		}
		if f.Status != git.FileStatusAdded && f.Status != git.FileStatusModified {
			continue
		}

		testPath := deriveTestPath(f.Path)

		// Check if test file is in the diff.
		if testFilesInDiff[testPath] {
			continue
		}

		// Check if test file exists on disk.
		if input.RepoPath != "" {
			absPath := filepath.Join(input.RepoPath, testPath)
			_, err := os.Stat(absPath)
			if err == nil {
				continue
			}
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("checking test file %s: %w", testPath, err)
			}
		}

		finding := model.NewFinding(
			model.SeverityWarning,
			model.CategoryMissingTest,
			fmt.Sprintf("code file %s has no corresponding test file", filepath.Base(f.Path)),
			model.Location{FilePath: f.Path},
		)
		finding.CriticName = c.Name()
		finding.Suggestion = fmt.Sprintf("consider adding %s", filepath.Base(testPath))
		findings = append(findings, finding)
	}

	return findings, nil
}

// deriveTestPath returns the expected test file path for a given source file
// by inserting "_test" before the file extension.
// Examples:
//
//	"internal/foo.go"  -> "internal/foo_test.go"
//	"pkg/bar.ts"       -> "pkg/bar_test.ts"
func deriveTestPath(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + "_test" + ext
}
