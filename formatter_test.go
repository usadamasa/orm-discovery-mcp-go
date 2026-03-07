package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

func TestFormatSearchResultsMarkdown(t *testing.T) {
	t.Run("normal results", func(t *testing.T) {
		result := &SearchContentResult{
			Count:        2,
			Total:        2,
			TotalResults: 50,
			Mode:         SearchModeBFS,
			HistoryID:    "req_abc123",
			Results: []map[string]any{
				{"id": "123", "title": "Docker: Up & Running", "authors": []any{"Sean P. Kane"}},
				{"id": "456", "title": "Kubernetes in Action", "authors": []any{"Marko Lukša"}},
			},
		}

		md := formatSearchResultsMarkdown(result)

		assert.Contains(t, md, "Search Results")
		assert.Contains(t, md, "Docker: Up & Running")
		assert.Contains(t, md, "Kubernetes in Action")
		assert.Contains(t, md, "2 of 50")
		assert.Contains(t, md, "req_abc123")
	})

	t.Run("empty results", func(t *testing.T) {
		result := &SearchContentResult{
			Count:        0,
			Total:        0,
			TotalResults: 0,
			Mode:         SearchModeBFS,
			Results:      []map[string]any{},
		}

		md := formatSearchResultsMarkdown(result)

		assert.Contains(t, md, "No results found")
	})

	t.Run("has more results", func(t *testing.T) {
		result := &SearchContentResult{
			Count:        25,
			Total:        25,
			TotalResults: 100,
			HasMore:      true,
			NextOffset:   25,
			Mode:         SearchModeBFS,
			Results: []map[string]any{
				{"id": "123", "title": "Test Book"},
			},
		}

		md := formatSearchResultsMarkdown(result)

		assert.Contains(t, md, "More results available")
		assert.Contains(t, md, "offset=25")
	})
}

func TestFormatSearchResultsMarkdown_BrowserAuthor(t *testing.T) {
	// Bug #132: authors が []browser.Author 型のとき、Markdown に著者名が表示されること
	result := &SearchContentResult{
		Count:        1,
		Total:        1,
		TotalResults: 1,
		Mode:         SearchModeBFS,
		Results: []map[string]any{
			{
				"id":    "123",
				"title": "Go Programming",
				"authors": []browser.Author{
					{Name: "John Doe"},
					{Name: "Jane Smith"},
				},
			},
		},
	}

	md := formatSearchResultsMarkdown(result)

	assert.Contains(t, md, "Go Programming")
	assert.Contains(t, md, "John Doe", "[]browser.Author should be rendered in Markdown")
	assert.Contains(t, md, "Jane Smith", "[]browser.Author should be rendered in Markdown")
}

func TestFormatSearchResultsMarkdown_StringAuthors(t *testing.T) {
	// Bug #132: authors が []string 型のとき、Markdown に著者名が表示されること
	result := &SearchContentResult{
		Count:        1,
		Total:        1,
		TotalResults: 1,
		Mode:         SearchModeBFS,
		Results: []map[string]any{
			{
				"id":      "456",
				"title":   "Rust Programming",
				"authors": []string{"Alice", "Bob"},
			},
		},
	}

	md := formatSearchResultsMarkdown(result)

	assert.Contains(t, md, "Rust Programming")
	assert.Contains(t, md, "Alice", "[]string authors should be rendered in Markdown")
	assert.Contains(t, md, "Bob", "[]string authors should be rendered in Markdown")
}

func TestFormatAskQuestionMarkdown(t *testing.T) {
	t.Run("normal answer with sources", func(t *testing.T) {
		result := &AskQuestionResult{
			QuestionID: "q123",
			Question:   "How to use Docker?",
			Answer:     "Docker is a containerization platform...",
			IsFinished: true,
			Sources: []browser.AnswerSource{
				{Title: "Docker Deep Dive", URL: "https://example.com/docker"},
			},
			FollowupQuestions: []string{"What is Docker Compose?"},
		}

		md := formatAskQuestionMarkdown(result)

		assert.Contains(t, md, "How to use Docker?")
		assert.Contains(t, md, "Docker is a containerization platform")
		assert.Contains(t, md, "Docker Deep Dive")
		assert.Contains(t, md, "What is Docker Compose?")
	})

	t.Run("answer without sources", func(t *testing.T) {
		result := &AskQuestionResult{
			QuestionID: "q456",
			Question:   "Simple question",
			Answer:     "Simple answer",
			IsFinished: true,
		}

		md := formatAskQuestionMarkdown(result)

		assert.Contains(t, md, "Simple question")
		assert.Contains(t, md, "Simple answer")
		assert.NotContains(t, md, "Sources")
	})
}
