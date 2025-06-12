package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// BookDetailAPI endpoints with fallback strategy
const (
	BookAPIV1       = "/api/v1/book/%s/"
	BookAPIV2       = "/api/v2/products/%s/"
	BookChaptersAPI = "/api/v1/book/%s/chapters/"
	BookTOCAPI      = "/api/v2/products/%s/toc/"
	BookStructureAPI = "/learningapi/v1/book/%s/structure/"
)

// GetBookDetails retrieves comprehensive book metadata from O'Reilly API
func (bc *BrowserClient) GetBookDetails(productID string) (*BookDetailResponse, error) {
	log.Printf("書籍詳細を取得しています: %s", productID)
	
	endpoints := []string{
		fmt.Sprintf("https://learning.oreilly.com"+BookAPIV1, productID),
		fmt.Sprintf("https://learning.oreilly.com"+BookAPIV2, productID),
	}
	
	for _, endpoint := range endpoints {
		log.Printf("書籍詳細APIを試行しています: %s", endpoint)
		response, err := bc.makeHTTPRequest("GET", endpoint, nil)
		if err == nil {
			bookDetail, parseErr := parseBookDetailResponse(response)
			if parseErr == nil {
				log.Printf("書籍詳細取得に成功しました: %s", bookDetail.Title)
				return bookDetail, nil
			}
			log.Printf("書籍詳細レスポンス解析エラー: %v", parseErr)
		}
		log.Printf("書籍詳細API失敗: %s, エラー: %v", endpoint, err)
	}
	
	return nil, fmt.Errorf("全ての書籍詳細APIエンドポイントが失敗しました: %s", productID)
}

// GetBookTableOfContents retrieves table of contents for a book
func (bc *BrowserClient) GetBookTableOfContents(productID string) (*TableOfContentsResponse, error) {
	log.Printf("書籍目次を取得しています: %s", productID)
	
	// Try TOC API endpoints
	tocEndpoints := []string{
		fmt.Sprintf("https://learning.oreilly.com"+BookChaptersAPI, productID),
		fmt.Sprintf("https://learning.oreilly.com"+BookTOCAPI, productID),
		fmt.Sprintf("https://learning.oreilly.com"+BookStructureAPI, productID),
	}
	
	for _, endpoint := range tocEndpoints {
		log.Printf("目次APIを試行しています: %s", endpoint)
		response, err := bc.makeHTTPRequest("GET", endpoint, nil)
		if err == nil {
			tocResponse, parseErr := parseTOCResponse(response)
			if parseErr == nil {
				log.Printf("目次取得に成功しました: %d章", len(tocResponse.TableOfContents))
				return tocResponse, nil
			}
			log.Printf("目次レスポンス解析エラー: %v", parseErr)
		}
		log.Printf("目次API失敗: %s, エラー: %v", endpoint, err)
	}
	
	// Fallback to DOM scraping
	log.Printf("API手法が失敗したため、DOM抽出を試行します: %s", productID)
	return bc.extractTOCFromDOM(productID)
}

// SearchBookByTitle searches for a book by title and returns the first match
func (bc *BrowserClient) SearchBookByTitle(title string) (*BookSearchResult, error) {
	log.Printf("書籍タイトルで検索しています: %s", title)
	
	// Use existing search functionality
	searchOptions := map[string]interface{}{
		"rows": 10,
		"languages": []string{"en", "ja"},
	}
	
	results, err := bc.SearchContent(title, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("書籍検索失敗: %w", err)
	}
	
	if len(results) == 0 {
		return nil, fmt.Errorf("書籍が見つかりません: %s", title)
	}
	
	// Find the best match (exact title or first result)
	for _, result := range results {
		resultTitle := getStringFromMap(result, "title")
		if strings.Contains(strings.ToLower(resultTitle), strings.ToLower(title)) {
			productID := extractProductIDFromSearchResult(result)
			if productID != "" {
				return &BookSearchResult{
					ProductID:   productID,
					Title:       resultTitle,
					Description: getStringFromMap(result, "description"),
					URL:         getStringFromMap(result, "url"),
					Authors:     getStringArrayFromMap(result, "authors"),
					Publisher:   getStringFromMap(result, "publisher"),
				}, nil
			}
		}
	}
	
	// If no exact match, use first result
	firstResult := results[0]
	productID := extractProductIDFromSearchResult(firstResult)
	if productID == "" {
		return nil, fmt.Errorf("検索結果からプロダクトIDを抽出できませんでした")
	}
	
	return &BookSearchResult{
		ProductID:   productID,
		Title:       getStringFromMap(firstResult, "title"),
		Description: getStringFromMap(firstResult, "description"),
		URL:         getStringFromMap(firstResult, "url"),
		Authors:     getStringArrayFromMap(firstResult, "authors"),
		Publisher:   getStringFromMap(firstResult, "publisher"),
	}, nil
}

// GetBookDetailsByURL retrieves book details and table of contents from O'Reilly book URL
func (bc *BrowserClient) GetBookDetailsByURL(bookURL string) (*BookOverviewAndTOCResponse, error) {
	log.Printf("URLから書籍詳細と目次を取得しています: %s", bookURL)
	
	// Extract product ID from URL
	productID := extractProductIDFromURL(bookURL)
	if productID == "" {
		return nil, fmt.Errorf("URLからプロダクトIDを抽出できませんでした: %s", bookURL)
	}
	
	log.Printf("抽出されたプロダクトID: %s", productID)
	
	// Get book details from API
	bookDetail, err := bc.GetBookDetails(productID)
	if err != nil {
		log.Printf("書籍詳細API取得失敗、DOM抽出にフォールバック: %v", err)
		// Fallback to DOM extraction
		bookDetail, err = bc.extractBookDetailFromDOM(bookURL)
		if err != nil {
			return nil, fmt.Errorf("書籍詳細取得失敗: %w", err)
		}
	}
	
	// Get table of contents
	toc, err := bc.GetBookTableOfContents(productID)
	if err != nil {
		log.Printf("目次API取得失敗、DOM抽出にフォールバック: %v", err)
		// Fallback to DOM extraction
		toc, err = bc.extractTOCFromDOM(productID)
		if err != nil {
			log.Printf("目次DOM抽出も失敗: %v", err)
			// Create empty TOC as last resort
			toc = &TableOfContentsResponse{
				BookID:          productID,
				BookTitle:       bookDetail.Title,
				TableOfContents: []TableOfContentsItem{},
				TotalChapters:   0,
				Metadata: map[string]interface{}{
					"extraction_method": "failed",
					"original_url": bookURL,
				},
			}
		}
	}
	
	return &BookOverviewAndTOCResponse{
		BookDetail:      *bookDetail,
		TableOfContents: *toc,
	}, nil
}

// extractBookDetailFromDOM extracts book details from the book page DOM
func (bc *BrowserClient) extractBookDetailFromDOM(bookURL string) (*BookDetailResponse, error) {
	log.Printf("DOMから書籍詳細を抽出しています: %s", bookURL)
	
	var bookDetail BookDetailResponse
	
	err := chromedp.Run(bc.ctx,
		chromedp.Navigate(bookURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Extract book title
			var title string
			chromedp.Evaluate(`
				document.querySelector('h1, .book-title, [data-testid*="title"]')?.textContent?.trim() || document.title
			`, &title).Do(ctx)
			
			// Extract description
			var description string
			chromedp.Evaluate(`
				document.querySelector('.description, .book-description, .summary, [data-testid*="description"]')?.innerHTML || 
				document.querySelector('.description, .book-description, .summary, [data-testid*="description"]')?.textContent || ''
			`, &description).Do(ctx)
			
			// Extract authors
			var authors []string
			chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.author, .book-author, [data-testid*="author"]')).map(el => el.textContent?.trim()).filter(a => a)
			`, &authors).Do(ctx)
			
			// Extract publisher
			var publisher string
			chromedp.Evaluate(`
				document.querySelector('.publisher, .book-publisher, [data-testid*="publisher"]')?.textContent?.trim() || ''
			`, &publisher).Do(ctx)
			
			bookDetail = BookDetailResponse{
				ID:          extractProductIDFromURL(bookURL),
				Title:       title,
				Description: description,
				Authors:     authors,
				Publishers:  []string{publisher},
				Metadata: map[string]interface{}{
					"extraction_method": "dom",
					"original_url": bookURL,
				},
			}
			
			return nil
		}),
	)
	
	if err != nil {
		return nil, fmt.Errorf("DOM書籍詳細抽出失敗: %w", err)
	}
	
	return &bookDetail, nil
}

// Helper functions

// extractProductIDFromSearchResult extracts product ID from search result
func extractProductIDFromSearchResult(result map[string]interface{}) string {
	// Try different possible ID fields
	if id := getStringFromMap(result, "id"); id != "" {
		return id
	}
	if productID := getStringFromMap(result, "product_id"); productID != "" {
		return productID
	}
	if isbn := getStringFromMap(result, "isbn"); isbn != "" {
		return isbn
	}
	
	// Extract from URL
	url := getStringFromMap(result, "url")
	if url != "" {
		return extractProductIDFromURL(url)
	}
	
	return ""
}

// extractProductIDFromURL extracts product ID from O'Reilly URL
func extractProductIDFromURL(url string) string {
	// Pattern: /library/view/{title}/{product_id}/
	re := regexp.MustCompile(`/library/view/[^/]+/([^/\?]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		productID := matches[1]
		// Remove any trailing slash or parameters
		if idx := strings.Index(productID, "/"); idx != -1 {
			productID = productID[:idx]
		}
		return productID
	}
	
	// Alternative pattern: /library/view/-/{product_id}/
	re2 := regexp.MustCompile(`/library/view/-/([^/\?]+)`)
	matches2 := re2.FindStringSubmatch(url)
	if len(matches2) > 1 {
		productID := matches2[1]
		// Remove any trailing slash or parameters
		if idx := strings.Index(productID, "/"); idx != -1 {
			productID = productID[:idx]
		}
		return productID
	}
	
	return ""
}

// parseBookDetailResponse parses book detail API response
func parseBookDetailResponse(response []byte) (*BookDetailResponse, error) {
	var bookDetail BookDetailResponse
	if err := json.Unmarshal(response, &bookDetail); err != nil {
		return nil, fmt.Errorf("書籍詳細レスポンス解析失敗: %w", err)
	}
	return &bookDetail, nil
}

// parseTOCResponse parses table of contents API response
func parseTOCResponse(response []byte) (*TableOfContentsResponse, error) {
	var tocResponse TableOfContentsResponse
	if err := json.Unmarshal(response, &tocResponse); err != nil {
		return nil, fmt.Errorf("目次レスポンス解析失敗: %w", err)
	}
	return &tocResponse, nil
}

// extractTOCFromDOM extracts table of contents from DOM as fallback
func (bc *BrowserClient) extractTOCFromDOM(productID string) (*TableOfContentsResponse, error) {
	log.Printf("DOM抽出による目次取得を試行しています: %s", productID)
	
	url := fmt.Sprintf("https://learning.oreilly.com/library/view/-/%s/", productID)
	
	var tocItems []TableOfContentsItem
	
	err := chromedp.Run(bc.ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Extract TOC from DOM
			var tocElements []interface{}
			err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.toc, .contents, .chapter-list, [data-testid*="toc"]')).map(el => {
					const items = Array.from(el.querySelectorAll('a, li, .chapter')).map((item, index) => {
						const titleEl = item.querySelector('a') || item;
						const title = titleEl.textContent?.trim();
						const href = titleEl.href || '';
						const level = item.tagName === 'H1' ? 1 : item.tagName === 'H2' ? 2 : item.tagName === 'H3' ? 3 : 1;
						
						return {
							id: 'toc_' + index,
							title: title,
							href: href,
							level: level
						};
					}).filter(item => item.title && item.title.length > 0);
					
					return items;
				}).flat()
			`, &tocElements).Do(ctx)
			
			if err != nil {
				return err
			}
			
			// Convert to TableOfContentsItem
			for _, element := range tocElements {
				if elementMap, ok := element.(map[string]interface{}); ok {
					tocItem := TableOfContentsItem{
						ID:       getStringFromMap(elementMap, "id"),
						Title:    getStringFromMap(elementMap, "title"),
						Href:     getStringFromMap(elementMap, "href"),
						Level:    getIntFromMap(elementMap, "level"),
						Metadata: elementMap,
					}
					tocItems = append(tocItems, tocItem)
				}
			}
			
			return nil
		}),
	)
	
	if err != nil {
		return nil, fmt.Errorf("DOM目次抽出失敗: %w", err)
	}
	
	return &TableOfContentsResponse{
		BookID:          productID,
		BookTitle:       fmt.Sprintf("Book %s", productID),
		TableOfContents: tocItems,
		TotalChapters:   len(tocItems),
		Metadata: map[string]interface{}{
			"extraction_method": "dom",
			"url": url,
		},
	}, nil
}