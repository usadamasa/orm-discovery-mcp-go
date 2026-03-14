package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// maxToolDescriptionLen is the guardrail limit for tool description length.
// Target: ~270-280 chars per progressive-disclosure SKILL.md guidelines.
const maxToolDescriptionLen = 350

// TestToolDescriptionSizes verifies that tool descriptions stay within the guardrail limit.
func TestToolDescriptionSizes(t *testing.T) {
	tests := []struct {
		name string
		desc string
	}{
		{"descSearchContent", descSearchContent},
		{"descAskQuestion", descAskQuestion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charCount := utf8.RuneCountInString(tt.desc)
			t.Logf("%s: %d chars (limit: %d)", tt.name, charCount, maxToolDescriptionLen)
			if charCount > maxToolDescriptionLen {
				t.Errorf("%s is %d chars, exceeds limit of %d chars (over by %d)",
					tt.name, charCount, maxToolDescriptionLen, charCount-maxToolDescriptionLen)
			}
		})
	}
}

// estimateTokens returns an estimated token count for the given text.
func estimateTokens(s string) int {
	return int(math.Ceil(float64(utf8.RuneCountInString(s)) / 3.5))
}

// estimateTokensFromCount returns an estimated token count from a char count.
func estimateTokensFromCount(charCount int) int {
	return int(math.Ceil(float64(charCount) / 3.5))
}

// namedDesc pairs a label with a description string for deterministic report ordering.
type namedDesc struct {
	name string
	desc string
}

// TestContextEfficiencyReport outputs a comprehensive report of all MCP metadata sizes.
func TestContextEfficiencyReport(t *testing.T) {
	toolDescs := []namedDesc{
		{"oreilly_search_content", descSearchContent},
		{"oreilly_ask_question", descAskQuestion},
	}

	totalToolChars := 0
	t.Log("=== A. Tool Descriptions ===")
	t.Logf("%-30s %8s %8s", "Tool", "Chars", "~Tokens")
	t.Logf("%-30s %8s %8s", "----", "-----", "-------")
	for _, td := range toolDescs {
		chars := utf8.RuneCountInString(td.desc)
		totalToolChars += chars
		t.Logf("%-30s %8d %8d", td.name, chars, estimateTokens(td.desc))
	}
	t.Logf("%-30s %8d %8d", "TOTAL (Tools)", totalToolChars, estimateTokensFromCount(totalToolChars))

	resourceDescs := []namedDesc{
		{"book-details", descResBookDetails},
		{"book-toc", descResBookTOC},
		{"book-chapter", descResBookChapter},
		{"answer", descResAnswer},
		{"history/recent", descResHistRecent},
	}

	totalResourceChars := 0
	t.Log("")
	t.Log("=== B. Resource Descriptions ===")
	t.Logf("%-30s %8s %8s", "Resource", "Chars", "~Tokens")
	t.Logf("%-30s %8s %8s", "--------", "-----", "-------")
	for _, td := range resourceDescs {
		chars := utf8.RuneCountInString(td.desc)
		totalResourceChars += chars
		t.Logf("%-30s %8d %8d", td.name, chars, estimateTokens(td.desc))
	}

	templateDescs := []namedDesc{
		{"book-details-tmpl", descTmplBookDetails},
		{"book-toc-tmpl", descTmplBookTOC},
		{"book-chapter-tmpl", descTmplBookChapter},
		{"answer-tmpl", descTmplAnswer},
		{"history/search-tmpl", descTmplHistSearch},
		{"history/{id}-tmpl", descTmplHistDetail},
		{"history/{id}/full-tmpl", descTmplHistFull},
	}

	totalTemplateChars := 0
	t.Log("")
	t.Log("=== C. Resource Template Descriptions ===")
	t.Logf("%-30s %8s %8s", "Template", "Chars", "~Tokens")
	t.Logf("%-30s %8s %8s", "--------", "-----", "-------")
	for _, td := range templateDescs {
		chars := utf8.RuneCountInString(td.desc)
		totalTemplateChars += chars
		t.Logf("%-30s %8d %8d", td.name, chars, estimateTokens(td.desc))
	}

	promptDescs := []namedDesc{
		{"learn-technology", descPromptLearnTech},
		{"review-history", descPromptReviewHist},
		{"continue-research", descPromptContRes},
		{"research-topic", descPromptResTopic},
		{"debug-error", descPromptDebugErr},
		{"summarize-history", descPromptSumHist},
	}

	totalPromptChars := 0
	t.Log("")
	t.Log("=== D. Prompt Descriptions ===")
	t.Logf("%-30s %8s %8s", "Prompt", "Chars", "~Tokens")
	t.Logf("%-30s %8s %8s", "------", "-----", "-------")
	for _, td := range promptDescs {
		chars := utf8.RuneCountInString(td.desc)
		totalPromptChars += chars
		t.Logf("%-30s %8d %8d", td.name, chars, estimateTokens(td.desc))
	}

	grandTotal := totalToolChars + totalResourceChars + totalTemplateChars + totalPromptChars
	t.Log("")
	t.Log("=== E. Total MCP Metadata Payload ===")
	t.Logf("%-30s %8s %8s", "Category", "Chars", "~Tokens")
	t.Logf("%-30s %8s %8s", "--------", "-----", "-------")
	t.Logf("%-30s %8d %8d", "Tools", totalToolChars, estimateTokensFromCount(totalToolChars))
	t.Logf("%-30s %8d %8d", "Resources", totalResourceChars, estimateTokensFromCount(totalResourceChars))
	t.Logf("%-30s %8d %8d", "Templates", totalTemplateChars, estimateTokensFromCount(totalTemplateChars))
	t.Logf("%-30s %8d %8d", "Prompts", totalPromptChars, estimateTokensFromCount(totalPromptChars))
	t.Logf("%-30s %8d %8d", "GRAND TOTAL", grandTotal, estimateTokensFromCount(grandTotal))
}

// syntheticSearchResults generates n synthetic search results with full detail.
func syntheticSearchResults(n int) []map[string]any {
	results := make([]map[string]any, n)
	for i := range n {
		results[i] = map[string]any{
			"product_id":       fmt.Sprintf("978014310%04d", i),
			"title":            fmt.Sprintf("Sample Book Title Number %d: A Comprehensive Guide to Modern Development", i),
			"authors":          []string{"John Author", "Jane Writer", "Bob Developer"},
			"content_type":     "book",
			"description":      "This is a detailed description of the book that covers many topics in modern software development, including best practices, design patterns, and real-world examples.",
			"isbn":             fmt.Sprintf("978-0-14-310%04d", i),
			"publisher":        "O'Reilly Media",
			"publication_date": "2024-01-15",
			"pages":            450,
			"language":         "en",
			"topics":           []string{"programming", "software-engineering", "best-practices"},
			"average_rating":   4.5,
			"url":              fmt.Sprintf("https://learning.oreilly.com/library/view/-/978014310%04d/", i),
		}
	}
	return results
}

// syntheticHistoryEntries generates n synthetic research history entries.
func syntheticHistoryEntries(n int) []ResearchEntry {
	entries := make([]ResearchEntry, n)
	for i := range n {
		entries[i] = ResearchEntry{
			ID:       fmt.Sprintf("req_%06d", i),
			Type:     "search",
			Query:    fmt.Sprintf("sample search query %d", i),
			ToolName: "oreilly_search_content",
			ResultSummary: ResultSummary{
				Count: 25,
				TopResults: []TopResultSummary{
					{Title: "Book A", Author: "Author X", ProductID: "111"},
					{Title: "Book B", Author: "Author Y", ProductID: "222"},
				},
			},
			DurationMs: 1200,
		}
	}
	return entries
}

// TestCacheFileFormat_Regression verifies that cache files have no HTML tags
// in descriptions and that Total Results is accurate.
func TestCacheFileFormat_Regression(t *testing.T) {
	results := syntheticSearchResults(5)
	// Inject HTML into descriptions
	for i := range results {
		results[i]["description"] = fmt.Sprintf("<span class=\"highlight\">Book %d</span> covers <div>important topics</div> in <p>software</p> development.", i)
	}

	cacheDir := t.TempDir()
	filePath, err := saveResponseAsMarkdown(cacheDir, "regression test", results, "req_regression", 42)
	if err != nil {
		t.Fatalf("saveResponseAsMarkdown failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	content := string(data)

	// No HTML tags in output
	htmlTags := []string{"<span", "</span>", "<div", "</div>", "<p>", "</p>"}
	for _, tag := range htmlTags {
		if strings.Contains(content, tag) {
			t.Errorf("cache file contains HTML tag %q", tag)
		}
	}

	// Total Results should be accurate
	if !strings.Contains(content, "Total Results: 42") {
		t.Error("cache file does not contain accurate Total Results")
	}

	// Text content should be preserved
	if !strings.Contains(content, "important topics") {
		t.Error("cache file should contain text content 'important topics'")
	}
}

// TestLightweightResponseSize measures the JSON size of lightweight responses
// and ensures they stay under 2KB for 25-result searches.
func TestLightweightResponseSize(t *testing.T) {
	srv := &Server{}
	results := syntheticSearchResults(25)

	_, structured := srv.buildLightweightResponse(results, "req_test123", "/tmp/cache/test.md", 0, 100)

	data, err := json.Marshal(structured)
	if err != nil {
		t.Fatalf("failed to marshal lightweight response: %v", err)
	}

	sizeKB := float64(len(data)) / 1024
	t.Logf("Lightweight response (25 results): %d bytes (~%.1f KB)", len(data), sizeKB)

	if len(data) > 2048 {
		t.Errorf("Lightweight response is %d bytes (%.1f KB), exceeds 2KB guardrail", len(data), sizeKB)
	}
}

// TestCacheFileSize measures the Markdown file size for 25-result searches.
func TestCacheFileSize(t *testing.T) {
	results := syntheticSearchResults(25)
	cacheDir := t.TempDir()

	filePath, err := saveResponseAsMarkdown(cacheDir, "test query", results, "req_test123", 100)
	if err != nil {
		t.Fatalf("failed to save cache file: %v", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat cache file: %v", err)
	}

	sizeKB := float64(info.Size()) / 1024
	t.Logf("Cache file (25 results): %d bytes (~%.1f KB)", info.Size(), sizeKB)
}

// TestHistoryRecentResponseSize measures the JSON size of history/recent responses.
func TestHistoryRecentResponseSize(t *testing.T) {
	entries := syntheticHistoryEntries(20)
	response := struct {
		Count   int             `json:"count"`
		Entries []ResearchEntry `json:"entries"`
	}{
		Count:   len(entries),
		Entries: entries,
	}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal history response: %v", err)
	}
	t.Logf("history/recent response (20 entries): %d bytes (~%.1f KB)", len(data), float64(len(data))/1024)
}
