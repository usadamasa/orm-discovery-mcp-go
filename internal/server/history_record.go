package server

import (
	"log/slog"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/history"
)

// saveHistoryEntry adds a history entry and persists it.
func (s *Server) saveHistoryEntry(entry history.Entry) {
	if s.historyManager == nil {
		return
	}
	if err := s.historyManager.AddEntry(entry); err != nil {
		slog.Warn("調査履歴の追加に失敗しました", "error", err)
		return
	}
	if err := s.historyManager.Save(); err != nil {
		slog.Warn("調査履歴の保存に失敗しました", "error", err)
	}
}

// recordSearchHistory records a search to the research history.
// If entryID is provided, it is used as the history entry ID (to match the cache file).
func (s *Server) recordSearchHistory(query string, options map[string]any, results []map[string]any, filePath string, duration time.Duration, entryID string) {
	topResults := make([]history.TopResultSummary, 0, 5)
	for i, result := range results {
		if i >= 5 {
			break
		}
		summary := history.TopResultSummary{}
		if title, ok := result["title"].(string); ok {
			summary.Title = title
		}
		if authors, ok := result["authors"].([]any); ok && len(authors) > 0 {
			if author, ok := authors[0].(string); ok {
				summary.Author = author
			}
		}
		if productID, ok := result["product_id"].(string); ok {
			summary.ProductID = productID
		} else if isbn, ok := result["isbn"].(string); ok {
			summary.ProductID = isbn
		}
		topResults = append(topResults, summary)
	}

	s.saveHistoryEntry(history.Entry{
		ID:         entryID,
		Type:       history.EntryTypeSearch,
		Query:      query,
		ToolName:   "oreilly_search_content",
		Parameters: options,
		ResultSummary: history.ResultSummary{
			Count:      len(results),
			TopResults: topResults,
		},
		DurationMs: duration.Milliseconds(),
		FilePath:   filePath,
	})
}

// recordQuestionHistory records a question to the research history.
func (s *Server) recordQuestionHistory(question string, answer *browser.AnswerResponse, duration time.Duration) {
	answerPreview := answer.MisoResponse.Data.Answer
	if len(answerPreview) > 200 {
		answerPreview = answerPreview[:200] + "..."
	}

	s.saveHistoryEntry(history.Entry{
		Type:     history.EntryTypeQuestion,
		Query:    question,
		ToolName: "oreilly_ask_question",
		ResultSummary: history.ResultSummary{
			AnswerPreview: answerPreview,
			SourcesCount:  len(answer.MisoResponse.Data.Sources),
			FollowupCount: len(answer.MisoResponse.Data.FollowupQuestions),
		},
		DurationMs: duration.Milliseconds(),
	})
}
