package browser

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
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
	parsedContent, err := bc.parseHTMLContent(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("HTML解析失敗: %w", err)
	}

	// Use parsed title if available, otherwise use TOC title
	if parsedContent.Title != "" {
		chapterTitle = parsedContent.Title
	} else if len(parsedContent.Headings) > 0 && parsedContent.Headings[0].Text != "" {
		chapterTitle = parsedContent.Headings[0].Text
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
			"word_count":        countWords(parsedContent.Paragraphs),
		},
	}

	slog.Info("チャプター本文取得に成功しました",
		"title", chapterTitle,
		"paragraph_count", len(parsedContent.Paragraphs),
		"heading_count", len(parsedContent.Headings),
		"code_block_count", len(parsedContent.CodeBlocks))

	return response, nil
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

// getChapterHrefFromTOC retrieves chapter href URL from flat-toc
func (bc *BrowserClient) getChapterHrefFromTOC(productID, chapterName string) (string, error) {
	slog.Debug("flat-tocからチャプターhrefを取得しています", "product_id", productID, "chapter_name", chapterName)

	// Get flat-toc for the book
	toc, err := bc.getBookTOC(productID)
	if err != nil {
		return "", fmt.Errorf("failed to get book TOC: %w", err)
	}

	// Search for chapter by name in TOC items
	for _, item := range toc.TableOfContents {
		// Match by exact ID or by href containing the chapter name
		if item.ID == chapterName ||
			strings.Contains(item.Href, chapterName) ||
			strings.HasSuffix(item.Href, chapterName+".html") ||
			strings.HasSuffix(item.Href, chapterName+".xhtml") {

			// Convert relative href to full URL if needed
			href := item.Href
			if strings.HasPrefix(href, "/") {
				// Absolute path - add base URL
				href = APIEndpointBase + href
			} else if !strings.HasPrefix(href, "http") {
				// Relative path - construct full URL
				href = APIEndpointBase + "/api/v2/epubs/urn:orm:book:" + productID + "/files/" + href
			}

			slog.Debug("チャプターhref取得に成功しました", "chapter_name", chapterName, "href", href)
			return href, nil
		}
	}

	// If not found by exact match, try partial matching
	var bestMatch *TableOfContentsItem
	for _, item := range toc.TableOfContents {
		// Check if the chapter name appears anywhere in the href or ID
		if strings.Contains(strings.ToLower(item.Href), strings.ToLower(chapterName)) ||
			strings.Contains(strings.ToLower(item.ID), strings.ToLower(chapterName)) {
			bestMatch = &item
			break
		}
	}

	if bestMatch != nil {
		href := bestMatch.Href
		if strings.HasPrefix(href, "/") {
			href = APIEndpointBase + href
		} else if !strings.HasPrefix(href, "http") {
			href = APIEndpointBase + "/api/v2/epubs/urn:orm:book:" + productID + "/files/" + href
		}

		slog.Debug("部分マッチでチャプターhref取得", "chapter_name", chapterName, "href", href)
		return href, nil
	}

	return "", fmt.Errorf("chapter '%s' not found in TOC for book %s", chapterName, productID)
}

// getChapterTitleFromTOC retrieves chapter title from flat-toc
func (bc *BrowserClient) getChapterTitleFromTOC(productID, chapterName string) (string, error) {
	// Get flat-toc for the book
	toc, err := bc.getBookTOC(productID)
	if err != nil {
		return "", fmt.Errorf("failed to get book TOC: %w", err)
	}

	// Search for chapter by name in TOC items
	for _, item := range toc.TableOfContents {
		// Match by exact ID or by href containing the chapter name
		if item.ID == chapterName ||
			strings.Contains(item.Href, chapterName) ||
			strings.HasSuffix(item.Href, chapterName+".html") ||
			strings.HasSuffix(item.Href, chapterName+".xhtml") {
			return item.Title, nil
		}
	}

	// If not found by exact match, try partial matching
	for _, item := range toc.TableOfContents {
		// Check if the chapter name appears anywhere in the href or ID
		if strings.Contains(strings.ToLower(item.Href), strings.ToLower(chapterName)) ||
			strings.Contains(strings.ToLower(item.ID), strings.ToLower(chapterName)) {
			return item.Title, nil
		}
	}

	return "", fmt.Errorf("chapter '%s' not found in TOC for book %s", chapterName, productID)
}

// parseHTMLContent parses HTML content into structured format
func (bc *BrowserClient) parseHTMLContent(htmlContent string) (*ParsedChapterContent, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("HTML parsing failed: %w", err)
	}

	content := &ParsedChapterContent{
		Sections:   []ContentSection{},
		Paragraphs: []string{},
		Headings:   []ContentHeading{},
		CodeBlocks: []CodeBlock{},
		Images:     []ImageReference{},
		Links:      []LinkReference{},
	}

	// Extract title from document
	content.Title = extractTitle(doc)

	// Parse the HTML tree
	parseHTMLNode(doc, content, 0)

	// Organize content into sections
	content.Sections = organizeSections(content.Headings, content.Paragraphs, content.CodeBlocks, content.Images)

	return content, nil
}

// handleHeadingNode processes heading elements (h1-h6).
func handleHeadingNode(n *html.Node, content *ParsedChapterContent) {
	heading := parseHeading(n)
	if heading.Text != "" {
		content.Headings = append(content.Headings, heading)
	}
}

// handleParagraphNode processes paragraph elements.
func handleParagraphNode(n *html.Node, content *ParsedChapterContent) {
	text := strings.TrimSpace(extractTextContent(n))
	if text != "" {
		content.Paragraphs = append(content.Paragraphs, text)
	}
}

// handleCodeNode processes code block elements (pre, code).
func handleCodeNode(n *html.Node, content *ParsedChapterContent) {
	if n.Data == "pre" || hasClass(n, "highlight") || hasClass(n, "code") {
		codeBlock := parseCodeBlock(n)
		if codeBlock.Code != "" {
			content.CodeBlocks = append(content.CodeBlocks, codeBlock)
		}
	}
}

// handleImageNode processes image elements.
func handleImageNode(n *html.Node, content *ParsedChapterContent) {
	img := parseImage(n)
	if img.Src != "" {
		content.Images = append(content.Images, img)
	}
}

// handleLinkNode processes link elements.
func handleLinkNode(n *html.Node, content *ParsedChapterContent) {
	link := parseLink(n)
	if link.Href != "" && link.Text != "" {
		content.Links = append(content.Links, link)
	}
}

// parseHTMLNode recursively parses HTML nodes
func parseHTMLNode(n *html.Node, content *ParsedChapterContent, depth int) {
	if n.Type == html.ElementNode {
		switch strings.ToLower(n.Data) {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			handleHeadingNode(n, content)
		case "p":
			handleParagraphNode(n, content)
		case "pre", "code":
			handleCodeNode(n, content)
		case "img":
			handleImageNode(n, content)
		case "a":
			handleLinkNode(n, content)
		}
	}

	// Recursively parse child nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseHTMLNode(c, content, depth+1)
	}
}

// extractTitle extracts the title from HTML document
func extractTitle(doc *html.Node) string {
	var title string

	var findTitle func(*html.Node)
	findTitle = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "title":
				title = extractTextContent(n)
				return
			case "h1":
				if title == "" {
					title = extractTextContent(n)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findTitle(c)
			if title != "" {
				return
			}
		}
	}

	findTitle(doc)
	return strings.TrimSpace(title)
}

// parseHeading parses heading elements
func parseHeading(n *html.Node) ContentHeading {
	level := 1
	if len(n.Data) == 2 && n.Data[0] == 'h' {
		switch n.Data[1] {
		case '1':
			level = 1
		case '2':
			level = 2
		case '3':
			level = 3
		case '4':
			level = 4
		case '5':
			level = 5
		case '6':
			level = 6
		}
	}

	heading := ContentHeading{
		Level: level,
		Text:  extractTextContent(n),
		ID:    getAttr(n, "id"),
	}

	return heading
}

// parseCodeBlock parses code block elements
func parseCodeBlock(n *html.Node) CodeBlock {
	code := extractTextContent(n)
	language := ""
	caption := ""

	// Try to extract language from class attribute
	class := getAttr(n, "class")
	if class != "" {
		// Look for language patterns like "language-go", "highlight-go", etc.
		re := regexp.MustCompile(`(?:language-|highlight-)(\w+)`)
		matches := re.FindStringSubmatch(class)
		if len(matches) > 1 {
			language = matches[1]
		}
	}

	// Look for captions in nearby elements (common in O'Reilly books)
	if n.Parent != nil && n.Parent.NextSibling != nil {
		if n.Parent.NextSibling.Type == html.ElementNode &&
			(strings.ToLower(n.Parent.NextSibling.Data) == "p" || hasClass(n.Parent.NextSibling, "caption")) {
			captionText := extractTextContent(n.Parent.NextSibling)
			if strings.Contains(strings.ToLower(captionText), "example") ||
				strings.Contains(strings.ToLower(captionText), "listing") {
				caption = strings.TrimSpace(captionText)
			}
		}
	}

	return CodeBlock{
		Language: language,
		Code:     strings.TrimSpace(code),
		Caption:  caption,
	}
}

// parseImage parses image elements
func parseImage(n *html.Node) ImageReference {
	return ImageReference{
		Src:     getAttr(n, "src"),
		Alt:     getAttr(n, "alt"),
		Caption: "", // Caption extraction would need more complex logic
	}
}

// parseLink parses link elements
func parseLink(n *html.Node) LinkReference {
	href := getAttr(n, "href")
	text := extractTextContent(n)
	linkType := "internal"

	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		linkType = "external"
	} else if strings.HasPrefix(href, "#") {
		linkType = "anchor"
	}

	return LinkReference{
		Href: href,
		Text: strings.TrimSpace(text),
		Type: linkType,
	}
}

// organizeSections organizes content into sections based on headings
func organizeSections(headings []ContentHeading, paragraphs []string, codeBlocks []CodeBlock, images []ImageReference) []ContentSection {
	sections := []ContentSection{}

	// For now, create a simple structure with one section per heading
	// More sophisticated organization could be implemented later
	for _, heading := range headings {
		section := ContentSection{
			Heading: heading,
			Content: []any{},
		}

		// This is a simplified implementation - in practice, you'd need to
		// associate content with the correct headings based on document structure
		sections = append(sections, section)
	}

	return sections
}

// Utility functions

// extractTextContent extracts all text content from a node and its children
func extractTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(extractTextContent(c))
	}

	return text.String()
}

// getAttr gets an attribute value from a node
func getAttr(n *html.Node, attrName string) string {
	for _, attr := range n.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

// hasClass checks if a node has a specific CSS class
func hasClass(n *html.Node, className string) bool {
	class := getAttr(n, "class")
	if class == "" {
		return false
	}

	classes := strings.Split(class, " ")
	for _, c := range classes {
		if strings.TrimSpace(c) == className {
			return true
		}
	}
	return false
}

// countWords counts words in a slice of paragraphs
func countWords(paragraphs []string) int {
	totalWords := 0
	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		totalWords += len(words)
	}
	return totalWords
}
