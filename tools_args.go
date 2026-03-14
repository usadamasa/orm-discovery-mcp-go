package main

import "github.com/usadamasa/orm-discovery-mcp-go/browser"

// ResponseFormat defines the output format for tool results.
type ResponseFormat string

const (
	// ResponseFormatJSON returns structured JSON (default).
	ResponseFormatJSON ResponseFormat = "json"
	// ResponseFormatMarkdown returns human-readable Markdown.
	ResponseFormatMarkdown ResponseFormat = "markdown"
)

// Input validation constants.
const (
	maxQueryLength    = 500
	maxQuestionLength = 500
	maxRows           = 100
)

// SearchContentArgs represents the parameters for the oreilly_search_content tool.
type SearchContentArgs struct {
	Query        string   `json:"query" jsonschema:"2-5 focused keywords for specific technologies or frameworks. Avoid full sentences.,minLength=1,maxLength=500"`
	Rows         int      `json:"rows,omitempty" jsonschema:"Number of results per page (default: 25, max: 100),minimum=1,maximum=100"`
	Languages    []string `json:"languages,omitempty" jsonschema:"Languages to search in (default: en and ja)"`
	TzOffset     int      `json:"tzOffset,omitempty" jsonschema:"Timezone offset (default: -9 for JST)"`
	AiaOnly      bool     `json:"aia_only,omitempty" jsonschema:"Search only AI-assisted content (default: false)"`
	FeatureFlags string   `json:"feature_flags,omitempty" jsonschema:"Feature flags (default: improveSearchFilters)"`
	Report       bool     `json:"report,omitempty" jsonschema:"Include reporting data (default: true)"`
	IsTopics     bool     `json:"isTopics,omitempty" jsonschema:"Search topics only (default: false)"`

	// Pagination parameters
	Offset int `json:"offset,omitempty" jsonschema:"Pagination offset (0-based, default: 0)"`

	// Response format
	Format ResponseFormat `json:"format,omitempty" jsonschema:"Output format: 'json' (default) or 'markdown' for human-readable output"`
}

// AskQuestionArgs represents the parameters for the oreilly_ask_question tool.
type AskQuestionArgs struct {
	Question           string         `json:"question" jsonschema:"Focused technical question in English (under 100 characters preferred),minLength=1,maxLength=500"`
	MaxWaitTimeSeconds int            `json:"max_wait_time_seconds,omitempty" jsonschema:"Maximum time to wait for answer generation in seconds (default: 300, max: 600)"`
	Format             ResponseFormat `json:"format,omitempty" jsonschema:"Output format: 'json' (default) or 'markdown' for human-readable output"`
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

	HistoryID string `json:"history_id,omitempty"` // Research history ID
	FilePath  string `json:"file_path,omitempty"`  // Path to cached Markdown file with full results
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

// ReauthResult represents the structured output for the oreilly_reauthenticate tool.
type ReauthResult struct {
	Status  string `json:"status"`  // "authenticated" | "setup_completed"
	Message string `json:"message"` // Human-readable description
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
