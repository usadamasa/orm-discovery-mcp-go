package main

import (
	"testing"
)

func TestSearchMode_Constants(t *testing.T) {
	// Verify mode constants are defined correctly
	if SearchModeBFS != "bfs" {
		t.Errorf("expected SearchModeBFS to be 'bfs', got %q", SearchModeBFS)
	}
	if SearchModeDFS != "dfs" {
		t.Errorf("expected SearchModeDFS to be 'dfs', got %q", SearchModeDFS)
	}
}

func TestSearchMode_DefaultMode(t *testing.T) {
	args := SearchContentArgs{
		Query: "Docker",
	}

	// Mode should be empty by default
	if args.Mode != "" {
		t.Errorf("expected default Mode to be empty, got %q", args.Mode)
	}

	// Summarize should be false by default
	if args.Summarize {
		t.Error("expected default Summarize to be false")
	}
}

func TestBFSResult_Structure(t *testing.T) {
	result := BFSResult{
		ID:      "123",
		Title:   "Docker: Up & Running",
		Authors: []string{"Sean P. Kane"},
	}

	if result.ID != "123" {
		t.Errorf("expected ID '123', got %q", result.ID)
	}
	if result.Title != "Docker: Up & Running" {
		t.Errorf("expected Title 'Docker: Up & Running', got %q", result.Title)
	}
	if len(result.Authors) != 1 || result.Authors[0] != "Sean P. Kane" {
		t.Errorf("expected Authors ['Sean P. Kane'], got %v", result.Authors)
	}
}

func TestSearchContentResult_BFSFields(t *testing.T) {
	result := SearchContentResult{
		Count:     10,
		Total:     100,
		Mode:      SearchModeBFS,
		HistoryID: "req_abc123",
		Note:      "Use oreilly://book-details/{id} for full details.",
	}

	if result.Mode != SearchModeBFS {
		t.Errorf("expected Mode to be BFS, got %q", result.Mode)
	}
	if result.HistoryID != "req_abc123" {
		t.Errorf("expected HistoryID 'req_abc123', got %q", result.HistoryID)
	}
	if result.Note == "" {
		t.Error("expected Note to be set for BFS mode")
	}
}

func TestSearchContentResult_DFSFields(t *testing.T) {
	result := SearchContentResult{
		Count:     10,
		Total:     100,
		Mode:      SearchModeDFS,
		HistoryID: "req_xyz789",
		Summary:   "Docker books summary...",
	}

	if result.Mode != SearchModeDFS {
		t.Errorf("expected Mode to be DFS, got %q", result.Mode)
	}
	if result.Summary != "Docker books summary..." {
		t.Errorf("expected Summary to be set for DFS mode, got %q", result.Summary)
	}
}
