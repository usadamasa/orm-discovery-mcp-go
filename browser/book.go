package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"net/http"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

// GetBookDetailsAndTOC retrieves book details and table of contents from O'Reilly book URL
func (bc *BrowserClient) GetBookDetailsAndTOC(productID string) (*BookOverviewAndTOCResponse, error) {
	log.Printf("プロダクトIDから書籍詳細と目次を取得しています: %s", productID)

	// Get book details from API
	bookDetail, err := bc.getBookDetails(productID)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細取得失敗: %w", err)
	}

	// Get table of contents from API
	toc, err := bc.getBookTOC(productID)
	if err != nil {
		log.Printf("目次取得失敗、空の目次を返します: %v", err)
		// Create simple TOC response without API call
		toc = &TableOfContentsResponse{
			BookID:          productID,
			BookTitle:       bookDetail.Title,
			TableOfContents: []TableOfContentsItem{},
			TotalChapters:   0,
			Metadata: map[string]interface{}{
				"extraction_method": "fallback",
			},
		}
	}

	return &BookOverviewAndTOCResponse{
		BookDetail:      *bookDetail,
		TableOfContents: *toc,
	}, nil
}

// GetBookTOC retrieves table of contents for a specific book
func (bc *BrowserClient) GetBookTOC(productID string) (*TableOfContentsResponse, error) {
	return bc.getBookTOC(productID)
}

// Helper functions

// getBookDetails retrieves comprehensive book metadata from O'Reilly API using OpenAPI client
func (bc *BrowserClient) getBookDetails(productID string) (*BookDetailResponse, error) {
	log.Printf("書籍詳細を取得しています: %s", productID)

	// Create OpenAPI client
	client, err := api.NewClientWithResponses(APIEndpointBase,
		api.WithHTTPClient(bc.httpClient),
		api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			// Set headers
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
			req.Header.Set("User-Agent", bc.userAgent)

			// Add cookies if available
			for _, cookie := range bc.cookies {
				req.AddCookie(cookie)
			}
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAPI client: %v", err)
	}

	log.Printf("書籍詳細APIを試行しています: %s", productID)
	resp, err := client.GetBookDetailsWithResponse(context.Background(), productID)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細APIエンドポイントが失敗しました: %v", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no valid JSON response received")
	}

	// Convert from generated API type to local type
	bookDetail := convertAPIBookDetailToLocal(resp.JSON200)
	log.Printf("書籍詳細取得に成功しました: %s", bookDetail.Title)
	return bookDetail, nil
}

// convertAPIBookDetailToLocal converts from generated API BookDetailResponse to local BookDetailResponse
func convertAPIBookDetailToLocal(apiBook *api.BookDetailResponse) *BookDetailResponse {
	bookDetail := &BookDetailResponse{
		Metadata: make(map[string]interface{}),
	}

	if apiBook.Id != nil {
		bookDetail.ID = *apiBook.Id
	}
	if apiBook.Url != nil {
		bookDetail.URL = *apiBook.Url
	}
	if apiBook.WebUrl != nil {
		bookDetail.WebURL = *apiBook.WebUrl
	}
	if apiBook.Title != nil {
		bookDetail.Title = *apiBook.Title
	}
	if apiBook.Description != nil {
		bookDetail.Description = *apiBook.Description
	}
	if apiBook.Isbn != nil {
		bookDetail.ISBN = *apiBook.Isbn
	}
	if apiBook.VirtualPages != nil {
		bookDetail.VirtualPages = *apiBook.VirtualPages
	}
	if apiBook.AverageRating != nil {
		bookDetail.AverageRating = float64(*apiBook.AverageRating)
	}
	if apiBook.Cover != nil {
		bookDetail.Cover = *apiBook.Cover
	}
	if apiBook.Issued != nil {
		bookDetail.Issued = *apiBook.Issued
	}
	if apiBook.Language != nil {
		bookDetail.Language = *apiBook.Language
	}
	if apiBook.Metadata != nil {
		bookDetail.Metadata = *apiBook.Metadata
	}

	// Convert authors
	if apiBook.Authors != nil {
		for _, apiAuthor := range *apiBook.Authors {
			if apiAuthor.Name != nil {
				bookDetail.Authors = append(bookDetail.Authors, Author{Name: *apiAuthor.Name})
			}
		}
	}

	// Convert publishers
	if apiBook.Publishers != nil {
		for _, apiPublisher := range *apiBook.Publishers {
			publisher := Publisher{}
			if apiPublisher.Id != nil {
				publisher.ID = *apiPublisher.Id
			}
			if apiPublisher.Name != nil {
				publisher.Name = *apiPublisher.Name
			}
			if apiPublisher.Slug != nil {
				publisher.Slug = *apiPublisher.Slug
			}
			bookDetail.Publishers = append(bookDetail.Publishers, publisher)
		}
	}

	// Convert topics
	if apiBook.Topics != nil {
		for _, apiTopic := range *apiBook.Topics {
			topic := Topics{}
			if apiTopic.Name != nil {
				topic.Name = *apiTopic.Name
			}
			if apiTopic.Slug != nil {
				topic.Slug = *apiTopic.Slug
			}
			if apiTopic.Score != nil {
				topic.Score = float64(*apiTopic.Score)
			}
			if apiTopic.Uuid != nil {
				topic.UUID = *apiTopic.Uuid
			}
			if apiTopic.EpubIdentifier != nil {
				topic.EpubIdentifier = *apiTopic.EpubIdentifier
			}
			bookDetail.Topics = append(bookDetail.Topics, topic)
		}
	}

	return bookDetail
}

// getBookTOC retrieves table of contents from O'Reilly API using OpenAPI client
func (bc *BrowserClient) getBookTOC(productID string) (*TableOfContentsResponse, error) {
	log.Printf("目次を取得しています: %s", productID)

	// Create OpenAPI client
	client, err := api.NewClientWithResponses(APIEndpointBase,
		api.WithHTTPClient(bc.httpClient),
		api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			// Set headers
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
			req.Header.Set("User-Agent", bc.userAgent)

			// Add cookies if available
			for _, cookie := range bc.cookies {
				req.AddCookie(cookie)
			}
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAPI client: %v", err)
	}

	log.Printf("目次APIを試行しています: %s", productID)

	// Make raw HTTP request to see the actual response structure
	httpResp, err := client.GetBookFlatTOC(context.Background(), productID)
	if err != nil {
		return nil, fmt.Errorf("目次APIエンドポイントが失敗しました: %v", err)
	}
	defer httpResp.Body.Close()

	// Read the raw response body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check response status
	if httpResp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Try to parse as a flat TOC array first
	var flatTOCArray []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &flatTOCArray); err == nil {
		// Convert array to our expected structure
		return convertFlatTOCArrayToLocal(productID, flatTOCArray), nil
	}

	// If array parsing fails, try as object
	resp, err := client.GetBookFlatTOCWithResponse(context.Background(), productID)
	if err != nil {
		return nil, fmt.Errorf("目次APIエンドポイントが失敗しました: %v", err)
	}

	// Check response status
	if resp.HTTPResponse.StatusCode != 200 {
		// Log the raw response body for debugging
		log.Printf("API response status: %d", resp.HTTPResponse.StatusCode)
		log.Printf("API response body: %s", string(resp.Body))
		return nil, fmt.Errorf("API request failed with status %d", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		// Log the raw response body for debugging
		log.Printf("Failed to parse JSON response. Raw response: %s", string(resp.Body))
		return nil, fmt.Errorf("no valid JSON response received")
	}

	// Convert from generated API type to local type
	tocResponse := convertAPIFlatTOCToLocal(resp.JSON200)
	log.Printf("目次取得に成功しました: %s (%d items)", tocResponse.BookTitle, tocResponse.TotalChapters)
	return tocResponse, nil
}

// convertAPIFlatTOCToLocal converts from generated API FlatTOCResponse to local TableOfContentsResponse
func convertAPIFlatTOCToLocal(apiTOC *api.FlatTOCResponse) *TableOfContentsResponse {
	tocResponse := &TableOfContentsResponse{
		Metadata: make(map[string]interface{}),
	}

	if apiTOC.BookId != nil {
		tocResponse.BookID = *apiTOC.BookId
	}
	if apiTOC.BookTitle != nil {
		tocResponse.BookTitle = *apiTOC.BookTitle
	}
	if apiTOC.TotalItems != nil {
		tocResponse.TotalChapters = *apiTOC.TotalItems
	}
	if apiTOC.Metadata != nil {
		tocResponse.Metadata = *apiTOC.Metadata
	}

	// Convert TOC items
	if apiTOC.TocItems != nil {
		for _, apiItem := range *apiTOC.TocItems {
			item := TableOfContentsItem{}
			if apiItem.Id != nil {
				item.ID = *apiItem.Id
			}
			if apiItem.Title != nil {
				item.Title = *apiItem.Title
			}
			if apiItem.Href != nil {
				item.Href = *apiItem.Href
			}
			if apiItem.Level != nil {
				item.Level = *apiItem.Level
			}
			if apiItem.Parent != nil {
				item.Parent = *apiItem.Parent
			}
			if apiItem.Metadata != nil {
				item.Metadata = *apiItem.Metadata
			}

			tocResponse.TableOfContents = append(tocResponse.TableOfContents, item)
		}
	}

	// Mark as extracted via API
	tocResponse.Metadata["extraction_method"] = "api_flat_toc"

	return tocResponse
}

// convertFlatTOCArrayToLocal converts a flat TOC array response to local TableOfContentsResponse
func convertFlatTOCArrayToLocal(productID string, flatTOCArray []map[string]interface{}) *TableOfContentsResponse {
	tocResponse := &TableOfContentsResponse{
		BookID:          productID,
		BookTitle:       "", // Will be determined from first item or other means
		TableOfContents: []TableOfContentsItem{},
		TotalChapters:   len(flatTOCArray),
		Metadata: map[string]interface{}{
			"extraction_method": "api_flat_toc_array",
		},
	}

	// Convert array items to our structure
	for i, apiItem := range flatTOCArray {
		item := TableOfContentsItem{
			Metadata: make(map[string]interface{}),
		}

		if id, ok := apiItem["id"].(string); ok {
			item.ID = id
		} else {
			item.ID = fmt.Sprintf("toc-item-%d", i+1)
		}

		if title, ok := apiItem["title"].(string); ok {
			item.Title = title
		}

		if href, ok := apiItem["href"].(string); ok {
			item.Href = href
		}

		if level, ok := apiItem["level"].(float64); ok {
			item.Level = int(level)
		}

		if parent, ok := apiItem["parent"].(string); ok {
			item.Parent = parent
		}

		// Copy additional metadata
		for key, value := range apiItem {
			if key != "id" && key != "title" && key != "href" && key != "level" && key != "parent" {
				item.Metadata[key] = value
			}
		}

		tocResponse.TableOfContents = append(tocResponse.TableOfContents, item)
	}

	return tocResponse
}
