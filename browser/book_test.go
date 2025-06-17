package browser

import (
	"os"
	"testing"
)

// TestGetBookChapterContent_HappyPath tests the successful retrieval and parsing of chapter content
func TestGetBookChapterContent_HappyPath(t *testing.T) {
	// Skip test if credentials are not available
	userID := os.Getenv("OREILLY_USER_ID")
	password := os.Getenv("OREILLY_PASSWORD")
	if userID == "" || password == "" {
		t.Skip("OREILLY_USER_ID and OREILLY_PASSWORD environment variables required for integration test")
	}

	// Initialize browser client
	client, err := NewBrowserClient(userID, password)
	if err != nil {
		t.Fatalf("Failed to create browser client: %v", err)
	}
	defer client.Close()

	// Test with a known O'Reilly book
	// Using "Learning Go" book as it's commonly available
	testCases := []struct {
		name        string
		productID   string
		chapterName string
		expectTitle bool // Whether we expect a meaningful title
	}{
		{
			name:        "Learning Go - Preface",
			productID:   "9781492077206", // Learning Go book ID
			chapterName: "pr01",          // Preface chapter
			expectTitle: true,
		},
		{
			name:        "Docker Up and Running - Preface", 
			productID:   "9781098131814", // Docker Up and Running
			chapterName: "pr01",          // Preface chapter
			expectTitle: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call GetBookChapterContent
			result, err := client.GetBookChapterContent(tc.productID, tc.chapterName)
			if err != nil {
				t.Fatalf("GetBookChapterContent failed: %v", err)
			}

			// Validate basic response structure
			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			// Validate required fields
			if result.BookID != tc.productID {
				t.Errorf("Expected BookID %s, got %s", tc.productID, result.BookID)
			}

			if result.ChapterName != tc.chapterName {
				t.Errorf("Expected ChapterName %s, got %s", tc.chapterName, result.ChapterName)
			}

			// Validate content structure
			if result.Content.Title == "" && tc.expectTitle {
				t.Error("Expected non-empty content title")
			}

			// Validate that we extracted some paragraphs
			if len(result.Content.Paragraphs) == 0 {
				t.Error("Expected at least some paragraphs to be extracted")
			}

			// Validate metadata
			if result.Metadata == nil {
				t.Error("Expected metadata to be present")
			}

			if extractionMethod, ok := result.Metadata["extraction_method"]; !ok || extractionMethod != "html_parsing" {
				t.Error("Expected extraction_method to be 'html_parsing'")
			}

			if wordCount, ok := result.Metadata["word_count"]; !ok || wordCount == 0 {
				t.Error("Expected word_count to be greater than 0")
			}

			// Validate source URL
			if result.SourceURL == "" {
				t.Error("Expected non-empty source URL")
			}

			// Log some stats for debugging
			t.Logf("Chapter: %s", result.ChapterTitle)
			t.Logf("Paragraphs: %d", len(result.Content.Paragraphs))
			t.Logf("Headings: %d", len(result.Content.Headings))
			t.Logf("Code blocks: %d", len(result.Content.CodeBlocks))
			t.Logf("Images: %d", len(result.Content.Images))
			t.Logf("Links: %d", len(result.Content.Links))
			t.Logf("Word count: %v", result.Metadata["word_count"])

			// Validate that we have some structured content
			totalContent := len(result.Content.Paragraphs) + 
						   len(result.Content.Headings) + 
						   len(result.Content.CodeBlocks)
			
			if totalContent == 0 {
				t.Error("Expected to extract some structured content (paragraphs, headings, or code blocks)")
			}

			// Validate heading structure if present
			for i, heading := range result.Content.Headings {
				if heading.Level < 1 || heading.Level > 6 {
					t.Errorf("Heading %d has invalid level: %d", i, heading.Level)
				}
				if heading.Text == "" {
					t.Errorf("Heading %d has empty text", i)
				}
			}

			// Validate code blocks if present
			for i, codeBlock := range result.Content.CodeBlocks {
				if codeBlock.Code == "" {
					t.Errorf("Code block %d has empty code", i)
				}
			}

			// Validate links if present
			for i, link := range result.Content.Links {
				if link.Href == "" {
					t.Errorf("Link %d has empty href", i)
				}
				if link.Text == "" {
					t.Errorf("Link %d has empty text", i)
				}
				if link.Type != "external" && link.Type != "internal" && link.Type != "anchor" {
					t.Errorf("Link %d has invalid type: %s", i, link.Type)
				}
			}

			// Validate images if present
			for i, image := range result.Content.Images {
				if image.Src == "" {
					t.Errorf("Image %d has empty src", i)
				}
			}
		})
	}
}

// TestGetBookChapterContent_InvalidInputs tests error handling for invalid inputs
func TestGetBookChapterContent_InvalidInputs(t *testing.T) {
	// Skip test if credentials are not available
	userID := os.Getenv("OREILLY_USER_ID")
	password := os.Getenv("OREILLY_PASSWORD")
	if userID == "" || password == "" {
		t.Skip("OREILLY_USER_ID and OREILLY_PASSWORD environment variables required for integration test")
	}

	// Initialize browser client
	client, err := NewBrowserClient(userID, password)
	if err != nil {
		t.Fatalf("Failed to create browser client: %v", err)
	}
	defer client.Close()

	testCases := []struct {
		name        string
		productID   string
		chapterName string
		expectError bool
	}{
		{
			name:        "Empty product ID",
			productID:   "",
			chapterName: "ch01",
			expectError: true,
		},
		{
			name:        "Empty chapter name",
			productID:   "9781492077206",
			chapterName: "",
			expectError: true,
		},
		{
			name:        "Invalid product ID",
			productID:   "invalid_id",
			chapterName: "ch01",
			expectError: true,
		},
		{
			name:        "Invalid chapter name",
			productID:   "9781492077206",
			chapterName: "nonexistent_chapter",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.GetBookChapterContent(tc.productID, tc.chapterName)
			
			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if result != nil {
					t.Error("Expected nil result when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == nil {
					t.Error("Expected non-nil result")
				}
			}
		})
	}
}

// BenchmarkGetBookChapterContent benchmarks the chapter content retrieval
func BenchmarkGetBookChapterContent(b *testing.B) {
	// Skip benchmark if credentials are not available
	userID := os.Getenv("OREILLY_USER_ID")
	password := os.Getenv("OREILLY_PASSWORD")
	if userID == "" || password == "" {
		b.Skip("OREILLY_USER_ID and OREILLY_PASSWORD environment variables required for benchmark")
	}

	// Initialize browser client
	client, err := NewBrowserClient(userID, password)
	if err != nil {
		b.Fatalf("Failed to create browser client: %v", err)
	}
	defer client.Close()

	// Use a small chapter for benchmarking
	productID := "9781492077206"
	chapterName := "pr01"

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := client.GetBookChapterContent(productID, chapterName)
		if err != nil {
			b.Fatalf("GetBookChapterContent failed: %v", err)
		}
	}
}