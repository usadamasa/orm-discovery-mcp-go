package main

import (
	"context"
	"fmt"
	"log"
)

// OreillyClient はO'Reilly Learning Platform APIのクライアントです
type OreillyClient struct {
	browserClient *BrowserClient
}

// NewOreillyClient はブラウザクライアントを使用してO'Reillyクライアントを作成します
func NewOreillyClient(userID, password string) (*OreillyClient, error) {
	// ブラウザクライアントを作成してログイン
	browserClient, err := NewBrowserClient(userID, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser client: %w", err)
	}

	client := &OreillyClient{
		browserClient: browserClient,
	}

	log.Printf("ブラウザクライアントを使用したO'Reillyクライアントを作成しました")
	return client, nil
}

// Close はクライアントをクリーンアップします
func (c *OreillyClient) Close() {
	if c.browserClient != nil {
		c.browserClient.Close()
	}
}


// SearchParams は検索パラメータの構造体です
type SearchParams struct {
	Query       string   `json:"q"`
	Rows        int      `json:"rows,omitempty"`
	Languages   []string `json:"language,omitempty"`
	TzOffset    int      `json:"tzOffset,omitempty"`
	AiaOnly     bool     `json:"aia_only,omitempty"`
	FeatureFlags string  `json:"feature_flags,omitempty"`
	Report      bool     `json:"report,omitempty"`
	IsTopics    bool     `json:"isTopics,omitempty"`
}

// SearchResponse は検索レスポンスの構造体です
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
	Total   int            `json:"total"`
}

// SearchResult は検索結果の1件を表します
type SearchResult struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	WebURL      string                 `json:"web_url"`
	Type        string                 `json:"content_type"`
	Authors     []string               `json:"authors"`
	Publishers  []string               `json:"publishers"`
	Topics      []string               `json:"topics"`
	Language    string                 `json:"language"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Search はO'Reilly Learning Platformで検索を実行します（ブラウザベース）
func (c *OreillyClient) Search(ctx context.Context, params SearchParams) (*SearchResponse, error) {
	log.Printf("O'Reillyブラウザクライアントで検索が要求されました: %s\n", params.Query)
	if params.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// ブラウザクライアントが利用可能かチェック
	if c.browserClient == nil {
		return nil, fmt.Errorf("browser client is not available")
	}

	// デフォルト値の設定
	if params.Rows <= 0 {
		params.Rows = 100
	}
	if len(params.Languages) == 0 {
		params.Languages = []string{"en", "ja"}
	}

	// ブラウザクライアント用のオプションを準備
	options := map[string]interface{}{
		"rows":      params.Rows,
		"languages": params.Languages,
	}

	// JavaScript API検索を実行（ブラウザコンテキスト内で実行され、最も成功率が高い）
	results, err := c.browserClient.SearchContentAPI(params.Query, options)
	if err != nil {
		return nil, fmt.Errorf("API search failed: %w", err)
	}

	// ブラウザクライアントの結果をAPIレスポンス形式に変換
	searchResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		searchResult := SearchResult{
			ID:          getStringValue(result, "id"),
			Title:       getStringValue(result, "title"),
			Description: getStringValue(result, "description"),
			URL:         getStringValue(result, "url"),
			WebURL:      getStringValue(result, "url"),
			Type:        getStringValue(result, "content_type"),
			Language:    "unknown", // ブラウザからは言語情報を取得できない場合が多い
			Metadata:    result,
		}

		// 著者情報の変換
		if authors, ok := result["authors"].([]string); ok {
			searchResult.Authors = authors
		} else if author, ok := result["author"].(string); ok && author != "" {
			searchResult.Authors = []string{author}
		}

		// 出版社情報の変換
		if publisher, ok := result["publisher"].(string); ok && publisher != "" {
			searchResult.Publishers = []string{publisher}
		}

		searchResults = append(searchResults, searchResult)
	}

	// レスポンスを構築
	searchResp := &SearchResponse{
		Results: searchResults,
		Count:   len(searchResults),
		Total:   len(searchResults),
	}

	log.Printf("O'Reillyブラウザクライアントからの検索結果: %d件", len(searchResp.Results))
	return searchResp, nil
}

// getStringValue はmap[string]interface{}から文字列値を安全に取得するヘルパー関数
func getStringValue(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

type Collection struct {
	ID                    string           `json:"id"`
	Ourn                  string           `json:"ourn"`
	Name                  string           `json:"name"`
	Description           string           `json:"description"`
	IsDefault             bool             `json:"is_default"`
	Content               []CollectionItem `json:"content"`
	LastModifiedTime      string           `json:"last_modified_time"`
	CreatedTime           string           `json:"created_time"`
	CoverImage            string           `json:"cover_image"`
	FollowerCount         int              `json:"follower_count"`
	Sharing               string           `json:"sharing"`
	IsOwned               bool             `json:"is_owned"`
	IsFollowing           bool             `json:"is_following"`
	OwnerDisplayName      string           `json:"owner_display_name"`
	SharingOptions        []string         `json:"sharing_options"`
	WebURL                string           `json:"web_url"`
	CanBeAssigned         bool             `json:"can_be_assigned"`
	Type                  string           `json:"type"`
	PrimaryAccount        string           `json:"primary_account"`
	PrimaryAccountDisplay string           `json:"primary_account_display_name"`
	Topics                []string         `json:"topics"`
	PublicationTime       *string          `json:"publication_time"`
	MarketingType         MarketingType    `json:"marketing_type"`
}

type CollectionItem struct {
	ID          string                 `json:"id"`
	APIURL      string                 `json:"api_url"`
	Metadata    map[string]interface{} `json:"metadata"`
	Ourn        string                 `json:"ourn"`
	DateAdded   string                 `json:"date_added"`
	ContentType string                 `json:"content_type"`
	Index       float64                `json:"index"`
	Title       *string                `json:"title"`
	Description *string                `json:"description"`
}

type MarketingType struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}


// ExtractTableOfContentsParams は目次抽出パラメータの構造体です
type ExtractTableOfContentsParams struct {
	URL string `json:"url"`
}

// TableOfContentsItem は目次の1項目を表します
type TableOfContentsItem struct {
	Level       int    `json:"level"`
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	ChapterID   string `json:"chapter_id,omitempty"`
	SectionID   string `json:"section_id,omitempty"`
	PageNumber  string `json:"page_number,omitempty"`
}

// TableOfContentsResponse は目次抽出レスポンスの構造体です
type TableOfContentsResponse struct {
	BookTitle    string                `json:"book_title"`
	BookID       string                `json:"book_id"`
	BookURL      string                `json:"book_url"`
	Authors      []string              `json:"authors"`
	Publisher    string                `json:"publisher"`
	TableOfContents []TableOfContentsItem `json:"table_of_contents"`
	ExtractedAt  string                `json:"extracted_at"`
}

// ExtractTableOfContents はO'Reilly書籍の目次を抽出します
func (c *OreillyClient) ExtractTableOfContents(ctx context.Context, params ExtractTableOfContentsParams) (*TableOfContentsResponse, error) {
	log.Printf("O'Reilly書籍の目次抽出が要求されました: %s\n", params.URL)
	
	if params.URL == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}
	
	// ブラウザクライアントが利用可能かチェック
	if c.browserClient == nil {
		return nil, fmt.Errorf("browser client is not available")
	}
	
	// ブラウザクライアントで目次を抽出
	result, err := c.browserClient.ExtractTableOfContents(params.URL)
	if err != nil {
		return nil, fmt.Errorf("table of contents extraction failed: %w", err)
	}
	
	log.Printf("目次抽出が完了しました: %s", result.BookTitle)
	return result, nil
}
