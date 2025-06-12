package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	baseURL    = "https://learning.oreilly.com"
	apiTimeout = 30 * time.Second
)

// OreillyClient はO'Reilly Learning Platform APIのクライアントです
type OreillyClient struct {
	httpClient    *http.Client
	cookieStr     string
	jwtToken      string
	sessionID     string
	refreshToken  string
	browserClient *BrowserClient
}

// NewOreillyClient は新しいO'Reillyクライアントを作成します
func NewOreillyClient(cookieStr, jwtToken, sessionID, refreshToken string) *OreillyClient {
	return &OreillyClient{
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		cookieStr:    cookieStr,
		jwtToken:     jwtToken,
		sessionID:    sessionID,
		refreshToken: refreshToken,
	}
}

// NewOreillyClientWithBrowser はブラウザクライアントを使用してO'Reillyクライアントを作成します
func NewOreillyClientWithBrowser(userID, password string) (*OreillyClient, error) {
	// ブラウザクライアントを作成してログイン
	browserClient, err := NewBrowserClient(userID, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser client: %w", err)
	}

	client := &OreillyClient{
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		browserClient: browserClient,
		cookieStr:     browserClient.GetCookieString(),
		jwtToken:      browserClient.GetJWTToken(),
		sessionID:     browserClient.GetSessionID(),
		refreshToken:  browserClient.GetRefreshToken(),
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

// buildCookieString は認証に必要なCookieを構築します
func (c *OreillyClient) buildCookieString() string {
	if c.cookieStr != "" {
		// 完全なCookie文字列が提供されている場合はそれを使用
		return c.cookieStr
	}
	
	// 個別のキーから必要最小限のCookieを構築
	var cookies []string
	if c.jwtToken != "" {
		cookies = append(cookies, fmt.Sprintf("orm-jwt=%s", c.jwtToken))
	}
	if c.sessionID != "" {
		cookies = append(cookies, fmt.Sprintf("groot_sessionid=%s", c.sessionID))
	}
	if c.refreshToken != "" {
		cookies = append(cookies, fmt.Sprintf("orm-rt=%s", c.refreshToken))
	}
	
	if len(cookies) == 0 {
		return ""
	}
	
	// 複数のCookieを; で結合
	result := ""
	for i, cookie := range cookies {
		if i > 0 {
			result += "; "
		}
		result += cookie
	}
	return result
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

	// ブラウザクライアントで検索を実行
	results, err := c.browserClient.SearchContent(params.Query, options)
	if err != nil {
		return nil, fmt.Errorf("browser search failed: %w", err)
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

type CollectionResponse struct {
	Results []Collection `json:"collections"`
}

// Search はO'Reilly Learning Platformで検索を実行します
func (c *OreillyClient) ListCollections(ctx context.Context) (*CollectionResponse, error) {
	log.Printf("O'Reilly APIでコレクションが要求されました\n")
	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/v3/collections/", baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーの設定 - Cookie認証とJWT認証の両方を送信
	if c.jwtToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	}
	cookieStr := c.buildCookieString()
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// リクエストの実行
	log.Printf("O'Reilly APIでコレクションの取得を実行します\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("O'Reilly APIからのレスポンスステータス: %d\n", resp.StatusCode)
	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// レスポンスのパース
	var collectionResp CollectionResponse
	if err := json.Unmarshal(body, &collectionResp.Results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("O'Reilly APIからの検索結果: %d件", len(collectionResp.Results))
	return &collectionResp, nil
}

// CreateCollectionParams はコレクション作成パラメータの構造体です
type CreateCollectionParams struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	PrivacySetting string `json:"sharing"` // "private", "public", "unlisted"
}

// CreateCollectionResponse はコレクション作成レスポンスの構造体です
type CreateCollectionResponse struct {
	Collection Collection `json:"collection"`
}

// CreateCollection は新しいコレクションを作成します
func (c *OreillyClient) CreateCollection(ctx context.Context, params CreateCollectionParams) (*CreateCollectionResponse, error) {
	log.Printf("O'Reilly APIでコレクション作成が要求されました: %s\n", params.Name)
	
	if params.Name == "" {
		return nil, fmt.Errorf("collection name cannot be empty")
	}
	
	// デフォルト値の設定
	if params.PrivacySetting == "" {
		params.PrivacySetting = "private"
	}
	
	// リクエストボディの作成
	requestBody := map[string]interface{}{
		"name":        params.Name,
		"description": params.Description,
		"sharing":     params.PrivacySetting,
	}
	
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/v3/collections/", baseURL),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// ヘッダーの設定
	if c.jwtToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	}
	cookieStr := c.buildCookieString()
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// リクエストの実行
	log.Printf("O'Reilly APIでコレクション作成を実行します\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("O'Reilly APIからのレスポンスステータス: %d\n", resp.StatusCode)
	
	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// レスポンスのパース
	var createResp CreateCollectionResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	log.Printf("コレクション作成成功: %s", createResp.Collection.Name)
	return &createResp, nil
}

// AddToCollectionParams はコレクションへのコンテンツ追加パラメータの構造体です
type AddToCollectionParams struct {
	CollectionID string `json:"collection_id"`
	ContentID    string `json:"content_id"`
	ContentType  string `json:"content_type"`
}

// AddToCollection はコレクションにコンテンツを追加します
func (c *OreillyClient) AddToCollection(ctx context.Context, params AddToCollectionParams) error {
	log.Printf("O'Reilly APIでコレクションへのコンテンツ追加が要求されました: %s -> %s\n", params.ContentID, params.CollectionID)
	
	if params.CollectionID == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}
	if params.ContentID == "" {
		return fmt.Errorf("content ID cannot be empty")
	}
	
	// リクエストボディの作成
	requestBody := map[string]interface{}{
		"ourn": params.ContentID,
	}
	if params.ContentType != "" {
		requestBody["content_type"] = params.ContentType
	}
	
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}
	
	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/v3/collections/%s/content/", baseURL, params.CollectionID),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// ヘッダーの設定
	if c.jwtToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	}
	cookieStr := c.buildCookieString()
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// リクエストの実行
	log.Printf("O'Reilly APIでコンテンツ追加を実行します\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("O'Reilly APIからのレスポンスステータス: %d\n", resp.StatusCode)
	
	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	log.Printf("コンテンツ追加成功: %s", params.ContentID)
	return nil
}

// RemoveFromCollectionParams はコレクションからのコンテンツ削除パラメータの構造体です
type RemoveFromCollectionParams struct {
	CollectionID string `json:"collection_id"`
	ContentID    string `json:"content_id"`
}

// RemoveFromCollection はコレクションからコンテンツを削除します
func (c *OreillyClient) RemoveFromCollection(ctx context.Context, params RemoveFromCollectionParams) error {
	log.Printf("O'Reilly APIでコレクションからのコンテンツ削除が要求されました: %s <- %s\n", params.ContentID, params.CollectionID)
	
	if params.CollectionID == "" {
		return fmt.Errorf("collection ID cannot be empty")
	}
	if params.ContentID == "" {
		return fmt.Errorf("content ID cannot be empty")
	}
	
	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		fmt.Sprintf("%s/api/v3/collections/%s/content/%s/", baseURL, params.CollectionID, params.ContentID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// ヘッダーの設定
	if c.jwtToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	}
	cookieStr := c.buildCookieString()
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}
	req.Header.Set("Accept", "application/json")
	
	// リクエストの実行
	log.Printf("O'Reilly APIでコンテンツ削除を実行します\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("O'Reilly APIからのレスポンスステータス: %d\n", resp.StatusCode)
	
	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	log.Printf("コンテンツ削除成功: %s", params.ContentID)
	return nil
}

// GetCollectionDetailsParams はコレクション詳細取得パラメータの構造体です
type GetCollectionDetailsParams struct {
	CollectionID   string `json:"collection_id"`
	IncludeContent bool   `json:"include_content"`
}

// GetCollectionDetailsResponse はコレクション詳細レスポンスの構造体です
type GetCollectionDetailsResponse struct {
	Collection Collection `json:"collection"`
}

// GetCollectionDetails は特定のコレクションの詳細情報を取得します
func (c *OreillyClient) GetCollectionDetails(ctx context.Context, params GetCollectionDetailsParams) (*GetCollectionDetailsResponse, error) {
	log.Printf("O'Reilly APIでコレクション詳細が要求されました: %s\n", params.CollectionID)
	
	if params.CollectionID == "" {
		return nil, fmt.Errorf("collection ID cannot be empty")
	}
	
	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/api/v3/collections/%s/", baseURL, params.CollectionID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// クエリパラメータの設定
	if params.IncludeContent {
		q := req.URL.Query()
		q.Set("include_content", "true")
		req.URL.RawQuery = q.Encode()
	}
	
	// ヘッダーの設定
	if c.jwtToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	}
	cookieStr := c.buildCookieString()
	if cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}
	req.Header.Set("Accept", "application/json")
	
	// リクエストの実行
	log.Printf("O'Reilly APIでコレクション詳細取得を実行します\n")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("O'Reilly APIからのレスポンスステータス: %d\n", resp.StatusCode)
	
	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// エラーレスポンスの処理
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// レスポンスのパース
	var detailsResp GetCollectionDetailsResponse
	if err := json.Unmarshal(body, &detailsResp.Collection); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	log.Printf("コレクション詳細取得成功: %s", detailsResp.Collection.Name)
	return &detailsResp, nil
}
