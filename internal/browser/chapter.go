package browser

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/htmlparse"
)

// GetBookChapterContent retrieves and parses chapter content from O'Reilly book
func (bc *BrowserClient) GetBookChapterContent(productID, chapterName string) (*ChapterContentResponse, error) {
	slog.Info("チャプター本文を取得しています", "product_id", productID, "chapter_name", chapterName)

	// Step 1: Get chapter title from TOC
	chapterTitle, err := bc.getChapterTitleFromTOC(productID, chapterName)
	if err != nil {
		slog.Warn("TOCからタイトル取得に失敗、チャプター名を使用", "error", err, "chapter_name", chapterName)
		chapterTitle = chapterName
	}

	// Step 2: Get raw HTML content from API via flat-toc
	htmlContent, contentURL, err := bc.GetChapterHTMLContent(productID, chapterName)
	if err != nil {
		return nil, fmt.Errorf("チャプターHTML取得失敗: %w", err)
	}

	// Parse HTML content into structured format
	parsedContent, err := htmlparse.ParseHTMLContent(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("HTML解析失敗: %w", err)
	}

	// Use parsed title if available, otherwise use TOC title
	if parsedContent.Title != "" {
		chapterTitle = parsedContent.Title
	} else if len(parsedContent.Sections) > 0 && parsedContent.Sections[0].Heading.Text != "" {
		chapterTitle = parsedContent.Sections[0].Heading.Text
	}

	response := &ChapterContentResponse{
		BookID:       productID,
		ChapterName:  chapterName,
		ChapterTitle: chapterTitle,
		Content:      *parsedContent,
		SourceURL:    contentURL,
		Metadata: map[string]any{
			"extraction_method": "flat_toc_lookup",
			"processed_at":      time.Now().UTC().Format(time.RFC3339),
			"word_count":        htmlparse.CountWordsFromSections(parsedContent.Sections),
		},
	}

	slog.Info("チャプター本文取得に成功しました",
		"title", chapterTitle,
		"section_count", len(parsedContent.Sections))

	return response, nil
}

// GetChapterHTMLContent retrieves actual HTML content from O'Reilly API via flat-toc lookup
func (bc *BrowserClient) GetChapterHTMLContent(productID, chapterName string) (string, string, error) {
	// Step 1: Get chapter href from flat-toc
	chapterHref, err := bc.getChapterHrefFromTOC(productID, chapterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get chapter href from TOC: %w", err)
	}

	// Step 2: Get actual HTML content from the href URL
	htmlContent, err := bc.GetContentFromURL(chapterHref)
	if err != nil {
		return "", "", fmt.Errorf("failed to get HTML content from %s: %w", chapterHref, err)
	}

	slog.Debug("チャプターHTML取得に成功しました", "href", chapterHref, "content_size", len(htmlContent))
	return htmlContent, chapterHref, nil
}

// findTOCItem searches the book's TOC for a matching chapter by exact or partial match.
func (bc *BrowserClient) findTOCItem(productID, chapterName string) (*TableOfContentsItem, error) {
	toc, err := bc.getBookTOC(productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book TOC: %w", err)
	}

	// Exact match: ID or href containing the chapter name
	for _, item := range toc.TableOfContents {
		if item.ID == chapterName ||
			strings.Contains(item.Href, chapterName) ||
			strings.HasSuffix(item.Href, chapterName+".html") ||
			strings.HasSuffix(item.Href, chapterName+".xhtml") {
			return &item, nil
		}
	}

	// Partial match: case-insensitive search in href or ID
	for _, item := range toc.TableOfContents {
		if strings.Contains(strings.ToLower(item.Href), strings.ToLower(chapterName)) ||
			strings.Contains(strings.ToLower(item.ID), strings.ToLower(chapterName)) {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("chapter '%s' not found in TOC for book %s", chapterName, productID)
}

// resolveChapterHref converts a TOC item's href to a full URL.
func resolveChapterHref(href, productID string) string {
	if strings.HasPrefix(href, "/") {
		return APIEndpointBase + href
	}
	if !strings.HasPrefix(href, "http") {
		return APIEndpointBase + "/api/v2/epubs/urn:orm:book:" + productID + "/files/" + href
	}
	return href
}

// getChapterHrefFromTOC retrieves chapter href URL from flat-toc
func (bc *BrowserClient) getChapterHrefFromTOC(productID, chapterName string) (string, error) {
	slog.Debug("flat-tocからチャプターhrefを取得しています", "product_id", productID, "chapter_name", chapterName)

	item, err := bc.findTOCItem(productID, chapterName)
	if err != nil {
		return "", err
	}

	href := resolveChapterHref(item.Href, productID)
	slog.Debug("チャプターhref取得に成功しました", "chapter_name", chapterName, "href", href)
	return href, nil
}

// getChapterTitleFromTOC retrieves chapter title from flat-toc
func (bc *BrowserClient) getChapterTitleFromTOC(productID, chapterName string) (string, error) {
	item, err := bc.findTOCItem(productID, chapterName)
	if err != nil {
		return "", err
	}
	return item.Title, nil
}
