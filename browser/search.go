package browser

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

// normalizeSearchResult converts api.RawSearchResult to a map suitable for consumption
func normalizeSearchResult(raw api.RawSearchResult, index int) map[string]interface{} {
	// URL normalization
	itemURL := ""
	if raw.WebUrl != nil {
		itemURL = *raw.WebUrl
	}
	if itemURL == "" && raw.Url != nil {
		itemURL = *raw.Url
	}
	if itemURL == "" && raw.LearningUrl != nil {
		itemURL = *raw.LearningUrl
	}
	if itemURL == "" && raw.Link != nil {
		itemURL = *raw.Link
	}
	if itemURL == "" && raw.ProductId != nil && *raw.ProductId != "" {
		itemURL = "https://learning.oreilly.com/library/view/-/" + *raw.ProductId + "/"
	}
	if itemURL != "" && !strings.HasPrefix(itemURL, "http") {
		if strings.HasPrefix(itemURL, "/") {
			itemURL = "https://learning.oreilly.com" + itemURL
		}
	}

	// Authors normalization
	var authors []Author
	if raw.Authors != nil {
		for _, author := range *raw.Authors {
			authors = append(authors, Author{Name: author})
		}
	}
	if raw.Author != nil && raw.Author.Name != nil {
		authors = append(authors, Author{Name: *raw.Author.Name})
	}
	if raw.Creators != nil {
		for _, creator := range *raw.Creators {
			if creator.Name != nil {
				authors = append(authors, Author{Name: *creator.Name})
			}
		}
	}
	if raw.AuthorNames != nil {
		for _, name := range *raw.AuthorNames {
			authors = append(authors, Author{Name: name})
		}
	}

	// Content type determination
	contentType := ""
	if raw.ContentType != nil {
		contentType = *raw.ContentType
	}
	if contentType == "" && raw.Type != nil {
		contentType = *raw.Type
	}
	if contentType == "" && raw.Format != nil {
		contentType = *raw.Format
	}
	if contentType == "" && raw.ProductType != nil {
		contentType = *raw.ProductType
	}
	if contentType == "" {
		if strings.Contains(itemURL, "/video") {
			contentType = "video"
		} else if strings.Contains(itemURL, "/library/view/") || strings.Contains(itemURL, "/book/") {
			contentType = "book"
		} else {
			contentType = "unknown"
		}
	}

	// Title extraction
	title := ""
	if raw.Title != nil {
		title = *raw.Title
	}
	if title == "" && raw.Name != nil {
		title = *raw.Name
	}
	if title == "" && raw.DisplayTitle != nil {
		title = *raw.DisplayTitle
	}
	if title == "" && raw.ProductName != nil {
		title = *raw.ProductName
	}

	// Description extraction
	description := ""
	if raw.Description != nil {
		description = *raw.Description
	}
	if description == "" && raw.Summary != nil {
		description = *raw.Summary
	}
	if description == "" && raw.Excerpt != nil {
		description = *raw.Excerpt
	}
	if description == "" && raw.DescriptionWithMarkups != nil {
		description = *raw.DescriptionWithMarkups
	}
	if description == "" && raw.ShortDescription != nil {
		description = *raw.ShortDescription
	}

	// Publisher extraction
	publisher := ""
	if raw.Publisher != nil {
		publisher = *raw.Publisher
	}
	if publisher == "" && raw.Publishers != nil && len(*raw.Publishers) > 0 {
		publisher = (*raw.Publishers)[0]
	}
	if publisher == "" && raw.Imprint != nil {
		publisher = *raw.Imprint
	}
	if publisher == "" && raw.PublisherName != nil {
		publisher = *raw.PublisherName
	}

	// Published date extraction
	publishedDate := ""
	if raw.PublishedDate != nil {
		publishedDate = *raw.PublishedDate
	}
	if publishedDate == "" && raw.PublicationDate != nil {
		publishedDate = *raw.PublicationDate
	}
	if publishedDate == "" && raw.DatePublished != nil {
		publishedDate = *raw.DatePublished
	}
	if publishedDate == "" && raw.PubDate != nil {
		publishedDate = *raw.PubDate
	}

	// ID generation
	id := ""
	if raw.ProductId != nil {
		id = *raw.ProductId
	}
	if id == "" && raw.Id != nil {
		id = *raw.Id
	}
	if id == "" && raw.Ourn != nil {
		id = *raw.Ourn
	}
	if id == "" && raw.Isbn != nil {
		id = *raw.Isbn
	}
	if id == "" {
		id = fmt.Sprintf("api_result_%d", index)
	}

	return map[string]interface{}{
		"id":           id,
		"title":        title,
		"authors":      authors,
		"content_type": contentType,
		"description":  description,
		"url":          itemURL,
		"ourn": func() string {
			if raw.Ourn != nil {
				return *raw.Ourn
			}
			return ""
		}(),
		"publisher":      publisher,
		"published_date": publishedDate,
		"source":         "api_search_oreilly",
	}
}

// makeHTTPSearchRequest performs the O'Reilly search API call using generated OpenAPI client
func (bc *BrowserClient) makeHTTPSearchRequest(query string, rows, tzOffset int, aiaOnly bool, featureFlags string, report, isTopics bool) (*api.SearchAPIResponse, error) {
	// Create OpenAPI client
	client := &api.ClientWithResponses{
		ClientInterface: &api.Client{
			Server:         APIEndpointBase,
			Client:         bc.httpClient,
			RequestEditors: []api.RequestEditorFn{bc.CreateRequestEditor()},
		},
	}

	// Create search parameters
	params := &api.SearchContentV2Params{
		Query:        query,
		Rows:         &rows,
		TzOffset:     &tzOffset,
		AiaOnly:      &aiaOnly,
		FeatureFlags: &featureFlags,
		Report:       &report,
		IsTopics:     &isTopics,
	}

	// OpenAPI検索リクエスト (タイムアウト付き)
	apiCtx, apiCancel := context.WithTimeout(context.Background(), APIOperationTimeout)
	defer apiCancel()
	slog.Debug("OpenAPI検索リクエスト開始", "query", query, "rows", rows)

	// Make the API call
	resp, err := client.SearchContentV2WithResponse(apiCtx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAPI request failed: %w", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no valid JSON response received")
	}

	return resp.JSON200, nil
}

// SearchContent は O'Reilly Learning Platform の内部 API を使用して検索を実行します
func (bc *BrowserClient) SearchContent(query string, options map[string]interface{}) ([]map[string]interface{}, error) {
	slog.Info("API検索を開始します", "query", query)

	// オプションのデフォルト値を設定
	rows := 100
	if r, ok := options["rows"].(int); ok && r > 0 {
		rows = r
	}

	// 言語オプションは現在使用していないため、将来の拡張用として保持
	_ = options["languages"] // 未使用警告を回避

	tzOffset := -9 // JST
	if tz, ok := options["tzOffset"].(int); ok {
		tzOffset = tz
	}

	aiaOnly := false
	if aia, ok := options["aia_only"].(bool); ok {
		aiaOnly = aia
	}

	featureFlags := "improveSearchFilters"
	if ff, ok := options["feature_flags"].(string); ok && ff != "" {
		featureFlags = ff
	}

	report := true
	if rep, ok := options["report"].(bool); ok {
		report = rep
	}

	isTopics := false
	if topics, ok := options["isTopics"].(bool); ok {
		isTopics = topics
	}

	// Use OpenAPI generated client for search
	var results []map[string]interface{}

	apiResponse, err := bc.makeHTTPSearchRequest(query, rows, tzOffset, aiaOnly, featureFlags, report, isTopics)
	if err != nil {
		slog.Error("API検索に失敗しました", "error", err, "query", query)
		return nil, fmt.Errorf("API search failed: %w", err)
	}

	// Extract results from API response
	var rawResults []api.RawSearchResult
	if apiResponse.Data != nil && apiResponse.Data.Products != nil && len(*apiResponse.Data.Products) > 0 {
		rawResults = *apiResponse.Data.Products
	} else if apiResponse.Results != nil && len(*apiResponse.Results) > 0 {
		rawResults = *apiResponse.Results
	} else if apiResponse.Items != nil && len(*apiResponse.Items) > 0 {
		rawResults = *apiResponse.Items
	} else if apiResponse.Hits != nil && len(*apiResponse.Hits) > 0 {
		rawResults = *apiResponse.Hits
	}

	slog.Debug("API検索レスポンス取得", "result_count", len(rawResults))

	// Normalize results using Go instead of JavaScript
	for i, rawResult := range rawResults {
		if i >= rows {
			break
		}
		normalized := normalizeSearchResult(rawResult, i)
		results = append(results, normalized)
	}

	slog.Info("API検索が完了しました", "query", query, "result_count", len(results))
	return results, nil
}
