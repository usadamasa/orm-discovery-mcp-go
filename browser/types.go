package browser

import (
	"context"
	"net/http"
)

// O'Reilly API endpoints
const (
	APIEndpointBase = "https://learning.oreilly.com"

	BookAPIV1   = "/api/v1/book/%s/"
	SearchAPIV2 = "/api/v2/search/"
)

// SearchAPIResponse represents O'Reilly search API response structure
type SearchAPIResponse struct {
	Data    *SearchDataContainer `json:"data,omitempty"`
	Results []RawSearchResult    `json:"results,omitempty"`
	Items   []RawSearchResult    `json:"items,omitempty"`
	Hits    []RawSearchResult    `json:"hits,omitempty"`
}

type SearchDataContainer struct {
	Products []RawSearchResult `json:"products,omitempty"`
}

type RawSearchResult struct {
	ID           string   `json:"id,omitempty"`
	ProductID    string   `json:"product_id,omitempty"`
	Title        string   `json:"title,omitempty"`
	Name         string   `json:"name,omitempty"`
	DisplayTitle string   `json:"display_title,omitempty"`
	ProductName  string   `json:"product_name,omitempty"`
	Authors      []string `json:"authors,omitempty"`
	Author       Author   `json:"author,omitempty"`
	Creators     []struct {
		Name string `json:"name,omitempty"`
	} `json:"creators,omitempty"`
	AuthorNames            []string `json:"author_names,omitempty"`
	ContentType            string   `json:"content_type,omitempty"`
	Type                   string   `json:"type,omitempty"`
	Format                 string   `json:"format,omitempty"`
	ProductType            string   `json:"product_type,omitempty"`
	Description            string   `json:"description,omitempty"`
	Summary                string   `json:"summary,omitempty"`
	Excerpt                string   `json:"excerpt,omitempty"`
	DescriptionWithMarkups string   `json:"description_with_markups,omitempty"`
	ShortDescription       string   `json:"short_description,omitempty"`
	WebURL                 string   `json:"web_url,omitempty"`
	URL                    string   `json:"url,omitempty"`
	LearningURL            string   `json:"learning_url,omitempty"`
	Link                   string   `json:"link,omitempty"`
	OURN                   string   `json:"ourn,omitempty"`
	ISBN                   string   `json:"isbn,omitempty"`
	Publisher              string   `json:"publisher,omitempty"`
	Publishers             []string `json:"publishers,omitempty"`
	Imprint                string   `json:"imprint,omitempty"`
	PublisherName          string   `json:"publisher_name,omitempty"`
	PublishedDate          string   `json:"published_date,omitempty"`
	PublicationDate        string   `json:"publication_date,omitempty"`
	DatePublished          string   `json:"date_published,omitempty"`
	PubDate                string   `json:"pub_date,omitempty"`
}

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

// ExtractTableOfContentsParams represents parameters for extracting table of contents
type ExtractTableOfContentsParams struct {
	URL           string `json:"url"`
	IncludeHref   bool   `json:"include_href,omitempty"`
	MaxDepth      int    `json:"max_depth,omitempty"`
	IncludeParent bool   `json:"include_parent,omitempty"`
}

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

// BookSearchResult represents a book found through search
type BookSearchResult struct {
	ProductID   string   `json:"product_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Authors     []Author `json:"authors"`
	Publisher   string   `json:"publisher"`
}

// BookOverviewAndTOCResponse combines book details and table of contents
type BookOverviewAndTOCResponse struct {
	BookDetail      BookDetailResponse      `json:"book_detail"`
	TableOfContents TableOfContentsResponse `json:"table_of_contents"`
}
