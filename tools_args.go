package main

import "github.com/usadamasa/orm-discovery-mcp-go/browser"

// SearchMode defines the exploration mode for oreilly_search_content tool.
type SearchMode string

const (
	// SearchModeBFS is breadth-first search mode that returns lightweight results (id, title, authors only).
	// This is the default mode for reduced context consumption.
	SearchModeBFS SearchMode = "bfs"
	// SearchModeDFS is depth-first search mode that returns full detailed results.
	// Use this when you need complete information and are willing to consume more context.
	SearchModeDFS SearchMode = "dfs"
)

// SearchContentArgs represents the parameters for the oreilly_search_content tool.
type SearchContentArgs struct {
	Query        string   `json:"query" jsonschema:"2-5 focused keywords for specific technologies or frameworks. Avoid full sentences."`
	Rows         int      `json:"rows,omitempty" jsonschema:"Number of results per page (default: 25, max: 100)"`
	Languages    []string `json:"languages,omitempty" jsonschema:"Languages to search in (default: en and ja)"`
	TzOffset     int      `json:"tzOffset,omitempty" jsonschema:"Timezone offset (default: -9 for JST)"`
	AiaOnly      bool     `json:"aia_only,omitempty" jsonschema:"Search only AI-assisted content (default: false)"`
	FeatureFlags string   `json:"feature_flags,omitempty" jsonschema:"Feature flags (default: improveSearchFilters)"`
	Report       bool     `json:"report,omitempty" jsonschema:"Include reporting data (default: true)"`
	IsTopics     bool     `json:"isTopics,omitempty" jsonschema:"Search topics only (default: false)"`

	// Pagination parameters
	Offset int `json:"offset,omitempty" jsonschema:"Pagination offset (0-based, default: 0)"`

	// Exploration mode parameters
	Mode      SearchMode `json:"mode,omitempty" jsonschema:"Exploration mode: 'bfs' (default) returns lightweight results (id, title, authors), 'dfs' returns full detailed results"`
	Summarize bool       `json:"summarize,omitempty" jsonschema:"In DFS mode, use MCP Sampling to generate a summary of results (reduces context consumption)"`
}

// AskQuestionArgs represents the parameters for the oreilly_ask_question tool.
type AskQuestionArgs struct {
	Question           string `json:"question" jsonschema:"Focused technical question in English (under 100 characters preferred)"`
	MaxWaitTimeSeconds int    `json:"max_wait_time_seconds,omitempty" jsonschema:"Maximum time to wait for answer generation in seconds (default: 300, max: 600)"`
}

// SearchContentResult represents the structured output for oreilly_search_content tool.
type SearchContentResult struct {
	Count   int              `json:"count"`
	Total   int              `json:"total"`
	Results []map[string]any `json:"results"`

	// Pagination fields
	TotalResults int  `json:"total_results"`         // Total number of matching results from API
	HasMore      bool `json:"has_more"`              // Whether more results are available
	NextOffset   int  `json:"next_offset,omitempty"` // Next page offset (only set when HasMore=true)

	// BFS/DFS mode specific fields
	Mode      SearchMode `json:"mode,omitempty"`       // The mode used for this search
	HistoryID string     `json:"history_id,omitempty"` // Research history ID for accessing full data later
	Summary   string     `json:"summary,omitempty"`    // AI-generated summary (DFS mode with Summarize=true)
	Note      string     `json:"note,omitempty"`       // Helpful note for the user (BFS mode)
}

// calcPagination computes pagination state from offset, result count, and total results.
func calcPagination(offset, resultCount, totalResults int) (hasMore bool, nextOffset int) {
	if totalResults <= 0 {
		return false, 0
	}
	if (offset + resultCount) < totalResults {
		return true, offset + resultCount
	}
	return false, 0
}

// BFSResult represents a lightweight search result for BFS mode.
type BFSResult struct {
	ID      string   `json:"id"`                // product_id or ISBN
	Title   string   `json:"title"`             // Book/content title
	Authors []string `json:"authors,omitempty"` // Author names
}

// AskQuestionResult represents the structured output for oreilly_ask_question tool.
type AskQuestionResult struct {
	QuestionID          string                       `json:"question_id"`
	Question            string                       `json:"question"`
	Answer              string                       `json:"answer"`
	IsFinished          bool                         `json:"is_finished"`
	Sources             []browser.AnswerSource       `json:"sources"`
	RelatedResources    []browser.RelatedResource    `json:"related_resources"`
	AffiliationProducts []browser.AffiliationProduct `json:"affiliation_products"`
	FollowupQuestions   []string                     `json:"followup_questions"`
	CitationNote        string                       `json:"citation_note"`
}
