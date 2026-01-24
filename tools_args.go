package main

import "github.com/usadamasa/orm-discovery-mcp-go/browser"

// SearchContentArgs represents the parameters for the search_content tool.
type SearchContentArgs struct {
	Query        string   `json:"query" jsonschema:"2-5 focused keywords for specific technologies or frameworks. Avoid full sentences."`
	Rows         int      `json:"rows,omitempty" jsonschema:"Number of results to return (default: 100)"`
	Languages    []string `json:"languages,omitempty" jsonschema:"Languages to search in (default: en and ja)"`
	TzOffset     int      `json:"tzOffset,omitempty" jsonschema:"Timezone offset (default: -9 for JST)"`
	AiaOnly      bool     `json:"aia_only,omitempty" jsonschema:"Search only AI-assisted content (default: false)"`
	FeatureFlags string   `json:"feature_flags,omitempty" jsonschema:"Feature flags (default: improveSearchFilters)"`
	Report       bool     `json:"report,omitempty" jsonschema:"Include reporting data (default: true)"`
	IsTopics     bool     `json:"isTopics,omitempty" jsonschema:"Search topics only (default: false)"`
}

// AskQuestionArgs represents the parameters for the ask_question tool.
type AskQuestionArgs struct {
	Question           string `json:"question" jsonschema:"Focused technical question in English (under 100 characters preferred)"`
	MaxWaitTimeSeconds int    `json:"max_wait_time_seconds,omitempty" jsonschema:"Maximum time to wait for answer generation in seconds (default: 300, max: 600)"`
}

// SearchContentResult represents the structured output for search_content tool.
type SearchContentResult struct {
	Count   int                      `json:"count"`
	Total   int                      `json:"total"`
	Results []map[string]interface{} `json:"results"`
}

// AskQuestionResult represents the structured output for ask_question tool.
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
