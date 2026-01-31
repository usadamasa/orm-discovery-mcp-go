package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "simple query",
			query:    "Docker containers",
			expected: []string{"docker", "containers"},
		},
		{
			name:     "query with stop words",
			query:    "How to use Docker for containers",
			expected: []string{"use", "docker", "containers"},
		},
		{
			name:     "query with special characters",
			query:    "What is React.js?",
			expected: []string{"react", "js"},
		},
		{
			name:     "empty query",
			query:    "",
			expected: []string{},
		},
		{
			name:     "only stop words",
			query:    "how to do the thing",
			expected: []string{"thing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractKeywords(tt.query)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d keywords, got %d", len(tt.expected), len(result))
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("keyword %d: expected %q, got %q", i, tt.expected[i], kw)
				}
			}
		})
	}
}

func TestResearchHistoryManager_AddAndSearch(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "research-history.json")

	manager := NewResearchHistoryManager(filePath, 100)
	if err := manager.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Add a search entry
	entry1 := ResearchEntry{
		Type:     "search",
		Query:    "Docker containers",
		ToolName: "search_content",
		ResultSummary: ResultSummary{
			Count: 10,
			TopResults: []TopResultSummary{
				{Title: "Docker: Up & Running", Author: "Sean P. Kane", ProductID: "123"},
			},
		},
		DurationMs: 1000,
	}
	if err := manager.AddEntry(entry1); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Add a question entry
	entry2 := ResearchEntry{
		Type:     "question",
		Query:    "How to optimize React performance?",
		ToolName: "ask_question",
		ResultSummary: ResultSummary{
			AnswerPreview: "React performance can be optimized by...",
			SourcesCount:  5,
			FollowupCount: 3,
		},
		DurationMs: 2000,
	}
	if err := manager.AddEntry(entry2); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}

	// Save
	if err := manager.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Test GetRecent
	recent := manager.GetRecent(10)
	if len(recent) != 2 {
		t.Errorf("expected 2 recent entries, got %d", len(recent))
	}

	// Test SearchByKeyword
	dockerResults := manager.SearchByKeyword("docker")
	if len(dockerResults) != 1 {
		t.Errorf("expected 1 result for 'docker', got %d", len(dockerResults))
	}

	reactResults := manager.SearchByKeyword("react")
	if len(reactResults) != 1 {
		t.Errorf("expected 1 result for 'react', got %d", len(reactResults))
	}

	// Test SearchByType
	searchResults := manager.SearchByType("search")
	if len(searchResults) != 1 {
		t.Errorf("expected 1 search result, got %d", len(searchResults))
	}

	questionResults := manager.SearchByType("question")
	if len(questionResults) != 1 {
		t.Errorf("expected 1 question result, got %d", len(questionResults))
	}

	// Test GetByID
	if len(recent) > 0 {
		entry := manager.GetByID(recent[0].ID)
		if entry == nil {
			t.Error("expected to find entry by ID")
		}
	}
}

func TestResearchHistoryManager_Prune(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "research-history.json")

	// Create manager with max 5 entries
	manager := NewResearchHistoryManager(filePath, 5)
	if err := manager.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Add 10 entries
	for i := 0; i < 10; i++ {
		entry := ResearchEntry{
			Type:       "search",
			Query:      "Query " + string(rune('A'+i)),
			ToolName:   "search_content",
			DurationMs: int64(i * 100),
		}
		if err := manager.AddEntry(entry); err != nil {
			t.Fatalf("failed to add entry %d: %v", i, err)
		}
	}

	// Check that only 5 entries remain
	recent := manager.GetRecent(100)
	if len(recent) != 5 {
		t.Errorf("expected 5 entries after pruning, got %d", len(recent))
	}
}

func TestResearchHistoryManager_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "research-history.json")

	// Create and add entries
	manager1 := NewResearchHistoryManager(filePath, 100)
	if err := manager1.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	entry := ResearchEntry{
		Type:       "search",
		Query:      "Kubernetes",
		ToolName:   "search_content",
		Timestamp:  time.Now(),
		DurationMs: 1500,
	}
	if err := manager1.AddEntry(entry); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	if err := manager1.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Create new manager and load from file
	manager2 := NewResearchHistoryManager(filePath, 100)
	if err := manager2.Load(); err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	// Verify entry was persisted
	recent := manager2.GetRecent(10)
	if len(recent) != 1 {
		t.Errorf("expected 1 entry after reload, got %d", len(recent))
	}
	if len(recent) > 0 && recent[0].Query != "Kubernetes" {
		t.Errorf("expected query 'Kubernetes', got %q", recent[0].Query)
	}

	// Verify file was created with correct permissions
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected file permissions 0600, got %o", perm)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == id2 {
		t.Error("expected unique IDs")
	}

	if len(id1) < 10 {
		t.Error("expected ID to be at least 10 characters")
	}

	if id1[:4] != "req_" {
		t.Errorf("expected ID to start with 'req_', got %q", id1[:4])
	}
}

func TestResearchEntry_FullResponse(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "research-history.json")

	manager := NewResearchHistoryManager(filePath, 100)
	if err := manager.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Create full response data
	fullResponse := []map[string]any{
		{"id": "123", "title": "Docker: Up & Running", "authors": []string{"Sean P. Kane"}},
		{"id": "456", "title": "Kubernetes Patterns", "authors": []string{"Bilgin Ibryam"}},
	}

	// Add entry with full response
	entry := ResearchEntry{
		ID:       "req_test123",
		Type:     "search",
		Query:    "Docker containers",
		ToolName: "search_content",
		ResultSummary: ResultSummary{
			Count: 2,
			TopResults: []TopResultSummary{
				{Title: "Docker: Up & Running", Author: "Sean P. Kane", ProductID: "123"},
			},
		},
		FullResponse: fullResponse,
		DurationMs:   1000,
	}

	if err := manager.AddEntry(entry); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Retrieve and verify
	retrieved := manager.GetByID("req_test123")
	if retrieved == nil {
		t.Fatal("expected to find entry by ID")
	}

	// Check full response is preserved
	if retrieved.FullResponse == nil {
		t.Error("expected FullResponse to be preserved")
	}

	// Verify full response content
	fullResp, ok := retrieved.FullResponse.([]map[string]any)
	if !ok {
		t.Errorf("expected FullResponse to be []map[string]any, got %T", retrieved.FullResponse)
	} else if len(fullResp) != 2 {
		t.Errorf("expected 2 items in FullResponse, got %d", len(fullResp))
	}
}

func TestResearchEntry_FullResponsePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "research-history.json")

	// Create and add entry with full response
	manager1 := NewResearchHistoryManager(filePath, 100)
	if err := manager1.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	fullResponse := []map[string]any{
		{"id": "789", "title": "Test Book"},
	}

	entry := ResearchEntry{
		ID:           "req_persist",
		Type:         "search",
		Query:        "Test query",
		ToolName:     "search_content",
		FullResponse: fullResponse,
		DurationMs:   500,
	}

	if err := manager1.AddEntry(entry); err != nil {
		t.Fatalf("failed to add entry: %v", err)
	}
	if err := manager1.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load from file with new manager
	manager2 := NewResearchHistoryManager(filePath, 100)
	if err := manager2.Load(); err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	// Verify full response was persisted
	retrieved := manager2.GetByID("req_persist")
	if retrieved == nil {
		t.Fatal("expected to find entry by ID after reload")
	}
	if retrieved.FullResponse == nil {
		t.Error("expected FullResponse to be persisted")
	}
}
