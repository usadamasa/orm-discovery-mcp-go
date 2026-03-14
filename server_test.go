package main

import (
	"testing"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

func TestExtractProductIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "book-details URI",
			uri:      "oreilly://book-details/12345",
			expected: "12345",
		},
		{
			name:     "book-toc URI",
			uri:      "oreilly://book-toc/12345",
			expected: "12345",
		},
		{
			name:     "URL-encoded product ID",
			uri:      "oreilly://book-details/978%2D1%2D491",
			expected: "978-1-491",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "trailing slash only",
			uri:      "oreilly://book-details/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProductIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractProductIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestExtractProductIDAndChapterFromURI(t *testing.T) {
	tests := []struct {
		name            string
		uri             string
		expectedProduct string
		expectedChapter string
	}{
		{
			name:            "standard chapter URI",
			uri:             "oreilly://book-chapter/12345/ch01.html",
			expectedProduct: "12345",
			expectedChapter: "ch01.html",
		},
		{
			name:            "URL-encoded slash in chapter name",
			uri:             "oreilly://book-chapter/12345/ch%2F01.html",
			expectedProduct: "12345",
			expectedChapter: "ch/01.html",
		},
		{
			name:            "URL-encoded space in chapter name",
			uri:             "oreilly://book-chapter/12345/ch01%20intro.html",
			expectedProduct: "12345",
			expectedChapter: "ch01 intro.html",
		},
		{
			name:            "empty URI",
			uri:             "",
			expectedProduct: "",
			expectedChapter: "",
		},
		{
			name:            "missing chapter",
			uri:             "oreilly://book-chapter/12345",
			expectedProduct: "",
			expectedChapter: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, chapter := extractProductIDAndChapterFromURI(tt.uri)
			if product != tt.expectedProduct {
				t.Errorf("product: got %q, want %q", product, tt.expectedProduct)
			}
			if chapter != tt.expectedChapter {
				t.Errorf("chapter: got %q, want %q", chapter, tt.expectedChapter)
			}
		})
	}
}

func TestExtractQuestionIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "standard answer URI",
			uri:      "oreilly://answer/q-abc-123",
			expected: "q-abc-123",
		},
		{
			name:     "URL-encoded hyphen",
			uri:      "oreilly://answer/q%2Dabc",
			expected: "q-abc",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractQuestionIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractQuestionIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestBuildLightweightResponse_BrowserAuthorConversion(t *testing.T) {
	srv := &Server{}

	results := []map[string]any{
		{
			"id":           "123",
			"title":        "Go Programming",
			"content_type": "book",
			"authors": []browser.Author{
				{Name: "John Doe"},
				{Name: "Jane Smith"},
			},
		},
	}

	_, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/test.md", 0, 1)

	if structured == nil || len(structured.Results) == 0 {
		t.Fatal("expected structured results")
	}

	authors, ok := structured.Results[0]["authors"]
	if !ok {
		t.Fatal("expected authors key in result")
	}

	authorNames, ok := authors.([]string)
	if !ok {
		t.Fatalf("expected []string authors, got %T", authors)
	}
	if len(authorNames) != 2 || authorNames[0] != "John Doe" || authorNames[1] != "Jane Smith" {
		t.Errorf("expected [John Doe, Jane Smith], got %v", authorNames)
	}
}

func TestBuildLightweightResponse_FilePath(t *testing.T) {
	srv := &Server{}

	results := []map[string]any{
		{"id": "123", "title": "Test Book", "content_type": "book"},
	}

	toolResult, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/cache/test.md", 0, 1)

	if structured == nil {
		t.Fatal("expected structured result")
	}

	if structured.FilePath != "/tmp/cache/test.md" {
		t.Errorf("expected FilePath '/tmp/cache/test.md', got %q", structured.FilePath)
	}

	if structured.HistoryID != "hist_123" {
		t.Errorf("expected HistoryID 'hist_123', got %q", structured.HistoryID)
	}

	// Tool result should contain text with file path
	if toolResult == nil {
		t.Fatal("expected tool result with text content")
	}
}

func TestBuildLightweightResponse_LimitsTo5Results(t *testing.T) {
	srv := &Server{}

	results := make([]map[string]any, 10)
	for i := range results {
		results[i] = map[string]any{
			"id":    "id-" + string(rune('0'+i)),
			"title": "Book " + string(rune('0'+i)),
		}
	}

	toolResult, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/test.md", 0, 50)

	if structured == nil {
		t.Fatal("expected structured result")
	}

	// Only top 5 results should be in structured output for context efficiency
	if len(structured.Results) != 5 {
		t.Errorf("expected 5 results in structured (top 5 only), got %d", len(structured.Results))
	}

	// But Count should reflect total results returned by API
	if structured.Count != 10 {
		t.Errorf("expected Count 10, got %d", structured.Count)
	}

	// But text should mention "and X more results"
	if toolResult == nil || len(toolResult.Content) == 0 {
		t.Fatal("expected tool result content")
	}
}
