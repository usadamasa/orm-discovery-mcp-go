package mcputil

import (
	"net/url"
	"strings"
)

// ExtractProductIDFromURI extracts product_id from URIs like
// "oreilly://book-details/{product_id}" or "oreilly://book-toc/{product_id}".
func ExtractProductIDFromURI(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	id := strings.TrimPrefix(u.Path, "/")
	return id
}

// ExtractProductIDAndChapterFromURI extracts product_id and chapter_name from URIs like
// "oreilly://book-chapter/{product_id}/{chapter_name}".
func ExtractProductIDAndChapterFromURI(uri string) (string, string) {
	if uri == "" {
		return "", ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", ""
	}
	// Use RawPath to preserve %2F in chapter names; fall back to Path if RawPath is empty
	rawPath := u.RawPath
	if rawPath == "" {
		rawPath = u.Path
	}
	rawPath = strings.TrimPrefix(rawPath, "/")
	parts := strings.SplitN(rawPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	productID, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", ""
	}
	chapterName, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", ""
	}
	return productID, chapterName
}

// ExtractQuestionIDFromURI extracts question_id from URIs like
// "oreilly://answer/{question_id}".
func ExtractQuestionIDFromURI(uri string) string {
	return ExtractProductIDFromURI(uri)
}
