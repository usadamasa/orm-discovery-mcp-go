package main

import (
	"context"
	"fmt"
	"log"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

// OreillyClient はO'Reilly Learning Platform APIのクライアントです
type OreillyClient struct {
	browserClient *browser.BrowserClient
}

// NewOreillyClient はブラウザクライアントを使用してO'Reillyクライアントを作成します
func NewOreillyClient(userID string, password string, debug bool, tmpDir string) (*OreillyClient, error) {
	// Cookieマネージャーを作成
	cookieManager := cookie.NewCookieManager(tmpDir)

	// ブラウザクライアントを作成してログイン
	browserClient, err := browser.NewBrowserClient(userID, password, cookieManager, debug, tmpDir)
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
	Query        string   `json:"q"`
	Rows         int      `json:"rows,omitempty"`
	Languages    []string `json:"language,omitempty"`
	TzOffset     int      `json:"tzOffset,omitempty"`
	AiaOnly      bool     `json:"aia_only,omitempty"`
	FeatureFlags string   `json:"feature_flags,omitempty"`
	Report       bool     `json:"report,omitempty"`
	IsTopics     bool     `json:"isTopics,omitempty"`
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
	results, err := c.browserClient.SearchContent(params.Query, options)
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

// SearchContent はブラウザクライアントのSearchContentメソッドを直接呼び出します
func (c *OreillyClient) SearchContent(query string, options map[string]interface{}) ([]map[string]interface{}, error) {
	if c.browserClient == nil {
		return nil, fmt.Errorf("browser client is not available")
	}
	return c.browserClient.SearchContent(query, options)
}

// getStringValue はmap[string]interface{}から文字列値を安全に取得するヘルパー関数
func getStringValue(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}
