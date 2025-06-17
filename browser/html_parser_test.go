package browser

import (
	"testing"
)

// TestParseHTMLContent_BasicStructures tests HTML parsing with basic structures
func TestParseHTMLContent_BasicStructures(t *testing.T) {
	client := &BrowserClient{} // We only need the parsing methods

	testHTML := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Test Chapter</title>
	</head>
	<body>
		<h1 id="chapter-1">Chapter 1: Introduction</h1>
		<p>This is the first paragraph with some important information.</p>
		<p>This is the second paragraph explaining concepts.</p>
		
		<h2>Section 1.1</h2>
		<p>This paragraph is under a subsection.</p>
		
		<pre class="language-go"><code>package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}</code></pre>
		
		<img src="/images/diagram.png" alt="System diagram" />
		
		<p>For more information, see <a href="https://golang.org">Go documentation</a> 
		or <a href="#section-2">Section 2</a>.</p>
		
		<h3>Subsection 1.1.1</h3>
		<p>More detailed information here.</p>
	</body>
	</html>
	`

	result, err := client.parseHTMLContent(testHTML)
	if err != nil {
		t.Fatalf("parseHTMLContent failed: %v", err)
	}

	// Test title extraction
	if result.Title != "Test Chapter" {
		t.Errorf("Expected title 'Test Chapter', got '%s'", result.Title)
	}

	// Test paragraphs extraction
	expectedParagraphs := 4
	if len(result.Paragraphs) != expectedParagraphs {
		t.Errorf("Expected %d paragraphs, got %d", expectedParagraphs, len(result.Paragraphs))
	}

	// Test headings extraction
	expectedHeadings := 3
	if len(result.Headings) != expectedHeadings {
		t.Errorf("Expected %d headings, got %d", expectedHeadings, len(result.Headings))
	}

	// Validate heading levels and text
	expectedHeadingData := []struct {
		level int
		text  string
		id    string
	}{
		{1, "Chapter 1: Introduction", "chapter-1"},
		{2, "Section 1.1", ""},
		{3, "Subsection 1.1.1", ""},
	}

	for i, expected := range expectedHeadingData {
		if i >= len(result.Headings) {
			t.Errorf("Missing heading %d", i)
			continue
		}
		
		heading := result.Headings[i]
		if heading.Level != expected.level {
			t.Errorf("Heading %d: expected level %d, got %d", i, expected.level, heading.Level)
		}
		if heading.Text != expected.text {
			t.Errorf("Heading %d: expected text '%s', got '%s'", i, expected.text, heading.Text)
		}
		if heading.ID != expected.id {
			t.Errorf("Heading %d: expected ID '%s', got '%s'", i, expected.id, heading.ID)
		}
	}

	// Test code blocks extraction
	expectedCodeBlocks := 1
	if len(result.CodeBlocks) != expectedCodeBlocks {
		t.Errorf("Expected %d code blocks, got %d", expectedCodeBlocks, len(result.CodeBlocks))
	}

	if len(result.CodeBlocks) > 0 {
		codeBlock := result.CodeBlocks[0]
		if codeBlock.Language != "go" {
			t.Errorf("Expected language 'go', got '%s'", codeBlock.Language)
		}
		if !contains(codeBlock.Code, "package main") {
			t.Error("Expected code block to contain 'package main'")
		}
	}

	// Test images extraction
	expectedImages := 1
	if len(result.Images) != expectedImages {
		t.Errorf("Expected %d images, got %d", expectedImages, len(result.Images))
	}

	if len(result.Images) > 0 {
		image := result.Images[0]
		if image.Src != "/images/diagram.png" {
			t.Errorf("Expected image src '/images/diagram.png', got '%s'", image.Src)
		}
		if image.Alt != "System diagram" {
			t.Errorf("Expected image alt 'System diagram', got '%s'", image.Alt)
		}
	}

	// Test links extraction
	expectedLinks := 2
	if len(result.Links) != expectedLinks {
		t.Errorf("Expected %d links, got %d", expectedLinks, len(result.Links))
	}

	if len(result.Links) >= 2 {
		// Test external link
		externalLink := result.Links[0]
		if externalLink.Href != "https://golang.org" {
			t.Errorf("Expected external link href 'https://golang.org', got '%s'", externalLink.Href)
		}
		if externalLink.Type != "external" {
			t.Errorf("Expected external link type 'external', got '%s'", externalLink.Type)
		}

		// Test anchor link
		anchorLink := result.Links[1]
		if anchorLink.Href != "#section-2" {
			t.Errorf("Expected anchor link href '#section-2', got '%s'", anchorLink.Href)
		}
		if anchorLink.Type != "anchor" {
			t.Errorf("Expected anchor link type 'anchor', got '%s'", anchorLink.Type)
		}
	}
}

// TestParseHTMLContent_ComplexCodeBlocks tests parsing of various code block formats
func TestParseHTMLContent_ComplexCodeBlocks(t *testing.T) {
	client := &BrowserClient{}

	testHTML := `
	<html>
	<body>
		<pre class="highlight-python"><code>def hello():
    print("Hello from Python")</code></pre>
		
		<code class="language-javascript">console.log("Hello from JS");</code>
		
		<pre><code>// Plain code block
int main() {
    return 0;
}</code></pre>
		
		<div class="code-example">
			<pre class="language-yaml"><code>apiVersion: v1
kind: Pod
metadata:
  name: test-pod</code></pre>
			<p class="caption">Example 1-1. Basic Pod configuration</p>
		</div>
	</body>
	</html>
	`

	result, err := client.parseHTMLContent(testHTML)
	if err != nil {
		t.Fatalf("parseHTMLContent failed: %v", err)
	}

	// Should extract multiple code blocks
	if len(result.CodeBlocks) < 3 {
		t.Errorf("Expected at least 3 code blocks, got %d", len(result.CodeBlocks))
	}

	// Test language detection
	languages := make(map[string]bool)
	for _, block := range result.CodeBlocks {
		if block.Language != "" {
			languages[block.Language] = true
		}
	}

	expectedLanguages := []string{"python", "javascript", "yaml"}
	for _, lang := range expectedLanguages {
		if !languages[lang] {
			t.Errorf("Expected to detect language '%s'", lang)
		}
	}
}

// TestParseHTMLContent_EmptyInput tests parsing with empty or minimal input
func TestParseHTMLContent_EmptyInput(t *testing.T) {
	client := &BrowserClient{}

	testCases := []struct {
		name  string
		html  string
		valid bool
	}{
		{
			name:  "Empty string",
			html:  "",
			valid: true, // Should not error, just return empty content
		},
		{
			name:  "Minimal HTML",
			html:  "<html><body></body></html>",
			valid: true,
		},
		{
			name:  "Only text",
			html:  "Just some plain text",
			valid: true,
		},
		{
			name:  "Invalid HTML",
			html:  "<html><body><p>Unclosed paragraph</body></html>",
			valid: true, // HTML parser should handle this gracefully
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.parseHTMLContent(tc.html)
			
			if tc.valid {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result == nil {
					t.Error("Expected non-nil result")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

// TestCountWords tests the word counting utility function
func TestCountWords(t *testing.T) {
	testCases := []struct {
		name       string
		paragraphs []string
		expected   int
	}{
		{
			name:       "Empty paragraphs",
			paragraphs: []string{},
			expected:   0,
		},
		{
			name:       "Single paragraph",
			paragraphs: []string{"This is a test paragraph with seven words."},
			expected:   8,
		},
		{
			name:       "Multiple paragraphs",
			paragraphs: []string{
				"First paragraph has four words.",
				"Second paragraph also has four words.",
				"Third.",
			},
			expected:   9,
		},
		{
			name:       "Paragraphs with extra whitespace",
			paragraphs: []string{
				"  Paragraph   with   extra   spaces  ",
				"\t\nAnother\tparagraph\nwith\ttabs\n",
			},
			expected:   9,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := countWords(tc.paragraphs)
			if result != tc.expected {
				t.Errorf("Expected %d words, got %d", tc.expected, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}