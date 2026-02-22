package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected FileCategory
	}{
		// test
		{"Go test file", "foo_test.go", FileCategoryTest},
		{"Nested Go test file", "internal/git/diff_test.go", FileCategoryTest},
		{"testdata file", "testdata/fixture.json", FileCategoryTest},
		{"nested testdata", "internal/testdata/sample.txt", FileCategoryTest},
		{"test directory", "test/integration.go", FileCategoryTest},
		{"TS test file", "app_test.ts", FileCategoryTest},
		{"JS test file", "utils.test.js", FileCategoryTest},

		// infra
		{"Dockerfile", "Dockerfile", FileCategoryInfra},
		{"Dockerfile with suffix", "Dockerfile.prod", FileCategoryInfra},
		{"docker-compose", "docker-compose.yml", FileCategoryInfra},
		{"GitHub Actions", ".github/workflows/ci.yml", FileCategoryInfra},
		{"Terraform file", "infra/main.tf", FileCategoryInfra},
		{"k8s directory", "k8s/deployment.yaml", FileCategoryInfra},
		{"kubernetes directory", "kubernetes/service.yaml", FileCategoryInfra},

		// config
		{"go.mod", "go.mod", FileCategoryConfig},
		{"go.sum", "go.sum", FileCategoryConfig},
		{"TOML config", "config.toml", FileCategoryConfig},
		{"YAML config", "config.yaml", FileCategoryConfig},
		{"YML config", "settings.yml", FileCategoryConfig},
		{"env file", ".env", FileCategoryConfig},
		{"env.local", ".env.local", FileCategoryConfig},
		{"gitignore", ".gitignore", FileCategoryConfig},
		{"golangci config", ".golangci.yml", FileCategoryConfig},
		{"aqua.yaml", "aqua.yaml", FileCategoryConfig},
		{"Taskfile", "Taskfile.yml", FileCategoryConfig},
		{"Makefile", "Makefile", FileCategoryConfig},

		// code
		{"Go source file", "main.go", FileCategoryCode},
		{"Nested Go source", "internal/git/diff.go", FileCategoryCode},
		{"TypeScript file", "app.ts", FileCategoryCode},
		{"JavaScript file", "utils.js", FileCategoryCode},
		{"Python file", "script.py", FileCategoryCode},
		{"Rust file", "main.rs", FileCategoryCode},
		{"Java file", "App.java", FileCategoryCode},

		// other
		{"README", "README.md", FileCategoryOther},
		{"LICENSE", "LICENSE", FileCategoryOther},
		{"Text file", "notes.txt", FileCategoryOther},
		{"Unknown extension", "data.dat", FileCategoryOther},
		{"No extension", "CODEOWNERS", FileCategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyFile(tt.path)
			assert.Equal(t, tt.expected, got, "ClassifyFile(%q)", tt.path)
		})
	}
}

func TestClassifyChangedFiles(t *testing.T) {
	input := []ChangedFile{
		{Path: "internal/git/diff.go", Status: FileStatusModified, Additions: 5, Deletions: 2},
		{Path: "internal/git/diff_test.go", Status: FileStatusModified, Additions: 10, Deletions: 0},
		{Path: "Dockerfile", Status: FileStatusAdded, Additions: 20, Deletions: 0},
		{Path: "go.mod", Status: FileStatusModified, Additions: 1, Deletions: 1},
		{Path: "README.md", Status: FileStatusModified, Additions: 3, Deletions: 1},
	}

	result := ClassifyChangedFiles(input)

	assert.Len(t, result, 5)

	assert.Equal(t, FileCategoryCode, result[0].Category)
	assert.Equal(t, "internal/git/diff.go", result[0].Path)

	assert.Equal(t, FileCategoryTest, result[1].Category)
	assert.Equal(t, "internal/git/diff_test.go", result[1].Path)

	assert.Equal(t, FileCategoryInfra, result[2].Category)
	assert.Equal(t, "Dockerfile", result[2].Path)

	assert.Equal(t, FileCategoryConfig, result[3].Category)
	assert.Equal(t, "go.mod", result[3].Path)

	assert.Equal(t, FileCategoryOther, result[4].Category)
	assert.Equal(t, "README.md", result[4].Path)

	// Verify ChangedFile fields are preserved
	assert.Equal(t, FileStatusModified, result[0].Status)
	assert.Equal(t, 5, result[0].Additions)
	assert.Equal(t, 2, result[0].Deletions)
}

func TestClassifyChangedFiles_Empty(t *testing.T) {
	result := ClassifyChangedFiles(nil)
	assert.Empty(t, result)

	result = ClassifyChangedFiles([]ChangedFile{})
	assert.Empty(t, result)
}

func TestFileCategory_Valid(t *testing.T) {
	tests := []struct {
		name     string
		category FileCategory
		want     bool
	}{
		{"code", FileCategoryCode, true},
		{"test", FileCategoryTest, true},
		{"infra", FileCategoryInfra, true},
		{"config", FileCategoryConfig, true},
		{"other", FileCategoryOther, true},
		{"empty string", FileCategory(""), false},
		{"unknown", FileCategory("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.category.Valid())
		})
	}
}
