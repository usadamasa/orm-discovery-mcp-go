package browser

import (
	"context"
	"net/http"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// APIEndpointBase O'Reilly API endpoints
const (
	APIEndpointBase = "https://learning.oreilly.com"
)

// BrowserClient はヘッドレスブラウザを使用したO'Reillyクライアントです
type BrowserClient struct {
	ctx           context.Context
	cancel        context.CancelFunc
	httpClient    *http.Client
	cookieJar     http.CookieJar
	cookies       []*http.Cookie
	userAgent     string
	cookieManager cookie.Manager
	debug         bool
	tmpDir        string
}

// TableOfContentsItem represents a single item in the table of contents
type TableOfContentsItem struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Href     string                 `json:"href"`
	Level    int                    `json:"level"`
	Parent   string                 `json:"parent,omitempty"`
	Children []TableOfContentsItem  `json:"children,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TableOfContentsResponse represents the complete table of contents response
type TableOfContentsResponse struct {
	BookID          string                 `json:"book_id"`
	BookTitle       string                 `json:"book_title"`
	TableOfContents []TableOfContentsItem  `json:"table_of_contents"`
	TotalChapters   int                    `json:"total_chapters"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Author is used locally for normalization
type Author struct {
	Name string `json:"name"`
}

type Publisher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Topics struct {
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Score          float64 `json:"score"`
	UUID           string  `json:"uuid"`
	EpubIdentifier string  `json:"epub_identifier,omitempty"`
}

// BookDetailResponse represents comprehensive book metadata from O'Reilly API
type BookDetailResponse struct {
	ID            string                 `json:"id"`
	URL           string                 `json:"url"`
	WebURL        string                 `json:"web_url"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Authors       []Author               `json:"authors"`
	Publishers    []Publisher            `json:"publishers"`
	ISBN          string                 `json:"isbn"`
	VirtualPages  int                    `json:"virtual_pages"`
	AverageRating float64                `json:"average_rating"`
	Cover         string                 `json:"cover"`
	Issued        string                 `json:"issued"`
	Topics        []Topics               `json:"topics"`
	Language      string                 `json:"language"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ChapterContentResponse represents structured chapter content with parsed HTML
type ChapterContentResponse struct {
	BookID       string                 `json:"book_id"`
	ChapterName  string                 `json:"chapter_name"`
	ChapterTitle string                 `json:"chapter_title"`
	Content      ParsedChapterContent   `json:"content"`
	SourceURL    string                 `json:"source_url"`
	Metadata     map[string]interface{} `json:"metadata"`
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
	Content []interface{}  `json:"content"` // Can contain strings, CodeBlocks, or ImageReferences
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
