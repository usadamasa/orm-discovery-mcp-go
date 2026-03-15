package cache

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SaveParams groups the parameters for SaveResponseAsMarkdown.
type SaveParams struct {
	Dir          string
	Query        string
	Results      []map[string]any
	HistoryID    string
	TotalResults int
}

// markdownHeader groups the header fields for markdown generation.
type markdownHeader struct {
	query        string
	date         string
	totalResults int
	resultCount  int
	historyID    string
}

// resultView holds extracted fields from a search result map.
type resultView struct {
	title       string
	id          string
	authors     string
	contentType string
	publisher   string
	pubDate     string
	url         string
	description string
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a query string into a filesystem-safe slug.
func slugify(query string) string {
	s := strings.ToLower(query)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}

	if s == "" {
		h := fnv.New32a()
		h.Write([]byte(query))
		s = fmt.Sprintf("query-%08x", h.Sum32())
	}

	return s
}

// SaveResponseAsMarkdown saves search results as a Markdown file.
// Returns the file path on success.
func SaveResponseAsMarkdown(p SaveParams) (string, error) {
	if err := os.MkdirAll(p.Dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	now := time.Now()
	filename := now.Format("20060102-150405") + "_" + slugify(p.Query) + ".md"
	filePath := filepath.Join(p.Dir, filename)
	tmpPath := filePath + ".tmp"

	hdr := markdownHeader{
		query:        p.Query,
		date:         now.Format(time.RFC3339),
		totalResults: EffectiveTotalResults(p.TotalResults, len(p.Results)),
		resultCount:  len(p.Results),
		historyID:    p.HistoryID,
	}

	var b strings.Builder

	// Header
	b.WriteString("# O'Reilly Search Results\n\n")
	fmt.Fprintf(&b, "- Query: %s\n", hdr.query)
	fmt.Fprintf(&b, "- Date: %s\n", hdr.date)
	fmt.Fprintf(&b, "- Total Results: %d\n", hdr.totalResults)
	fmt.Fprintf(&b, "- Results in this file: %d\n", hdr.resultCount)
	fmt.Fprintf(&b, "- History ID: %s\n", hdr.historyID)
	b.WriteString("\n---\n")

	// Results
	for i, result := range p.Results {
		writeResultMarkdown(&b, i+1, result)
	}

	if err := os.WriteFile(tmpPath, []byte(b.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			slog.Warn("failed to remove temp file", "path", tmpPath, "error", removeErr)
		}
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	return filePath, nil
}

// htmlTagPattern matches HTML tags (opening, closing, self-closing).
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags from a string and normalizes whitespace.
func stripHTML(s string) string {
	if !strings.ContainsAny(s, "<>") {
		return s // fast path: no HTML tags
	}
	stripped := htmlTagPattern.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(stripped), " ")
}

// writeResultMarkdown writes a single search result as Markdown to the builder.
func writeResultMarkdown(b *strings.Builder, index int, result map[string]any) {
	v := resultView{}
	v.title, _ = result["title"].(string)
	v.id = ExtractStringField(result, "product_id", "isbn", "id")
	v.authors = extractAuthorStringSimple(result["authors"])
	if ct, ok := result["content_type"].(string); ok {
		v.contentType = ct
	}
	if p, ok := result["publisher"].(string); ok {
		v.publisher = p
	}
	if pd, ok := result["published_date"].(string); ok {
		v.pubDate = pd
	}
	if u, ok := result["url"].(string); ok {
		v.url = u
	}
	if d, ok := result["description"].(string); ok {
		v.description = stripHTML(d)
	}

	fmt.Fprintf(b, "\n## Result %d: %s\n\n", index, v.title)
	if v.id != "" {
		fmt.Fprintf(b, "- ID: %s\n", v.id)
	}
	if v.authors != "" {
		fmt.Fprintf(b, "- Authors: %s\n", v.authors)
	}
	if v.contentType != "" {
		fmt.Fprintf(b, "- Content Type: %s\n", v.contentType)
	}
	if v.publisher != "" {
		fmt.Fprintf(b, "- Publisher: %s\n", v.publisher)
	}
	if v.pubDate != "" {
		fmt.Fprintf(b, "- Published: %s\n", v.pubDate)
	}
	if v.url != "" {
		fmt.Fprintf(b, "- URL: %s\n", v.url)
	}
	if v.description != "" {
		fmt.Fprintf(b, "- Description: %s\n", v.description)
	}
}

// EffectiveTotalResults returns totalResults if positive, or falls back to
// resultCount when the API returns 0 (e.g. nil *int pointer).
func EffectiveTotalResults(totalResults, resultCount int) int {
	if totalResults == 0 && resultCount > 0 {
		return resultCount
	}
	return totalResults
}

// ExtractStringField returns the first non-empty string value from the map for the given keys.
func ExtractStringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// extractAuthorStringSimple extracts a comma-separated author string
// from []string or []any types (no browser-specific type handling).
func extractAuthorStringSimple(v any) string {
	switch authors := v.(type) {
	case []string:
		if len(authors) > 0 {
			return strings.Join(authors, ", ")
		}
	case []any:
		strs := make([]string, 0, len(authors))
		for _, a := range authors {
			if s, ok := a.(string); ok {
				strs = append(strs, s)
			}
		}
		if len(strs) > 0 {
			return strings.Join(strs, ", ")
		}
	}
	return ""
}
