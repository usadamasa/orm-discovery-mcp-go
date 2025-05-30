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
	baseURL    = "https://learning.oreilly.com/api"
	apiTimeout = 30 * time.Second
)

// OreillyClient はO'Reilly Learning Platform APIのクライアントです
type OreillyClient struct {
	httpClient *http.Client
	jwtToken   string
}

// NewOreillyClient は新しいO'Reillyクライアントを作成します
func NewOreillyClient(jwtToken string) *OreillyClient {
	return &OreillyClient{
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		jwtToken: jwtToken,
	}
}

// SearchRequest は検索リクエストの構造体です
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// SearchResponse は検索レスポンスの構造体です
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Count   int            `json:"count"`
}

// SearchResult は検索結果の1件を表します
type SearchResult struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Type        string `json:"type"`
}

// Search はO'Reilly Learning Platformで検索を実行します
func (c *OreillyClient) Search(ctx context.Context, query string, limit int) (*SearchResponse, error) {
	log.Printf("O'Reilly APIで検索が要求されました: %s\n", query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 10 // デフォルト値
	}

	// リクエストボディの作成
	reqBody := SearchRequest{
		Query: query,
		Limit: limit,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// リクエストの作成
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/v2/search/", baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーの設定
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// リクエストの実行
	log.Printf("O'Reilly APIで検索を実行します: %s\n", query)
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
	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("O'Reilly APIからの検索結果: %d件", searchResp.Count)
	return &searchResp, nil
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
		fmt.Sprintf("%s/v3/collections/", baseURL),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーの設定
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.jwtToken))
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
