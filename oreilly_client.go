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
	baseURL    = "https://learning.oreilly.com/api/v2"
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
		fmt.Sprintf("%s/search/", baseURL),
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
