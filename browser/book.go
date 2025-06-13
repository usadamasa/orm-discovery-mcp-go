package browser

import (
	"fmt"
	"log"

	"encoding/json"
)

// GetBookDetailsAndTOC retrieves book details and table of contents from O'Reilly book URL
func (bc *BrowserClient) GetBookDetailsAndTOC(productID string) (*BookOverviewAndTOCResponse, error) {
	log.Printf("プロダクトIDから書籍詳細と目次を取得しています: %s", productID)

	// Get book details from API
	bookDetail, err := bc.getBookDetails(productID)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細取得失敗: %w", err)
	}

	// TODO: Get the table of contents use https://learning.oreilly.com/api/v1/book/${PRODUCT_ID}/flat-toc/
	toc := &TableOfContentsResponse{
		BookID:          productID,
		BookTitle:       bookDetail.Title,
		TableOfContents: []TableOfContentsItem{},
		TotalChapters:   0,
		Metadata: map[string]interface{}{
			"extraction_method": "failed",
		},
	}

	return &BookOverviewAndTOCResponse{
		BookDetail:      *bookDetail,
		TableOfContents: *toc,
	}, nil
}

// Helper functions

// getBookDetails retrieves comprehensive book metadata from O'Reilly API
func (bc *BrowserClient) getBookDetails(productID string) (*BookDetailResponse, error) {
	log.Printf("書籍詳細を取得しています: %s", productID)

	endpoint := fmt.Sprintf(APIEndpointBase+BookAPIV1, productID)

	log.Printf("書籍詳細APIを試行しています: %s", endpoint)
	response, err := bc.makeHTTPRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("書籍詳細APIエンドポイントが失敗しました: %v", err)
	}
	bookDetail, parseErr := parseBookDetailResponse(response)
	if parseErr != nil {
		log.Printf("書籍詳細レスポンス解析エラー: %v", parseErr)
		return nil, fmt.Errorf("書籍詳細レスポンス解析エラー: %v", parseErr)
	}
	log.Printf("書籍詳細取得に成功しました: %s", bookDetail.Title)
	return bookDetail, nil
}

// parseBookDetailResponse parses book detail API response
func parseBookDetailResponse(response []byte) (*BookDetailResponse, error) {
	var bookDetail BookDetailResponse
	if err := json.Unmarshal(response, &bookDetail); err != nil {
		return nil, fmt.Errorf("書籍詳細レスポンス解析失敗: %w", err)
	}
	return &bookDetail, nil
}
