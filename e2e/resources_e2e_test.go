//go:build e2e

package e2e

import (
	"encoding/json"
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

	// Verify expected fields
	if details.Title == "" {
		t.Error("Expected non-empty title")
	}

	if len(details.Authors) == 0 {
		t.Error("Expected at least one author")
	}

	t.Logf("Book: %s by %v", details.Title, details.Authors)

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
		t.Error("Expected at least one chapter in TOC")
	}

	t.Logf("TOC for %s: %d chapters", toc.BookTitle, toc.TotalChapters)

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

	t.Logf("Chapter: %s - %s", chapter.ChapterName, chapter.ChapterTitle)

	// Verify content was parsed
	if len(chapter.Content.Sections) == 0 && len(chapter.Content.Paragraphs) == 0 {
		t.Error("Expected some content in chapter")
	}

	t.Logf("Chapter has %d sections, %d paragraphs, %d code blocks",
		len(chapter.Content.Sections),
		len(chapter.Content.Paragraphs),
		len(chapter.Content.CodeBlocks))
}
