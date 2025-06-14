package browser

import (
	"context"

	"net/http"
)

// APIEndpointBase O'Reilly API endpoints
const (
	APIEndpointBase = "https://learning.oreilly.com"
)

// BrowserClient はヘッドレスブラウザを使用したO'Reillyクライアントです
type BrowserClient struct {
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
	cookies    []*http.Cookie
	userAgent  string
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
