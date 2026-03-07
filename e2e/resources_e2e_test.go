//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMCPResource_BookDetails tests the book-details resource with real API.
func TestMCPResource_BookDetails(t *testing.T) {
	client := GetSharedClient()

	// Get book details for test book
	details, err := client.GetBookDetails(TestBookID)
	if err != nil {
		t.Fatalf("GetBookDetails failed: %v", err)
	}

	if details == nil {
		t.Fatal("Expected book details, got nil")
	}

	// Verify essential fields have substantive data
	if details.Title == "" {
		t.Error("Expected non-empty title")
	}

	if details.Identifier == "" {
		t.Error("Expected non-empty identifier")
	}

	// Verify OURN follows expected format
	if !strings.HasPrefix(details.OURN, "urn:orm:book:") {
		t.Errorf("Expected OURN to start with 'urn:orm:book:', got %q", details.OURN)
	}

	// Verify numeric fields have substantive values (real book has pages)
	if details.PageCount == 0 {
		t.Error("Expected PageCount > 0 for a real book")
	}

	// Verify descriptions contain actual content
	if len(details.Descriptions) == 0 {
		t.Error("Expected at least one description entry")
	}

	t.Logf("Book: %s (ISBN: %s, pages: %d)", details.Title, details.ISBN, details.PageCount)

	// Log full response for debugging
	detailsJSON, _ := json.MarshalIndent(details, "", "  ")
	t.Logf("Book details:\n%s", string(detailsJSON))
}

// TestMCPResource_BookTOC tests the book-toc resource with real API.
func TestMCPResource_BookTOC(t *testing.T) {
	client := GetSharedClient()

	// Get table of contents for test book
	toc, err := client.GetBookTOC(TestBookID)
	if err != nil {
		t.Fatalf("GetBookTOC failed: %v", err)
	}

	if toc == nil {
		t.Fatal("Expected TOC, got nil")
	}

	// Verify expected fields
	if toc.BookID == "" {
		t.Error("Expected non-empty book_id")
	}

	if len(toc.TableOfContents) == 0 {
		t.Fatal("Expected at least one chapter in TOC")
	}

	// Verify TOC has reasonable volume (a real book has many entries)
	if toc.TotalChapters < 10 {
		t.Errorf("Expected TotalChapters >= 10 for a real book, got %d", toc.TotalChapters)
	}

	// Verify first TOC item has a valid href (structure check)
	first := toc.TableOfContents[0]
	if !strings.HasSuffix(first.Href, ".html") && !strings.HasSuffix(first.Href, ".xhtml") {
		t.Errorf("Expected first TOC item href to end with .html or .xhtml, got %q", first.Href)
	}

	t.Logf("TOC for book %s: %d chapters", toc.BookID, toc.TotalChapters)

	// Log first few chapters
	for i, ch := range toc.TableOfContents {
		if i >= 5 {
			t.Logf("... and %d more chapters", len(toc.TableOfContents)-5)
			break
		}
		t.Logf("  Chapter %d: %s (%s)", i+1, ch.Title, ch.Href)
	}
}

// TestMCPResource_BookChapter tests the book-chapter resource with real API.
func TestMCPResource_BookChapter(t *testing.T) {
	client := GetSharedClient()

	// Get chapter content for test book
	chapter, err := client.GetBookChapterContent(TestBookID, TestChapterName)
	if err != nil {
		t.Fatalf("GetBookChapterContent failed: %v", err)
	}

	if chapter == nil {
		t.Fatal("Expected chapter content, got nil")
	}

	// Verify expected fields
	if chapter.BookID == "" {
		t.Error("Expected non-empty book_id")
	}

	if chapter.ChapterName == "" {
		t.Error("Expected non-empty chapter_name")
	}

	// Verify chapter title is meaningful (not just the raw chapter name)
	if chapter.ChapterTitle == "" || chapter.ChapterTitle == chapter.ChapterName {
		t.Errorf("Expected meaningful chapter title, got %q", chapter.ChapterTitle)
	}

	t.Logf("Chapter: %s - %s", chapter.ChapterName, chapter.ChapterTitle)

	// Verify content has substantive volume (a real chapter has many paragraphs)
	if len(chapter.Content.Paragraphs) < 10 {
		t.Errorf("Expected at least 10 paragraphs in a real chapter, got %d", len(chapter.Content.Paragraphs))
	}

	if len(chapter.Content.Sections) < 3 {
		t.Errorf("Expected at least 3 sections in a real chapter, got %d", len(chapter.Content.Sections))
	}

	t.Logf("Chapter has %d sections, %d paragraphs, %d code blocks",
		len(chapter.Content.Sections),
		len(chapter.Content.Paragraphs),
		len(chapter.Content.CodeBlocks))
}
