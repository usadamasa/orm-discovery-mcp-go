package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/config"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/history"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/mcputil"
)

// mockBrowserClient implements browser.Client for testing.
type mockBrowserClient struct {
	searchResults      []map[string]any
	searchTotalResults int
	searchErr          error
}

func (m *mockBrowserClient) SearchContent(_ string, _ map[string]any) ([]map[string]any, int, error) {
	return m.searchResults, m.searchTotalResults, m.searchErr
}
func (m *mockBrowserClient) AskQuestion(_ string, _ time.Duration) (*browser.AnswerResponse, error) {
	return nil, nil
}
func (m *mockBrowserClient) GetBookDetails(_ string) (*browser.BookDetailResponse, error) {
	return nil, nil
}
func (m *mockBrowserClient) GetBookTOC(_ string) (*browser.TableOfContentsResponse, error) {
	return nil, nil
}
func (m *mockBrowserClient) GetBookChapterContent(_, _ string) (*browser.ChapterContentResponse, error) {
	return nil, nil
}
func (m *mockBrowserClient) GetQuestionByID(_ string) (*browser.AnswerResponse, error) {
	return nil, nil
}
func (m *mockBrowserClient) Reauthenticate() error    { return nil }
func (m *mockBrowserClient) CheckAndResetAuth() error { return nil }
func (m *mockBrowserClient) Close()                   {}

// newTestServer creates a Server with mock browser client and temp directories.
func newTestServer(t *testing.T, mock *mockBrowserClient) *Server {
	t.Helper()
	tmpDir := t.TempDir()

	cfg := &config.Config{
		XDGDirs: &config.XDGDirs{
			CacheHome: filepath.Join(tmpDir, "cache"),
			StateHome: filepath.Join(tmpDir, "state"),
		},
		History: config.HistoryOpts{MaxEntries: 100},
	}

	historyManager := history.NewManager(
		cfg.XDGDirs.ResearchHistoryPath(),
		cfg.History.MaxEntries,
	)
	_ = historyManager.Load()

	return &Server{
		browserClient:  mock,
		config:         cfg,
		historyManager: historyManager,
		startedAt:      time.Now(),
		serverVersion:  "test",
	}
}

func TestSearchContentHandler_SingleSave(t *testing.T) {
	mock := &mockBrowserClient{
		searchResults: []map[string]any{
			{"title": "Book A", "product_id": "111", "content_type": "book"},
			{"title": "Book B", "product_id": "222", "content_type": "book"},
		},
		searchTotalResults: 0, // API returns 0 (nil pointer case)
	}

	srv := newTestServer(t, mock)

	req := &mcp.CallToolRequest{}
	args := SearchContentArgs{Query: "test query"}

	_, _, err := srv.SearchContentHandler(context.Background(), req, args)
	if err != nil {
		t.Fatalf("SearchContentHandler returned error: %v", err)
	}

	// Check cache directory has exactly 1 file
	cacheDir := srv.config.XDGDirs.ResponseCachePath()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("failed to read cache dir: %v", err)
	}

	mdFiles := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles++
		}
	}
	if mdFiles != 1 {
		t.Errorf("expected exactly 1 cache file, got %d", mdFiles)
	}
}

func TestSearchContentHandler_HistoryIDInFile(t *testing.T) {
	mock := &mockBrowserClient{
		searchResults: []map[string]any{
			{"title": "Book A", "product_id": "111", "content_type": "book"},
		},
		searchTotalResults: 1,
	}

	srv := newTestServer(t, mock)

	req := &mcp.CallToolRequest{}
	args := SearchContentArgs{Query: "history id test"}

	_, structured, err := srv.SearchContentHandler(context.Background(), req, args)
	if err != nil {
		t.Fatalf("SearchContentHandler returned error: %v", err)
	}
	if structured == nil {
		t.Fatal("expected structured result")
	}

	// Read the cache file and verify history ID is present
	if structured.FilePath == "" {
		t.Fatal("expected FilePath in structured result")
	}

	data, err := os.ReadFile(structured.FilePath)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, structured.HistoryID) {
		t.Errorf("cache file does not contain history ID %q", structured.HistoryID)
	}
}

func TestExtractProductIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "book-details URI",
			uri:      "oreilly://book-details/12345",
			expected: "12345",
		},
		{
			name:     "book-toc URI",
			uri:      "oreilly://book-toc/12345",
			expected: "12345",
		},
		{
			name:     "URL-encoded product ID",
			uri:      "oreilly://book-details/978%2D1%2D491",
			expected: "978-1-491",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "trailing slash only",
			uri:      "oreilly://book-details/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mcputil.ExtractProductIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("mcputil.ExtractProductIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestExtractProductIDAndChapterFromURI(t *testing.T) {
	tests := []struct {
		name            string
		uri             string
		expectedProduct string
		expectedChapter string
	}{
		{
			name:            "standard chapter URI",
			uri:             "oreilly://book-chapter/12345/ch01.html",
			expectedProduct: "12345",
			expectedChapter: "ch01.html",
		},
		{
			name:            "URL-encoded slash in chapter name",
			uri:             "oreilly://book-chapter/12345/ch%2F01.html",
			expectedProduct: "12345",
			expectedChapter: "ch/01.html",
		},
		{
			name:            "URL-encoded space in chapter name",
			uri:             "oreilly://book-chapter/12345/ch01%20intro.html",
			expectedProduct: "12345",
			expectedChapter: "ch01 intro.html",
		},
		{
			name:            "empty URI",
			uri:             "",
			expectedProduct: "",
			expectedChapter: "",
		},
		{
			name:            "missing chapter",
			uri:             "oreilly://book-chapter/12345",
			expectedProduct: "",
			expectedChapter: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, chapter := mcputil.ExtractProductIDAndChapterFromURI(tt.uri)
			if product != tt.expectedProduct {
				t.Errorf("product: got %q, want %q", product, tt.expectedProduct)
			}
			if chapter != tt.expectedChapter {
				t.Errorf("chapter: got %q, want %q", chapter, tt.expectedChapter)
			}
		})
	}
}

func TestExtractQuestionIDFromURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "standard answer URI",
			uri:      "oreilly://answer/q-abc-123",
			expected: "q-abc-123",
		},
		{
			name:     "URL-encoded hyphen",
			uri:      "oreilly://answer/q%2Dabc",
			expected: "q-abc",
		},
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mcputil.ExtractQuestionIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("mcputil.ExtractQuestionIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestBuildLightweightResponse_BrowserAuthorConversion(t *testing.T) {
	srv := &Server{}

	results := []map[string]any{
		{
			"id":           "123",
			"title":        "Go Programming",
			"content_type": "book",
			"authors": []browser.Author{
				{Name: "John Doe"},
				{Name: "Jane Smith"},
			},
		},
	}

	_, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/test.md", 0, 1)

	if structured == nil || len(structured.Results) == 0 {
		t.Fatal("expected structured results")
	}

	authors, ok := structured.Results[0]["authors"]
	if !ok {
		t.Fatal("expected authors key in result")
	}

	authorNames, ok := authors.([]string)
	if !ok {
		t.Fatalf("expected []string authors, got %T", authors)
	}
	if len(authorNames) != 2 || authorNames[0] != "John Doe" || authorNames[1] != "Jane Smith" {
		t.Errorf("expected [John Doe, Jane Smith], got %v", authorNames)
	}
}

func TestBuildLightweightResponse_FilePath(t *testing.T) {
	srv := &Server{}

	results := []map[string]any{
		{"id": "123", "title": "Test Book", "content_type": "book"},
	}

	toolResult, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/cache/test.md", 0, 1)

	if structured == nil {
		t.Fatal("expected structured result")
	}

	if structured.FilePath != "/tmp/cache/test.md" {
		t.Errorf("expected FilePath '/tmp/cache/test.md', got %q", structured.FilePath)
	}

	if structured.HistoryID != "hist_123" {
		t.Errorf("expected HistoryID 'hist_123', got %q", structured.HistoryID)
	}

	// Tool result should contain text with file path
	if toolResult == nil {
		t.Fatal("expected tool result with text content")
	}
}

func TestBuildLightweightResponse_LimitsTo5Results(t *testing.T) {
	srv := &Server{}

	results := make([]map[string]any, 10)
	for i := range results {
		results[i] = map[string]any{
			"id":    "id-" + string(rune('0'+i)),
			"title": "Book " + string(rune('0'+i)),
		}
	}

	toolResult, structured := srv.buildLightweightResponse(results, "hist_123", "/tmp/test.md", 0, 50)

	if structured == nil {
		t.Fatal("expected structured result")
	}

	// Only top 5 results should be in structured output for context efficiency
	if len(structured.Results) != 5 {
		t.Errorf("expected 5 results in structured (top 5 only), got %d", len(structured.Results))
	}

	// But Count should reflect total results returned by API
	if structured.Count != 10 {
		t.Errorf("expected Count 10, got %d", structured.Count)
	}

	// But text should mention "and X more results"
	if toolResult == nil || len(toolResult.Content) == 0 {
		t.Fatal("expected tool result content")
	}
}
