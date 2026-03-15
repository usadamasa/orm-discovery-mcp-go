package browser

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
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
	parsedContent, err := bc.parseHTMLContent(htmlContent)
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
			"word_count":        countWordsFromSections(parsedContent.Sections),
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

// sectionBuilder tracks the current section while walking the DOM tree.
type sectionBuilder struct {
	sections []ContentSection
	current  *ContentSection
}

// startSection begins a new section with the given heading.
func (sb *sectionBuilder) startSection(heading ContentHeading) {
	sb.flush()
	sb.current = &ContentSection{
		Heading: heading,
		Content: []any{},
	}
}

// appendContent adds a content element to the current section.
// If no section exists yet, a preamble section (empty heading) is created.
func (sb *sectionBuilder) appendContent(elem any) {
	if sb.current == nil {
		sb.current = &ContentSection{
			Heading: ContentHeading{},
			Content: []any{},
		}
	}
	sb.current.Content = append(sb.current.Content, elem)
}

// flush saves the current section to the sections slice.
func (sb *sectionBuilder) flush() {
	if sb.current != nil {
		sb.sections = append(sb.sections, *sb.current)
		sb.current = nil
	}
}

// build returns the final list of sections, filtering out empty preamble sections.
func (sb *sectionBuilder) build() []ContentSection {
	sb.flush()
	result := make([]ContentSection, 0, len(sb.sections))
	for _, s := range sb.sections {
		// Filter out empty preamble sections (empty heading + no content)
		if s.Heading.Text == "" && len(s.Content) == 0 {
			continue
		}
		result = append(result, s)
	}
	return result
}

// parseHTMLContent parses HTML content into structured format
func (bc *BrowserClient) parseHTMLContent(htmlContent string) (*ParsedChapterContent, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("HTML parsing failed: %w", err)
	}

	content := &ParsedChapterContent{}
	content.Title = extractTitle(doc)

	builder := &sectionBuilder{}
	walkDOM(doc, builder)
	content.Sections = builder.build()

	return content, nil
}

// walkDOM walks the DOM tree and populates the sectionBuilder.
func walkDOM(n *html.Node, sb *sectionBuilder) {
	if n.Type == html.ElementNode {
		if handleElement(n, sb) {
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkDOM(c, sb)
	}
}

// handleElement processes a single HTML element node.
// Returns true if the element was handled (no further recursion needed).
func handleElement(n *html.Node, sb *sectionBuilder) bool {
	switch strings.ToLower(n.Data) {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		heading := parseHeading(n)
		if heading.Text != "" {
			sb.startSection(heading)
		}
		return true
	case "p":
		text := strings.TrimSpace(extractTextContent(n))
		if text != "" {
			sb.appendContent(ParagraphElement{Type: "paragraph", Text: text})
		}
		return true
	case "pre":
		cb := parseCodeBlock(n)
		if cb.Code != "" {
			sb.appendContent(cb)
		}
		return true
	case "img":
		img := parseImage(n)
		if img.Src != "" {
			sb.appendContent(img)
		}
		return true
	case "ul", "ol":
		le := parseList(n)
		if len(le.Items) > 0 {
			sb.appendContent(le)
		}
		return true
	case "a":
		link := parseLinkElement(n)
		if link.Href != "" && link.Text != "" {
			sb.appendContent(link)
		}
		return true
	default:
		return false
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

// parseCodeBlock parses code block elements into CodeBlockElement.
func parseCodeBlock(n *html.Node) CodeBlockElement {
	code := extractTextContent(n)
	language := ""

	// Try to extract language from class attribute of pre or child code element
	class := getAttr(n, "class")
	if class == "" {
		// Check child code element for language class
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "code" {
				class = getAttr(c, "class")
				break
			}
		}
	}
	if class != "" {
		re := regexp.MustCompile(`(?:language-|highlight-)(\w+)`)
		matches := re.FindStringSubmatch(class)
		if len(matches) > 1 {
			language = matches[1]
		}
	}

	return CodeBlockElement{
		Type:     "code_block",
		Language: language,
		Code:     strings.TrimSpace(code),
	}
}

// parseImage parses image elements into ImageElement.
func parseImage(n *html.Node) ImageElement {
	return ImageElement{
		Type: "image",
		Src:  getAttr(n, "src"),
		Alt:  getAttr(n, "alt"),
	}
}

// parseList parses ul/ol elements into ListElement.
func parseList(n *html.Node) ListElement {
	ordered := strings.ToLower(n.Data) == "ol"
	var items []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && strings.ToLower(c.Data) == "li" {
			text := strings.TrimSpace(extractTextContent(c))
			if text != "" {
				items = append(items, text)
			}
		}
	}
	return ListElement{
		Type:    "list",
		Ordered: ordered,
		Items:   items,
	}
}

// parseLinkElement parses standalone link elements into LinkElement.
func parseLinkElement(n *html.Node) LinkElement {
	href := getAttr(n, "href")
	text := strings.TrimSpace(extractTextContent(n))
	linkType := "internal"

	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		linkType = "external"
	} else if strings.HasPrefix(href, "#") {
		linkType = "anchor"
	}

	return LinkElement{
		Type:     "link",
		Href:     href,
		Text:     text,
		LinkType: linkType,
	}
}

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

// countWordsFromSections counts words across all paragraph elements in sections.
func countWordsFromSections(sections []ContentSection) int {
	totalWords := 0
	for _, section := range sections {
		for _, item := range section.Content {
			if p, ok := item.(ParagraphElement); ok {
				totalWords += len(strings.Fields(p.Text))
			}
		}
	}
	return totalWords
}
