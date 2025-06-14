package browser

import (
	"context"
	"fmt"
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

	// Create simple TOC response without API call
	toc := &TableOfContentsResponse{
		BookID:          productID,
		BookTitle:       bookDetail.Title,
		TableOfContents: []TableOfContentsItem{},
		TotalChapters:   0,
		Metadata: map[string]interface{}{
			"extraction_method": "not_implemented",
		},
	}

	return &BookOverviewAndTOCResponse{
		BookDetail:      *bookDetail,
		TableOfContents: *toc,
	}, nil
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

