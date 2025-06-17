package browser

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"net/http"

	"golang.org/x/net/html"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/generated/api"
)

// GetBookDetails retrieves book details and table of contents from O'Reilly book Product ID
func (bc *BrowserClient) GetBookDetails(productID string) (*BookDetailResponse, error) {
	log.Printf("プロダクトIDから書籍詳細を取得しています: %s", productID)

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
	log.Printf("チャプター本文を取得しています: %s/%s", productID, chapterName)

	// Step 1: Get chapter metadata first
	metadata, _, err := bc.getChapterMetadata(productID, chapterName)
	if err != nil {
		return nil, fmt.Errorf("チャプターメタデータ取得失敗: %w", err)
	}

	// Extract chapter title from metadata
	chapterTitle := chapterName
	if title, ok := metadata["title"].(string); ok && title != "" {
		chapterTitle = title
	}

	// Step 2: Get raw HTML content from API
	htmlContent, contentURL, err := bc.GetChapterHTMLContent(productID, chapterName)
	if err != nil {
		return nil, fmt.Errorf("チャプターHTML取得失敗: %w", err)
	}

	// Parse HTML content into structured format
	parsedContent, err := bc.parseHTMLContent(htmlContent)
	if err != nil {
		return nil, fmt.Errorf("HTML解析失敗: %w", err)
	}

	// Use parsed title if available, otherwise use metadata title
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
		Metadata: map[string]interface{}{
			"extraction_method": "html_parsing",
			"processed_at":     time.Now().UTC().Format(time.RFC3339),
			"word_count":       countWords(parsedContent.Paragraphs),
			"book_title":       metadata["book_title"],
			"minutes_required": metadata["minutes_required"],
			"virtual_pages":    metadata["virtual_pages"],
		},
	}

	log.Printf("チャプター本文取得に成功しました: %s (%d paragraphs, %d headings, %d code blocks)",
		chapterTitle, len(parsedContent.Paragraphs), len(parsedContent.Headings), len(parsedContent.CodeBlocks))

	return response, nil
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

	// Make a raw HTTP request to see the actual response structure
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

// GetChapterHTMLContent retrieves actual HTML content from O'Reilly API via 2-step process
func (bc *BrowserClient) GetChapterHTMLContent(productID, chapterName string) (string, string, error) {
	// Step 1: Get chapter metadata 
	metadata, _, err := bc.getChapterMetadata(productID, chapterName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get chapter metadata: %w", err)
	}

	// Step 2: Get actual HTML content from the content URL
	contentURL, ok := metadata["content"].(string)
	if !ok {
		return "", "", fmt.Errorf("content URL not found in chapter metadata")
	}

	htmlContent, err := bc.getContentFromURL(contentURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to get HTML content from %s: %w", contentURL, err)
	}

	log.Printf("チャプターHTML取得に成功しました: %s (%d bytes)", contentURL, len(htmlContent))
	return htmlContent, contentURL, nil
}

// getChapterMetadata retrieves chapter metadata (JSON) from O'Reilly API
func (bc *BrowserClient) getChapterMetadata(productID, chapterName string) (map[string]interface{}, string, error) {
	// Only try the metadata endpoint that actually works
	endpoint := fmt.Sprintf("/api/v1/book/%s/chapter/%s.html", productID, chapterName)
	fullURL := APIEndpointBase + endpoint
	
	log.Printf("チャプターメタデータAPIを試行しています: %s", endpoint)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for JSON response
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", bc.userAgent)

	// Add cookies
	for _, cookie := range bc.cookies {
		req.AddCookie(cookie)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &metadata); err != nil {
		return nil, "", fmt.Errorf("failed to parse JSON metadata: %w", err)
	}

	log.Printf("チャプターメタデータ取得に成功しました: %s", endpoint)
	return metadata, fullURL, nil
}

// getContentFromURL retrieves HTML content from the specified URL with authentication
func (bc *BrowserClient) getContentFromURL(contentURL string) (string, error) {
	log.Printf("HTMLコンテンツを取得しています: %s", contentURL)

	req, err := http.NewRequest("GET", contentURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for HTML response (try different accept headers)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", bc.userAgent)

	// Add authentication cookies
	for _, cookie := range bc.cookies {
		req.AddCookie(cookie)
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("content request failed with status %d", resp.StatusCode)
	}

	// Handle gzip compression
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read content body: %w", err)
	}

	return string(bodyBytes), nil
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

// parseHTMLNode recursively parses HTML nodes
func parseHTMLNode(n *html.Node, content *ParsedChapterContent, depth int) {
	if n.Type == html.ElementNode {
		switch strings.ToLower(n.Data) {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			heading := parseHeading(n)
			if heading.Text != "" {
				content.Headings = append(content.Headings, heading)
			}

		case "p":
			text := extractTextContent(n)
			if strings.TrimSpace(text) != "" {
				content.Paragraphs = append(content.Paragraphs, strings.TrimSpace(text))
			}

		case "pre", "code":
			if n.Data == "pre" || hasClass(n, "highlight") || hasClass(n, "code") {
				codeBlock := parseCodeBlock(n)
				if codeBlock.Code != "" {
					content.CodeBlocks = append(content.CodeBlocks, codeBlock)
				}
			}

		case "img":
			img := parseImage(n)
			if img.Src != "" {
				content.Images = append(content.Images, img)
			}

		case "a":
			link := parseLink(n)
			if link.Href != "" && link.Text != "" {
				content.Links = append(content.Links, link)
			}
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
			Content: []interface{}{},
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
