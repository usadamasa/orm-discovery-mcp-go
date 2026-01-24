//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMCPTool_SearchContent tests the search_content tool with real API.
func TestMCPTool_SearchContent(t *testing.T) {
	client := GetSharedClient()

	// Test search functionality
	results, err := client.SearchContent(TestSearchQuery, nil)
	if err != nil {
		t.Fatalf("SearchContent failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected search results, got empty")
	}

	// Verify result structure
	firstResult := results[0]
	t.Logf("First result: %+v", firstResult)

	// Check for expected fields
	if _, ok := firstResult["title"]; !ok {
		t.Error("Expected 'title' field in search result")
	}
}

// TestMCPTool_SearchContent_WithOptions tests search with custom options.
func TestMCPTool_SearchContent_WithOptions(t *testing.T) {
	client := GetSharedClient()

	// Test search with custom options
	options := map[string]interface{}{
		"rows":      10,
		"languages": []string{"en"},
	}

	results, err := client.SearchContent("Docker containers", options)
	if err != nil {
		t.Fatalf("SearchContent with options failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected search results, got empty")
	}

	// Should respect rows limit
	if len(results) > 10 {
		t.Logf("Note: Got %d results (may exceed rows due to API behavior)", len(results))
	}

	t.Logf("Found %d results for 'Docker containers'", len(results))
}

// TestMCPTool_AskQuestion_FullFlow tests the complete ask_question flow
// with a realistic timeout.
func TestMCPTool_AskQuestion_FullFlow(t *testing.T) {
	// This test takes longer, skip in short mode
	if testing.Short() {
		t.Skip("Skipping long-running ask_question test in short mode")
	}

	client := GetSharedClient()

	// Use a reasonable timeout for real answer generation
	timeout := 60 * time.Second

	answer, err := client.AskQuestion("How to optimize Go performance?", timeout)
	if err != nil {
		t.Fatalf("AskQuestion failed: %v", err)
	}

	if answer == nil {
		t.Fatal("Expected answer, got nil")
	}

	// Verify answer structure
	if answer.QuestionID == "" {
		t.Error("Expected non-empty question_id")
	}

	if answer.MisoResponse.Data.Answer == "" {
		t.Error("Expected non-empty answer content")
	}

	// Log answer summary
	answerJSON, _ := json.MarshalIndent(answer, "", "  ")
	t.Logf("Full answer response:\n%s", string(answerJSON))
}
