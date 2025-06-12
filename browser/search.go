package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// normalizeSearchResult converts RawSearchResult to a map suitable for consumption
func normalizeSearchResult(raw RawSearchResult, index int) map[string]interface{} {
	// URL normalization
	itemURL := raw.WebURL
	if itemURL == "" {
		itemURL = raw.URL
	}
	if itemURL == "" {
		itemURL = raw.LearningURL
	}
	if itemURL == "" {
		itemURL = raw.Link
	}
	if itemURL == "" && raw.ProductID != "" {
		itemURL = "https://learning.oreilly.com/library/view/-/" + raw.ProductID + "/"
	}
	if itemURL != "" && !strings.HasPrefix(itemURL, "http") {
		if strings.HasPrefix(itemURL, "/") {
			itemURL = "https://learning.oreilly.com" + itemURL
		}
	}

	// Authors normalization
	var authors []string
	if len(raw.Authors) > 0 {
		authors = raw.Authors
	} else if raw.Author != "" {
		authors = []string{raw.Author}
	} else if len(raw.Creators) > 0 {
		for _, creator := range raw.Creators {
			if creator.Name != "" {
				authors = append(authors, creator.Name)
			}
		}
	} else if len(raw.AuthorNames) > 0 {
		authors = raw.AuthorNames
	}

	// Content type determination
	contentType := raw.ContentType
	if contentType == "" {
		contentType = raw.Type
	}
	if contentType == "" {
		contentType = raw.Format
	}
	if contentType == "" {
		contentType = raw.ProductType
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
	title := raw.Title
	if title == "" {
		title = raw.Name
	}
	if title == "" {
		title = raw.DisplayTitle
	}
	if title == "" {
		title = raw.ProductName
	}

	// Description extraction
	description := raw.Description
	if description == "" {
		description = raw.Summary
	}
	if description == "" {
		description = raw.Excerpt
	}
	if description == "" {
		description = raw.DescriptionWithMarkups
	}
	if description == "" {
		description = raw.ShortDescription
	}

	// Publisher extraction
	publisher := raw.Publisher
	if publisher == "" && len(raw.Publishers) > 0 {
		publisher = raw.Publishers[0]
	}
	if publisher == "" {
		publisher = raw.Imprint
	}
	if publisher == "" {
		publisher = raw.PublisherName
	}

	// Published date extraction
	publishedDate := raw.PublishedDate
	if publishedDate == "" {
		publishedDate = raw.PublicationDate
	}
	if publishedDate == "" {
		publishedDate = raw.DatePublished
	}
	if publishedDate == "" {
		publishedDate = raw.PubDate
	}

	// ID generation
	id := raw.ProductID
	if id == "" {
		id = raw.ID
	}
	if id == "" {
		id = raw.OURN
	}
	if id == "" {
		id = raw.ISBN
	}
	if id == "" {
		id = fmt.Sprintf("api_result_%d", index)
	}

	return map[string]interface{}{
		"id":             id,
		"title":          title,
		"authors":        authors,
		"content_type":   contentType,
		"description":    description,
		"url":            itemURL,
		"ourn":           raw.OURN,
		"publisher":      publisher,
		"published_date": publishedDate,
		"source":         "api_search_oreilly",
	}
}

// makeHTTPSearchRequest performs the O'Reilly search API call using HTTP client
func (bc *BrowserClient) makeHTTPSearchRequest(baseURL, query string, rows, tzOffset int, aiaOnly bool, featureFlags string, report, isTopics bool) (*SearchAPIResponse, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("rows", strconv.Itoa(rows))
	params.Set("tzOffset", strconv.Itoa(tzOffset))
	params.Set("aia_only", strconv.FormatBool(aiaOnly))
	params.Set("feature_flags", featureFlags)
	params.Set("report", strconv.FormatBool(report))
	params.Set("isTopics", strconv.FormatBool(isTopics))

	fullURL := baseURL + "?" + params.Encode()
	log.Printf("Making HTTP API request to: %s", fullURL)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", bc.userAgent)

	// Add cookies if available
	for _, cookie := range bc.cookies {
		req.AddCookie(cookie)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResponse SearchAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &apiResponse, nil
}

// SearchContent は O'Reilly Learning Platform の内部 API を使用して検索を実行します
func (bc *BrowserClient) SearchContent(query string, options map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("API検索を開始します: %s", query)
	
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

	// First, try to get current context (cookies) with minimal JavaScript
	err := chromedp.Run(bc.ctx,
		chromedp.Navigate("https://learning.oreilly.com/search/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		
		// Get current domain context with minimal JavaScript
		chromedp.ActionFunc(func(ctx context.Context) error {
			var domain string
			err := chromedp.Evaluate(`window.location.hostname`, &domain).Do(ctx)
			if err != nil {
				log.Printf("Could not get domain context: %v", err)
			} else {
				log.Printf("Current domain: %s", domain)
			}
			
			// Update cookies from browser
			cookiesResp, err := network.GetCookies().Do(ctx)
			if err != nil {
				log.Printf("Could not get cookies: %v", err)
				return nil
			}
			
			// Convert to http.Cookie
			bc.cookies = make([]*http.Cookie, len(cookiesResp))
			for i, c := range cookiesResp {
				bc.cookies[i] = &http.Cookie{
					Name:   c.Name,
					Value:  c.Value,
					Domain: c.Domain,
					Path:   c.Path,
				}
			}
			log.Printf("Updated %d cookies from browser", len(bc.cookies))
			return nil
		}),
	)
	
	if err != nil {
		log.Printf("Failed to get browser context: %v", err)
		return nil, fmt.Errorf("failed to get browser context: %w", err)
	}

	// Try different API endpoints using Go HTTP client
	endpoints := []string{
		"https://learning.oreilly.com" + APIEndpointV2,
		"https://learning.oreilly.com" + APIEndpointSearch,
		"https://www.oreilly.com" + APIEndpointSearch,
		"https://learning.oreilly.com" + APIEndpointLegacy,
	}
	
	var results []map[string]interface{}
	var lastErr error
	
	for i, endpoint := range endpoints {
		log.Printf("Trying API endpoint %d/%d: %s", i+1, len(endpoints), endpoint)
		
		apiResponse, err := bc.makeHTTPSearchRequest(endpoint, query, rows, tzOffset, aiaOnly, featureFlags, report, isTopics)
		if err != nil {
			log.Printf("Endpoint %s failed: %v", endpoint, err)
			lastErr = err
			continue
		}
		
		// Extract results from API response
		var rawResults []RawSearchResult
		if apiResponse.Data != nil && len(apiResponse.Data.Products) > 0 {
			rawResults = apiResponse.Data.Products
		} else if len(apiResponse.Results) > 0 {
			rawResults = apiResponse.Results
		} else if len(apiResponse.Items) > 0 {
			rawResults = apiResponse.Items
		} else if len(apiResponse.Hits) > 0 {
			rawResults = apiResponse.Hits
		}
		
		log.Printf("API endpoint %s returned %d results", endpoint, len(rawResults))
		
		// Normalize results using Go instead of JavaScript
		for i, rawResult := range rawResults {
			if i >= rows {
				break
			}
			normalized := normalizeSearchResult(rawResult, i)
			results = append(results, normalized)
		}
		
		if len(results) > 0 {
			log.Printf("Successfully retrieved %d results from %s", len(results), endpoint)
			break
		}
	}
	
	if len(results) == 0 && lastErr != nil {
		return nil, fmt.Errorf("all API endpoints failed, last error: %w", lastErr)
	}
	
	log.Printf("API検索が完了しました。%d件の結果を取得: %s", len(results), query)
	return results, nil
}