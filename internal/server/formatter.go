package server

import (
	"fmt"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
)

// formatSearchResultsMarkdown formats search results as human-readable Markdown.
func formatSearchResultsMarkdown(result *SearchContentResult) string {
	if len(result.Results) == 0 {
		return "## Search Results\n\nNo results found."
	}

	var b strings.Builder

	fmt.Fprintf(&b, "## Search Results (%d of %d)\n\n", result.Count, result.TotalResults)

	for i, r := range result.Results {
		title, _ := r["title"].(string)
		id, _ := r["id"].(string)

		fmt.Fprintf(&b, "%d. **%s**", i+1, title)

		// Authors
		if authorStr := extractAuthorString(r["authors"]); authorStr != "" {
			fmt.Fprintf(&b, " by %s", authorStr)
		}

		if id != "" {
			fmt.Fprintf(&b, "\n   - ID: `%s`", id)
		}
		b.WriteString("\n")
	}

	if result.HasMore {
		fmt.Fprintf(&b, "\n> More results available: use `offset=%d` to get next page.\n", result.NextOffset)
	}

	if result.HistoryID != "" {
		fmt.Fprintf(&b, "\n*History ID: %s*\n", result.HistoryID)
	}

	return b.String()
}

// extractAuthorSlice extracts author names as a []string from various author types.
func extractAuthorSlice(v any) []string {
	switch authors := v.(type) {
	case []string:
		return authors
	case []browser.Author:
		names := make([]string, 0, len(authors))
		for _, a := range authors {
			names = append(names, a.Name)
		}
		return names
	case []any:
		strs := make([]string, 0, len(authors))
		for _, a := range authors {
			if s, ok := a.(string); ok {
				strs = append(strs, s)
			}
		}
		return strs
	}
	return nil
}

// extractAuthorString extracts a comma-separated author string from various author types.
func extractAuthorString(v any) string {
	if names := extractAuthorSlice(v); len(names) > 0 {
		return strings.Join(names, ", ")
	}
	return ""
}

// formatAskQuestionMarkdown formats an answer as human-readable Markdown.
func formatAskQuestionMarkdown(result *AskQuestionResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## Q: %s\n\n", result.Question)
	b.WriteString(result.Answer)
	b.WriteString("\n")

	if len(result.Sources) > 0 {
		b.WriteString("\n### Sources\n\n")
		for _, src := range result.Sources {
			if src.URL != "" {
				fmt.Fprintf(&b, "- [%s](%s)\n", src.Title, src.URL)
			} else {
				fmt.Fprintf(&b, "- %s\n", src.Title)
			}
		}
	}

	if len(result.FollowupQuestions) > 0 {
		b.WriteString("\n### Follow-up Questions\n\n")
		for _, q := range result.FollowupQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
	}

	return b.String()
}
