package browser

import (
	"encoding/json"
	"strings"
	"testing"
)

func newTestClient() *BrowserClient {
	return &BrowserClient{}
}

func mustParse(t *testing.T, html string) *ParsedChapterContent {
	t.Helper()
	bc := newTestClient()
	result, err := bc.parseHTMLContent(html)
	if err != nil {
		t.Fatalf("parseHTMLContent failed: %v", err)
	}
	return result
}

func TestParseHTMLContent_SectionsAssociated(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1><p>Intro paragraph</p>
		<h2>Section Two</h2><p>Second paragraph</p><pre><code>code here</code></pre>
	</body></html>`)

	if len(r.Sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(r.Sections))
	}
	if r.Sections[0].Heading.Text != "Title" {
		t.Errorf("expected heading 'Title', got %q", r.Sections[0].Heading.Text)
	}
	if len(r.Sections[0].Content) < 1 {
		t.Fatal("expected h1 section to have content")
	}
	if r.Sections[1].Heading.Text != "Section Two" {
		t.Errorf("expected heading 'Section Two', got %q", r.Sections[1].Heading.Text)
	}
	if len(r.Sections[1].Content) < 2 {
		t.Fatalf("expected h2 section to have at least 2 content items, got %d", len(r.Sections[1].Content))
	}
}

func TestParseHTMLContent_PreambleContent(t *testing.T) {
	r := mustParse(t, `<html><body>
		<p>Before any heading</p>
		<h2>First Heading</h2><p>After heading</p>
	</body></html>`)

	if len(r.Sections) < 2 {
		t.Fatalf("expected at least 2 sections (preamble + heading), got %d", len(r.Sections))
	}
	if r.Sections[0].Heading.Text != "" {
		t.Errorf("expected preamble heading to be empty, got %q", r.Sections[0].Heading.Text)
	}
	if len(r.Sections[0].Content) < 1 {
		t.Fatal("expected preamble to have content")
	}
}

func TestParseHTMLContent_NoHeadings(t *testing.T) {
	r := mustParse(t, `<html><body><p>Just a paragraph</p><p>Another one</p></body></html>`)

	if len(r.Sections) != 1 {
		t.Fatalf("expected 1 preamble section, got %d", len(r.Sections))
	}
	if r.Sections[0].Heading.Text != "" {
		t.Errorf("expected empty heading for preamble, got %q", r.Sections[0].Heading.Text)
	}
	if len(r.Sections[0].Content) != 2 {
		t.Errorf("expected 2 content items, got %d", len(r.Sections[0].Content))
	}
}

func TestParseHTMLContent_EmptyHTML(t *testing.T) {
	r := mustParse(t, `<html><body></body></html>`)

	if len(r.Sections) != 0 {
		t.Errorf("expected 0 sections for empty HTML, got %d", len(r.Sections))
	}
}

func TestParseHTMLContent_CodeBlockLanguage(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h2>Code</h2>
		<pre><code class="language-go">fmt.Println("hello")</code></pre>
	</body></html>`)

	if len(r.Sections) < 1 {
		t.Fatal("expected at least 1 section")
	}
	found := false
	for _, item := range r.Sections[0].Content {
		if cb, ok := item.(CodeBlockElement); ok && cb.Language == "go" {
			found = true
		}
	}
	if !found {
		t.Error("expected CodeBlockElement with language 'go'")
	}
}

func TestParseHTMLContent_DocumentOrder(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1>
		<p>First para</p>
		<pre><code>some code</code></pre>
		<p>Second para</p>
	</body></html>`)

	if len(r.Sections) < 1 {
		t.Fatal("expected at least 1 section")
	}
	s := r.Sections[0]
	if len(s.Content) != 3 {
		t.Fatalf("expected 3 content items, got %d", len(s.Content))
	}
	if _, ok := s.Content[0].(ParagraphElement); !ok {
		t.Errorf("expected content[0] to be ParagraphElement, got %T", s.Content[0])
	}
	if _, ok := s.Content[1].(CodeBlockElement); !ok {
		t.Errorf("expected content[1] to be CodeBlockElement, got %T", s.Content[1])
	}
	if _, ok := s.Content[2].(ParagraphElement); !ok {
		t.Errorf("expected content[2] to be ParagraphElement, got %T", s.Content[2])
	}
}

func TestParseHTMLContent_InlineCodeNotDuplicated(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1>
		<p>Use <code>x</code> variable</p>
		<pre><code>real code block</code></pre>
	</body></html>`)

	if len(r.Sections) < 1 {
		t.Fatal("expected at least 1 section")
	}
	codeCount := 0
	for _, item := range r.Sections[0].Content {
		if _, ok := item.(CodeBlockElement); ok {
			codeCount++
		}
	}
	if codeCount != 1 {
		t.Errorf("expected exactly 1 CodeBlockElement, got %d", codeCount)
	}
}

func TestParseHTMLContent_ListInSection(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1>
		<ul><li>item one</li><li>item two</li></ul>
		<ol><li>first</li><li>second</li></ol>
	</body></html>`)

	if len(r.Sections) < 1 {
		t.Fatal("expected at least 1 section")
	}
	listCount := 0
	for _, item := range r.Sections[0].Content {
		if le, ok := item.(ListElement); ok {
			listCount++
			if listCount == 1 && le.Ordered {
				t.Error("expected first list to be unordered")
			}
			if listCount == 2 && !le.Ordered {
				t.Error("expected second list to be ordered")
			}
		}
	}
	if listCount != 2 {
		t.Errorf("expected 2 ListElements, got %d", listCount)
	}
}

func TestParseHTMLContent_LinkInSection(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1>
		<a href="https://example.com">Example</a>
	</body></html>`)

	if len(r.Sections) < 1 {
		t.Fatal("expected at least 1 section")
	}
	found := false
	for _, item := range r.Sections[0].Content {
		if le, ok := item.(LinkElement); ok {
			if le.Href == "https://example.com" && le.Text == "Example" && le.LinkType == "external" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected LinkElement with href 'https://example.com'")
	}
}

func TestParseHTMLContent_JSONSerialization(t *testing.T) {
	r := mustParse(t, `<html><body>
		<h1>Title</h1>
		<p>Hello</p>
		<pre><code>code</code></pre>
		<img src="img.png" alt="pic"/>
	</body></html>`)

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	jsonStr := string(data)
	for _, typ := range []string{`"type":"paragraph"`, `"type":"code_block"`, `"type":"image"`} {
		if !strings.Contains(jsonStr, typ) {
			t.Errorf("expected JSON to contain %s", typ)
		}
	}
}

func TestParseHTMLContent_NoFlatArrays(t *testing.T) {
	r := mustParse(t, `<html><body><h1>Title</h1><p>text</p></body></html>`)

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	jsonStr := string(data)
	for _, field := range []string{`"paragraphs"`, `"headings"`, `"code_blocks"`, `"images"`, `"links"`} {
		if strings.Contains(jsonStr, field) {
			t.Errorf("expected JSON to NOT contain flat array field %s", field)
		}
	}
}
