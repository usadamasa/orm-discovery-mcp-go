package browser

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

// firstString returns the value of the first non-nil, non-empty string pointer.
func firstString(ptrs ...*string) string {
	for _, p := range ptrs {
		if p != nil && *p != "" {
			return *p
		}
	}
	return ""
}

// normalizeURL extracts and normalizes a URL from the raw search result.
func normalizeURL(raw api.RawSearchResult) string {
	itemURL := firstString(raw.WebUrl, raw.Url, raw.LearningUrl, raw.Link)
	if itemURL == "" && raw.ProductId != nil && *raw.ProductId != "" {
		itemURL = "https://learning.oreilly.com/library/view/-/" + *raw.ProductId + "/"
	}
	if itemURL != "" && !strings.HasPrefix(itemURL, "http") && strings.HasPrefix(itemURL, "/") {
		itemURL = "https://learning.oreilly.com" + itemURL
	}
	return itemURL
}

// extractAuthors collects authors from the 4 possible source fields.
func extractAuthors(raw api.RawSearchResult) []Author {
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
	return authors
}

// Content type constants for search result classification.
const (
	ContentTypeBook    = "book"
	ContentTypeVideo   = "video"
	ContentTypeUnknown = "unknown"
)

// inferContentType determines content type from explicit fields or URL heuristics.
func inferContentType(raw api.RawSearchResult, itemURL string) string {
	ct := firstString(raw.ContentType, raw.Type, raw.Format, raw.ProductType)
	if ct != "" {
		return ct
	}
	if strings.Contains(itemURL, "/video") {
		return ContentTypeVideo
	}
	if strings.Contains(itemURL, "/library/view/") || strings.Contains(itemURL, "/book/") {
		return ContentTypeBook
	}
	return ContentTypeUnknown
}

// normalizeSearchResult converts api.RawSearchResult to a map suitable for consumption
func normalizeSearchResult(raw api.RawSearchResult, index int) map[string]interface{} {
	itemURL := normalizeURL(raw)

	id := firstString(raw.ProductId, raw.Id, raw.Ourn, raw.Isbn)
	if id == "" {
		id = fmt.Sprintf("api_result_%d", index)
	}

	publisher := firstString(raw.Publisher)
	if publisher == "" && raw.Publishers != nil && len(*raw.Publishers) > 0 {
		publisher = (*raw.Publishers)[0]
	}
	if publisher == "" {
		publisher = firstString(raw.Imprint, raw.PublisherName)
	}

	return map[string]interface{}{
		"id":             id,
		"title":          firstString(raw.Title, raw.Name, raw.DisplayTitle, raw.ProductName),
		"authors":        extractAuthors(raw),
		"content_type":   inferContentType(raw, itemURL),
		"description":    firstString(raw.Description, raw.Summary, raw.Excerpt, raw.DescriptionWithMarkups, raw.ShortDescription),
		"url":            itemURL,
		"ourn":           firstString(raw.Ourn),
		"publisher":      publisher,
		"published_date": firstString(raw.PublishedDate, raw.PublicationDate, raw.DatePublished, raw.PubDate),
		"source":         "api_search_oreilly",
	}
}

// makeHTTPSearchRequest performs the O'Reilly search API call using generated OpenAPI client.
// Returns the API response and total count of matching results.
func (bc *BrowserClient) makeHTTPSearchRequest(query string, rows, offset, tzOffset int, aiaOnly bool, featureFlags string, report, isTopics bool) (*api.SearchAPIResponse, int, error) {
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
		Offset:       &offset,
		TzOffset:     &tzOffset,
		AiaOnly:      &aiaOnly,
		FeatureFlags: &featureFlags,
		Report:       &report,
		IsTopics:     &isTopics,
	}

	// OpenAPI検索リクエスト (タイムアウト付き)
	apiCtx, apiCancel := context.WithTimeout(context.Background(), APIOperationTimeout)
	defer apiCancel()
	slog.Debug("OpenAPI検索リクエスト開始", "query", query, "rows", rows, "offset", offset)

	// Make the API call
	resp, err := client.SearchContentV2WithResponse(apiCtx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("OpenAPI request failed: %w", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 200 {
		return nil, 0, fmt.Errorf("API request failed with status %d", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		return nil, 0, fmt.Errorf("no valid JSON response received")
	}

	// Extract total count
	totalCount := 0
	if resp.JSON200.TotalCount != nil {
		totalCount = *resp.JSON200.TotalCount
	}

	return resp.JSON200, totalCount, nil
}

// searchOptions holds parsed search parameters with defaults applied.
type searchOptions struct {
	rows         int
	offset       int
	tzOffset     int
	aiaOnly      bool
	featureFlags string
	report       bool
	isTopics     bool
}

// parseSearchOptions extracts search parameters from the options map, applying defaults.
func parseSearchOptions(options map[string]interface{}) searchOptions {
	opts := searchOptions{
		rows:         100,
		offset:       0,
		tzOffset:     -9, // JST
		featureFlags: "improveSearchFilters",
		report:       true,
	}

	if r, ok := options["rows"].(int); ok && r > 0 {
		opts.rows = r
	}
	if o, ok := options["offset"].(int); ok && o > 0 {
		opts.offset = o
	}
	if tz, ok := options["tzOffset"].(int); ok {
		opts.tzOffset = tz
	}
	if aia, ok := options["aia_only"].(bool); ok {
		opts.aiaOnly = aia
	}
	if ff, ok := options["feature_flags"].(string); ok && ff != "" {
		opts.featureFlags = ff
	}
	if rep, ok := options["report"].(bool); ok {
		opts.report = rep
	}
	if topics, ok := options["isTopics"].(bool); ok {
		opts.isTopics = topics
	}

	return opts
}

// SearchContent は O'Reilly Learning Platform の内部 API を使用して検索を実行します。
// Returns normalized results and total count of matching results.
func (bc *BrowserClient) SearchContent(query string, options map[string]interface{}) ([]map[string]interface{}, int, error) {
	slog.Info("API検索を開始します", "query", query)

	opts := parseSearchOptions(options)

	// Use OpenAPI generated client for search
	var results []map[string]interface{}

	apiResponse, totalCount, err := bc.makeHTTPSearchRequest(query, opts.rows, opts.offset, opts.tzOffset, opts.aiaOnly, opts.featureFlags, opts.report, opts.isTopics)
	if err != nil {
		slog.Error("API検索に失敗しました", "error", err, "query", query)
		return nil, 0, fmt.Errorf("API search failed: %w", err)
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

	slog.Debug("API検索レスポンス取得", "result_count", len(rawResults), "total_count", totalCount)

	// Normalize results using Go instead of JavaScript
	for i, rawResult := range rawResults {
		if i >= opts.rows {
			break
		}
		normalized := normalizeSearchResult(rawResult, i)
		results = append(results, normalized)
	}

	slog.Info("API検索が完了しました", "query", query, "result_count", len(results), "total_count", totalCount)
	return results, totalCount, nil
}
