package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify_ASCII(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Docker containers", "docker-containers"},
		{"Go Programming", "go-programming"},
		{"hello world", "hello-world"},
		{"test123", "test123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSlugify_Japanese(t *testing.T) {
	// Japanese characters are non-ASCII, should be replaced with "-"
	got := slugify("日本語テスト")
	if got == "" {
		t.Error("slugify should not return empty for Japanese input")
	}
	// Should produce a hash-based fallback since all chars are non-alphanumeric ASCII
	if !strings.HasPrefix(got, "query-") {
		t.Errorf("slugify(Japanese) = %q, expected hash-based fallback starting with 'query-'", got)
	}
}

func TestSlugify_Empty(t *testing.T) {
	got := slugify("")
	if !strings.HasPrefix(got, "query-") {
		t.Errorf("slugify('') = %q, expected hash-based fallback", got)
	}
}

func TestSlugify_TruncatesLong(t *testing.T) {
	long := strings.Repeat("abcdefghij", 10) // 100 chars
	got := slugify(long)
	if len(got) > 50 {
		t.Errorf("slugify should truncate to 50 chars, got %d: %q", len(got), got)
	}
}

func TestSaveResponseAsMarkdown_CreatesFile(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "responses")

	results := []map[string]any{
		{
			"product_id":   "123",
			"title":        "Docker: Up & Running",
			"authors":      []any{"Sean P. Kane"},
			"content_type": "book",
			"publisher":    "O'Reilly Media",
			"description":  "A practical guide to Docker.",
			"url":          "https://learning.oreilly.com/library/view/-/123/",
		},
		{
			"product_id":   "456",
			"title":        "Kubernetes in Action",
			"authors":      []any{"Marko Lukša"},
			"content_type": "book",
		},
	}

	filePath, err := saveResponseAsMarkdown(cacheDir, "Docker containers", results, "req_abc123", 100)
	if err != nil {
		t.Fatalf("saveResponseAsMarkdown failed: %v", err)
	}

	// File should exist
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("file does not exist: %v", err)
	}

	// File permissions should be 0600
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permission = %o, want 0600", perm)
	}

	// Read and verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)

	// Check header
	if !strings.Contains(content, "# O'Reilly Search Results") {
		t.Error("missing header")
	}
	if !strings.Contains(content, "Docker containers") {
		t.Error("missing query")
	}
	if !strings.Contains(content, "req_abc123") {
		t.Error("missing history ID")
	}
	if !strings.Contains(content, "Total Results: 100") {
		t.Error("missing total results")
	}
	if !strings.Contains(content, "Results in this file: 2") {
		t.Error("missing results count")
	}

	// Check result content
	if !strings.Contains(content, "Docker: Up & Running") {
		t.Error("missing result title")
	}
	if !strings.Contains(content, "Sean P. Kane") {
		t.Error("missing author")
	}
	if !strings.Contains(content, "A practical guide to Docker.") {
		t.Error("missing description")
	}

	// Filename should contain slug
	if !strings.Contains(filepath.Base(filePath), "docker-containers") {
		t.Errorf("filename should contain slug, got %q", filepath.Base(filePath))
	}
}

func TestSaveResponseAsMarkdown_NonWritableDir(t *testing.T) {
	// Use a path that can't be created
	cacheDir := "/dev/null/impossible/path"

	results := []map[string]any{
		{"title": "Test"},
	}

	_, err := saveResponseAsMarkdown(cacheDir, "test", results, "req_123", 1)
	if err == nil {
		t.Error("expected error for non-writable directory")
	}
}

func TestSaveResponseAsMarkdown_EmptyResults(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "responses")

	filePath, err := saveResponseAsMarkdown(cacheDir, "empty query", []map[string]any{}, "req_empty", 0)
	if err != nil {
		t.Fatalf("saveResponseAsMarkdown failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "Results in this file: 0") {
		t.Error("missing results count for empty results")
	}
}
