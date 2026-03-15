package history

import "time"

// History は調査履歴全体を保持する構造体
type History struct {
	Version     int       `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	Entries     []Entry   `json:"entries"`
	Index       Index     `json:"index"`
}

// Index は検索用インデックス
type Index struct {
	ByKeyword map[string][]string `json:"by_keyword"`
	ByType    map[string][]string `json:"by_type"`
	ByDate    map[string][]string `json:"by_date"`
}

// Entry は個々の調査エントリ
type Entry struct {
	ID            string         `json:"id"`
	Timestamp     time.Time      `json:"timestamp"`
	Type          string         `json:"type"` // "search" or "question"
	Query         string         `json:"query"`
	Keywords      []string       `json:"keywords"`
	ToolName      string         `json:"tool_name"`
	Parameters    map[string]any `json:"parameters,omitempty"`
	ResultSummary ResultSummary  `json:"result_summary"`
	DurationMs    int64          `json:"duration_ms"`

	// FilePath stores the path to the cached Markdown file with full results.
	FilePath string `json:"file_path,omitempty"`
}

// ResultSummary は結果のサマリー（タイプ別に異なる構造）
type ResultSummary struct {
	// search 用
	Count      int                `json:"count,omitempty"`
	TopResults []TopResultSummary `json:"top_results,omitempty"`

	// question 用
	AnswerPreview string `json:"answer_preview,omitempty"`
	SourcesCount  int    `json:"sources_count,omitempty"`
	FollowupCount int    `json:"followup_count,omitempty"`
}

// TopResultSummary は検索結果のトップ結果サマリー
type TopResultSummary struct {
	Title     string `json:"title"`
	Author    string `json:"author,omitempty"`
	ProductID string `json:"product_id,omitempty"`
}
