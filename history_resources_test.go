package main

import (
	"testing"
)

func TestExtractHistoryIDFromFullURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "standard full URI",
			uri:      "orm-mcp://history/req_abc123/full",
			expected: "req_abc123",
		},
		{
			name:     "full URI with query params",
			uri:      "orm-mcp://history/req_xyz789/full?format=json",
			expected: "req_xyz789",
		},
		{
			name:     "full URI with underscore and numbers",
			uri:      "orm-mcp://history/req_12345678/full",
			expected: "req_12345678",
		},
		{
			name:     "recent should not be ID",
			uri:      "orm-mcp://history/recent/full",
			expected: "",
		},
		{
			name:     "search should not be ID",
			uri:      "orm-mcp://history/search/full",
			expected: "",
		},
		{
			name:     "empty path",
			uri:      "orm-mcp://history/",
			expected: "",
		},
		{
			name:     "URL-encoded underscore in ID",
			uri:      "orm-mcp://history/req%5Fabc123/full",
			expected: "req_abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHistoryIDFromFullURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractHistoryIDFromFullURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestExtractHistoryIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "standard detail URI",
			uri:      "orm-mcp://history/req_abc123",
			expected: "req_abc123",
		},
		{
			name:     "detail URI with query params",
			uri:      "orm-mcp://history/req_xyz789?foo=bar",
			expected: "req_xyz789",
		},
		{
			name:     "recent is not an ID",
			uri:      "orm-mcp://history/recent",
			expected: "",
		},
		{
			name:     "search is not an ID",
			uri:      "orm-mcp://history/search",
			expected: "",
		},
		{
			name:     "URL-encoded underscore in ID",
			uri:      "orm-mcp://history/req%5Fabc123",
			expected: "req_abc123",
		},
		{
			name:     "URL-encoded ID with query params",
			uri:      "orm-mcp://history/req%5Fabc123?foo=bar",
			expected: "req_abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHistoryIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractHistoryIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestExtractHistorySearchParams(t *testing.T) {
	tests := []struct {
		name            string
		uri             string
		expectedKeyword string
		expectedType    string
	}{
		{
			name:            "keyword only",
			uri:             "orm-mcp://history/search?keyword=docker",
			expectedKeyword: "docker",
			expectedType:    "",
		},
		{
			name:            "type only",
			uri:             "orm-mcp://history/search?type=search",
			expectedKeyword: "",
			expectedType:    "search",
		},
		{
			name:            "both keyword and type",
			uri:             "orm-mcp://history/search?keyword=react&type=question",
			expectedKeyword: "react",
			expectedType:    "question",
		},
		{
			name:            "no params",
			uri:             "orm-mcp://history/search",
			expectedKeyword: "",
			expectedType:    "",
		},
		{
			name:            "empty params",
			uri:             "orm-mcp://history/search?",
			expectedKeyword: "",
			expectedType:    "",
		},
		{
			name:            "URL-encoded C++ keyword",
			uri:             "orm-mcp://history/search?keyword=C%2B%2B&type=search",
			expectedKeyword: "C++",
			expectedType:    "search",
		},
		{
			name:            "URL-encoded Japanese keyword",
			uri:             "orm-mcp://history/search?keyword=%E3%83%86%E3%82%B9%E3%83%88",
			expectedKeyword: "テスト",
			expectedType:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyword, entryType := extractHistorySearchParams(tt.uri)
			if keyword != tt.expectedKeyword {
				t.Errorf("keyword: got %q, want %q", keyword, tt.expectedKeyword)
			}
			if entryType != tt.expectedType {
				t.Errorf("type: got %q, want %q", entryType, tt.expectedType)
			}
		})
	}
}
