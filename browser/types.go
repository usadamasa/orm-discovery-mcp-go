package browser

import (
	"net/http"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// Operation timeouts
const (
	AuthValidationTimeout = 15 * time.Second
	APIOperationTimeout   = 30 * time.Second

	// VisibleLoginTimeout はビジブルブラウザでの手動ログイン待機タイムアウト
	VisibleLoginTimeout = 5 * time.Minute
)

// APIEndpointBase O'Reilly API endpoints
const (
	APIEndpointBase = "https://learning.oreilly.com"
)

// HTTPDoer はHTTP通信を実行するインターフェース
// *http.Client はこのインターフェースを実装しています
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client は server.go が BrowserClient に期待するメソッドを定義するインターフェース。
// テスト時に mock に差し替えることで、全 O'Reilly ハンドラーの単体テストを可能にする。
type Client interface {
	SearchContent(query string, options map[string]any) ([]map[string]any, int, error)
	AskQuestion(question string, maxWaitTime time.Duration) (*AnswerResponse, error)
	GetBookDetails(productID string) (*BookDetailResponse, error)
	GetBookTOC(productID string) (*TableOfContentsResponse, error)
	GetBookChapterContent(productID, chapterName string) (*ChapterContentResponse, error)
	GetQuestionByID(questionID string) (*AnswerResponse, error)
	Reauthenticate() error
	CheckAndResetAuth() error
	Close()
}

// コンパイル時にインターフェース実装を検証
var _ Client = (*BrowserClient)(nil)

// BrowserClient はヘッドレスブラウザを使用したO'Reillyクライアントです
type BrowserClient struct {
	httpClient    HTTPDoer // HTTP通信を実行するインターフェース (*http.Clientが実装)
	userAgent     string
	cookieManager cookie.Manager
	debug         bool
	stateDir      string // XDG StateHome (Chrome一時データ用)
}

// TableOfContentsItem represents a single item in the table of contents
type TableOfContentsItem struct {
	ID       string                `json:"id"`
	Title    string                `json:"title"`
	Href     string                `json:"href"`
	Level    int                   `json:"level"`
	Parent   string                `json:"parent,omitempty"`
	Children []TableOfContentsItem `json:"children,omitempty"`
	Metadata map[string]any        `json:"metadata,omitempty"`
}

// TableOfContentsResponse represents the complete table of contents response
type TableOfContentsResponse struct {
	BookID          string                `json:"book_id"`
	BookTitle       string                `json:"book_title"`
	TableOfContents []TableOfContentsItem `json:"table_of_contents"`
	TotalChapters   int                   `json:"total_chapters"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
}

// Author is used for search result normalization
type Author struct {
	Name string `json:"name"`
}

// BookResource represents an external resource associated with a book
type BookResource struct {
	URL         string `json:"url"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// BookDetailResponse represents book metadata from O'Reilly v2 epubs API
type BookDetailResponse struct {
	OURN            string            `json:"ourn"`
	Identifier      string            `json:"identifier"`
	ISBN            string            `json:"isbn"`
	URL             string            `json:"url"`
	ContentFormat   string            `json:"content_format"`
	Title           string            `json:"title"`
	Descriptions    map[string]string `json:"descriptions"`
	PublicationDate string            `json:"publication_date"`
	VirtualPages    int               `json:"virtual_pages"`
	PageCount       int               `json:"page_count"`
	Language        string            `json:"language"`
	Resources       []BookResource    `json:"resources,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
}

// ChapterContentResponse represents structured chapter content with parsed HTML
type ChapterContentResponse struct {
	BookID       string               `json:"book_id"`
	ChapterName  string               `json:"chapter_name"`
	ChapterTitle string               `json:"chapter_title"`
	Content      ParsedChapterContent `json:"content"`
	SourceURL    string               `json:"source_url"`
	Metadata     map[string]any       `json:"metadata"`
}

// ParsedChapterContent represents structured content parsed from HTML
type ParsedChapterContent struct {
	Title      string           `json:"title"`
	Sections   []ContentSection `json:"sections"`
	Paragraphs []string         `json:"paragraphs"`
	Headings   []ContentHeading `json:"headings"`
	CodeBlocks []CodeBlock      `json:"code_blocks"`
	Images     []ImageReference `json:"images"`
	Links      []LinkReference  `json:"links"`
}

// ContentSection represents a section of content with heading and content
type ContentSection struct {
	Heading ContentHeading `json:"heading"`
	Content []any          `json:"content"` // Can contain strings, CodeBlocks, or ImageReferences
}

// ContentHeading represents a heading element
type ContentHeading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id,omitempty"`
}

// CodeBlock represents a code block
type CodeBlock struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code"`
	Caption  string `json:"caption,omitempty"`
}

// ImageReference represents an image reference
type ImageReference struct {
	Src     string `json:"src"`
	Alt     string `json:"alt,omitempty"`
	Caption string `json:"caption,omitempty"`
}

// LinkReference represents a link reference
type LinkReference struct {
	Href string `json:"href"`
	Text string `json:"text"`
	Type string `json:"type"` // "external", "internal", "anchor"
}

// Answer types for O'Reilly Answers API integration

// QuestionRequest represents a request to submit a question to O'Reilly Answers
type QuestionRequest struct {
	Question              string         `json:"question"`
	FilterQuery           string         `json:"fq"`
	SourceFields          []string       `json:"source_fl"`
	RelatedResourceFields []string       `json:"related_resource_fl"`
	PipelineConfig        PipelineConfig `json:"_pipeline_config"`
}

// PipelineConfig represents configuration for the answer generation pipeline
type PipelineConfig struct {
	SnippetLength   int `json:"snippet_length"`
	HighlightLength int `json:"highlight_length"`
}

// QuestionResponse represents the response from question submission
type QuestionResponse struct {
	QuestionID string `json:"question_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// AnswerResponse represents the response containing the answer to a submitted question
type AnswerResponse struct {
	QuestionID   string       `json:"question_id"`
	IsFinished   bool         `json:"is_finished"`
	MisoResponse MisoResponse `json:"miso_response"`
}

// MisoResponse represents the AI-generated response data
type MisoResponse struct {
	Data AnswerData `json:"data"`
}

// AnswerData represents the core answer data with content and references
type AnswerData struct {
	Answer              string               `json:"answer"`
	Sources             []AnswerSource       `json:"sources"`
	RelatedResources    []RelatedResource    `json:"related_resources"`
	AffiliationProducts []AffiliationProduct `json:"affiliation_products"`
	FollowupQuestions   []string             `json:"followup_questions"`
}

// AnswerSource represents a source document used to generate the answer
type AnswerSource struct {
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Authors    []string `json:"authors"`
	CoverImage string   `json:"cover_image"`
	Excerpt    string   `json:"excerpt"`
}

// RelatedResource represents a related resource for additional reading
type RelatedResource struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Authors     []string `json:"authors"`
	ContentType string   `json:"content_type"`
}

// AffiliationProduct represents an O'Reilly product related to the answer
type AffiliationProduct struct {
	ProductID   string   `json:"product_id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Authors     []string `json:"authors"`
	ContentType string   `json:"content_type"`
}
