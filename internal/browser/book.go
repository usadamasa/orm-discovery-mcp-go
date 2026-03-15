package browser

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/generated/api"
)

// GetBookDetails retrieves book details and table of contents from O'Reilly book Product ID
func (bc *BrowserClient) GetBookDetails(productID string) (*BookDetailResponse, error) {
	slog.Info("プロダクトIDから書籍詳細を取得しています", "product_id", productID)

	// Get book details from API
	bookDetail, err := bc.getBookDetails(productID)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細取得失敗: %w", err)
	}

	return bookDetail, nil
}

// GetBookTOC retrieves a table of contents for a specific book
func (bc *BrowserClient) GetBookTOC(productID string) (*TableOfContentsResponse, error) {
	return bc.getBookTOC(productID)
}

// Helper functions

// getBookDetails retrieves book metadata from O'Reilly v2 epubs API using OpenAPI client
func (bc *BrowserClient) getBookDetails(productID string) (*BookDetailResponse, error) {
	slog.Debug("書籍詳細APIを呼び出しています (v2)", "product_id", productID)

	client, err := api.NewClientWithResponses(APIEndpointBase,
		api.WithHTTPClient(bc.httpClient),
		api.WithRequestEditorFn(bc.CreateRequestEditor()))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAPI client: %v", err)
	}

	apiCtx, apiCancel := context.WithTimeout(context.Background(), APIOperationTimeout)
	defer apiCancel()
	resp, err := client.GetBookDetailsWithResponse(apiCtx, productID)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細APIエンドポイントが失敗しました: %v", err)
	}

	if resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.HTTPResponse.StatusCode)
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no valid JSON response received")
	}

	bookDetail := convertAPIBookDetailToLocal(resp.JSON200)
	slog.Info("書籍詳細取得に成功しました", "title", bookDetail.Title, "product_id", productID)
	return bookDetail, nil
}

// derefString returns the value of a string pointer, or empty string if nil.
func derefString(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}

// convertAPIBookDetailToLocal converts from generated API BookDetailResponse to local BookDetailResponse
func convertAPIBookDetailToLocal(apiBook *api.BookDetailResponse) *BookDetailResponse {
	bookDetail := &BookDetailResponse{
		OURN:            derefString(apiBook.Ourn),
		Identifier:      derefString(apiBook.Identifier),
		ISBN:            derefString(apiBook.Isbn),
		URL:             derefString(apiBook.Url),
		ContentFormat:   derefString(apiBook.ContentFormat),
		Title:           derefString(apiBook.Title),
		PublicationDate: derefString(apiBook.PublicationDate),
		Language:        derefString(apiBook.Language),
	}

	if apiBook.VirtualPages != nil {
		bookDetail.VirtualPages = *apiBook.VirtualPages
	}
	if apiBook.PageCount != nil {
		bookDetail.PageCount = *apiBook.PageCount
	}
	if apiBook.Descriptions != nil {
		bookDetail.Descriptions = *apiBook.Descriptions
	}
	if apiBook.Tags != nil {
		bookDetail.Tags = *apiBook.Tags
	}
	if apiBook.Resources != nil {
		for _, r := range *apiBook.Resources {
			bookDetail.Resources = append(bookDetail.Resources, BookResource{
				URL:         derefString(r.Url),
				Type:        derefString(r.Type),
				Description: derefString(r.Description),
			})
		}
	}

	return bookDetail
}

// getBookTOC retrieves table of contents from O'Reilly v2 API
func (bc *BrowserClient) getBookTOC(productID string) (*TableOfContentsResponse, error) {
	slog.Debug("目次APIを呼び出しています (v2)", "product_id", productID)

	client, err := api.NewClientWithResponses(APIEndpointBase,
		api.WithHTTPClient(bc.httpClient),
		api.WithRequestEditorFn(bc.CreateRequestEditor()))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAPI client: %v", err)
	}

	apiCtx, apiCancel := context.WithTimeout(context.Background(), APIOperationTimeout)
	defer apiCancel()
	resp, err := client.GetBookTOCWithResponse(apiCtx, productID)
	if err != nil {
		return nil, fmt.Errorf("目次APIエンドポイントが失敗しました: %v", err)
	}

	if resp.HTTPResponse.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.HTTPResponse.StatusCode, string(resp.Body))
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no valid JSON response received")
	}

	tocResponse := convertV2TOCToLocal(productID, *resp.JSON200)
	slog.Info("目次取得に成功しました", "book_id", productID, "chapter_count", tocResponse.TotalChapters)
	return tocResponse, nil
}

// convertV2TOCToLocal converts v2 nested TOC items to local flat TableOfContentsResponse
func convertV2TOCToLocal(productID string, v2Items []api.V2TOCItem) *TableOfContentsResponse {
	tocResponse := &TableOfContentsResponse{
		BookID:          productID,
		TableOfContents: []TableOfContentsItem{},
		Metadata: map[string]any{
			"extraction_method": "api_v2_toc",
		},
	}

	// Flatten nested TOC into a flat list
	flattenV2TOCItems(v2Items, "", tocResponse)

	tocResponse.TotalChapters = len(tocResponse.TableOfContents)
	return tocResponse
}

// flattenV2TOCItems recursively flattens nested v2 TOC items into the response
func flattenV2TOCItems(items []api.V2TOCItem, parentID string, tocResponse *TableOfContentsResponse) {
	for _, item := range items {
		localItem := convertV2TOCItemToLocal(item, parentID)
		tocResponse.TableOfContents = append(tocResponse.TableOfContents, localItem)

		// Recurse into children
		if item.Children != nil {
			flattenV2TOCItems(*item.Children, localItem.ID, tocResponse)
		}
	}
}

// convertV2TOCItemToLocal converts a single v2 TOC item to local TableOfContentsItem
func convertV2TOCItemToLocal(item api.V2TOCItem, parentID string) TableOfContentsItem {
	localItem := TableOfContentsItem{
		Title:  derefString(item.Title),
		Parent: parentID,
	}

	// Extract href from reference_id (format: "bookId-/filename.html")
	if item.ReferenceId != nil {
		refID := *item.ReferenceId
		localItem.ID = refID
		// Extract filename part after the "-/" prefix
		if idx := strings.Index(refID, "-/"); idx >= 0 {
			localItem.Href = refID[idx+2:] // e.g., "ch01.html"
		}
	}

	if item.Depth != nil {
		localItem.Level = *item.Depth
	}

	return localItem
}
