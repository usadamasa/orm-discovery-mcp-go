package git

import (
	"path/filepath"
	"strings"
)

// FileCategory represents the classification of a changed file.
type FileCategory string

const (
	FileCategoryCode   FileCategory = "code"
	FileCategoryTest   FileCategory = "test"
	FileCategoryInfra  FileCategory = "infra"
	FileCategoryConfig FileCategory = "config"
	FileCategoryOther  FileCategory = "other"
)

// Valid reports whether c is a recognized FileCategory value.
func (c FileCategory) Valid() bool {
	switch c {
	case FileCategoryCode, FileCategoryTest, FileCategoryInfra, FileCategoryConfig, FileCategoryOther:
		return true
	}
	return false
}

// ClassifyFile determines the category of a file based on its path.
// Priority: test > infra > config > code > other.
func ClassifyFile(path string) FileCategory {
	base := filepath.Base(path)
	ext := filepath.Ext(path)

	// test: *_test.go, *.test.js, *_test.ts, testdata/**, test/**
	if isTestFile(path, base, ext) {
		return FileCategoryTest
	}

	// infra: Dockerfile*, docker-compose*, .github/workflows/**, *.tf, k8s/**, kubernetes/**
	if isInfraFile(path, base, ext) {
		return FileCategoryInfra
	}

	// config: go.mod, go.sum, *.toml, *.yaml, *.yml, .env*, .gitignore, .golangci*, aqua.yaml, Taskfile.yml, Makefile
	if isConfigFile(path, base, ext) {
		return FileCategoryConfig
	}

	// code: *.go, *.ts, *.js, *.py, *.rs, *.java
	if isCodeFile(ext) {
		return FileCategoryCode
	}

	return FileCategoryOther
}

// ClassifyChangedFiles adds category information to a list of changed files.
func ClassifyChangedFiles(files []ChangedFile) []ClassifiedFile {
	if len(files) == 0 {
		return nil
	}

	result := make([]ClassifiedFile, 0, len(files))
	for _, f := range files {
		result = append(result, ClassifiedFile{
			ChangedFile: f,
			Category:    ClassifyFile(f.Path),
		})
	}
	return result
}

// ClassifiedFile extends ChangedFile with category information.
type ClassifiedFile struct {
	ChangedFile
	Category FileCategory `json:"category"`
}

func isTestFile(path, base, ext string) bool {
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.ts") {
		return true
	}
	if strings.HasSuffix(base, ".test.js") {
		return true
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		if p == "testdata" || p == "test" {
			return true
		}
	}
	return false
}

func isInfraFile(path, base, ext string) bool {
	if strings.HasPrefix(base, "Dockerfile") || strings.HasPrefix(base, "docker-compose") {
		return true
	}
	if ext == ".tf" {
		return true
	}
	normalized := filepath.ToSlash(path)
	if strings.HasPrefix(normalized, ".github/workflows/") {
		return true
	}
	parts := strings.Split(normalized, "/")
	for _, p := range parts {
		if p == "k8s" || p == "kubernetes" {
			return true
		}
	}
	return false
}

func isConfigFile(path, base, ext string) bool {
	switch base {
	case "go.mod", "go.sum", "Makefile", "Taskfile.yml", "aqua.yaml":
		return true
	}
	if strings.HasPrefix(base, ".env") {
		return true
	}
	if strings.HasPrefix(base, ".golangci") {
		return true
	}
	if base == ".gitignore" {
		return true
	}
	switch ext {
	case ".toml", ".yaml", ".yml":
		return true
	}
	return false
}

func isCodeFile(ext string) bool {
	switch ext {
	case ".go", ".ts", ".js", ".py", ".rs", ".java":
		return true
	}
	return false
}
