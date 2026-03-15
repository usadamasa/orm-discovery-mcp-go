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

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a query string into a filesystem-safe slug.
func Slugify(query string) string {
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
func SaveResponseAsMarkdown(cacheDir, query string, results []map[string]any, historyID string, totalResults int) (string, error) {
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	filename := time.Now().Format("20060102-150405") + "_" + Slugify(query) + ".md"
	filePath := filepath.Join(cacheDir, filename)
	tmpPath := filePath + ".tmp"

	var b strings.Builder

	// Header
	b.WriteString("# O'Reilly Search Results\n\n")
	fmt.Fprintf(&b, "- Query: %s\n", query)
	fmt.Fprintf(&b, "- Date: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Total Results: %d\n", EffectiveTotalResults(totalResults, len(results)))
	fmt.Fprintf(&b, "- Results in this file: %d\n", len(results))
	fmt.Fprintf(&b, "- History ID: %s\n", historyID)
	b.WriteString("\n---\n")

	// Results
	for i, result := range results {
		WriteResultMarkdown(&b, i+1, result)
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

// StripHTML removes HTML tags from a string and normalizes whitespace.
func StripHTML(s string) string {
	if !strings.ContainsAny(s, "<>") {
		return s // fast path: no HTML tags
	}
	stripped := htmlTagPattern.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(stripped), " ")
}

// WriteResultMarkdown writes a single search result as Markdown to the builder.
func WriteResultMarkdown(b *strings.Builder, index int, result map[string]any) {
	title, _ := result["title"].(string)
	fmt.Fprintf(b, "\n## Result %d: %s\n\n", index, title)

	id := ExtractStringField(result, "product_id", "isbn", "id")
	if id != "" {
		fmt.Fprintf(b, "- ID: %s\n", id)
	}

	if authorStr := extractAuthorStringSimple(result["authors"]); authorStr != "" {
		fmt.Fprintf(b, "- Authors: %s\n", authorStr)
	}

	if ct, ok := result["content_type"].(string); ok && ct != "" {
		fmt.Fprintf(b, "- Content Type: %s\n", ct)
	}

	if publisher, ok := result["publisher"].(string); ok && publisher != "" {
		fmt.Fprintf(b, "- Publisher: %s\n", publisher)
	}

	if pubDate, ok := result["published_date"].(string); ok && pubDate != "" {
		fmt.Fprintf(b, "- Published: %s\n", pubDate)
	}

	if u, ok := result["url"].(string); ok && u != "" {
		fmt.Fprintf(b, "- URL: %s\n", u)
	}

	if desc, ok := result["description"].(string); ok && desc != "" {
		fmt.Fprintf(b, "- Description: %s\n", StripHTML(desc))
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
