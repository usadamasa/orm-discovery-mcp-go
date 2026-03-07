package main

import (
	"encoding/json"
	"fmt"
	"math"
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
// Uses ~3.5 chars per token as a rough approximation for English text.
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
// This test always passes — it's a measurement tool for before/after comparison.
func TestContextEfficiencyReport(t *testing.T) {
	// A. Tool descriptions (from tool_descriptions.go constants)
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

	// B. Resource descriptions (from tool_descriptions.go constants)
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

	// C. Resource template descriptions (from tool_descriptions.go constants)
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

	// D. Prompt descriptions (from tool_descriptions.go constants)
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

	// E. Total payload summary
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

// syntheticBFSResults generates n synthetic BFS-mode search results.
func syntheticBFSResults(n int) []map[string]any {
	results := make([]map[string]any, n)
	for i := range n {
		results[i] = map[string]any{
			"id":      fmt.Sprintf("978014310%04d", i),
			"title":   fmt.Sprintf("Sample Book Title Number %d: A Comprehensive Guide", i),
			"authors": []string{"John Author", "Jane Writer"},
		}
	}
	return results
}

// syntheticDFSResults generates n synthetic DFS-mode search results with full detail.
func syntheticDFSResults(n int) []map[string]any {
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

// TestBFSResponseSize measures the JSON size of BFS mode responses.
func TestBFSResponseSize(t *testing.T) {
	results := syntheticBFSResults(25)
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("failed to marshal BFS results: %v", err)
	}
	t.Logf("BFS response (25 results): %d bytes (~%.1f KB)", len(data), float64(len(data))/1024)
}

// TestDFSResponseSize measures the JSON size of DFS mode responses.
func TestDFSResponseSize(t *testing.T) {
	results := syntheticDFSResults(25)
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("failed to marshal DFS results: %v", err)
	}
	t.Logf("DFS response (25 results): %d bytes (~%.1f KB)", len(data), float64(len(data))/1024)
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
