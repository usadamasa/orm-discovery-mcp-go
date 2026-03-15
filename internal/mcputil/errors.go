package mcputil

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errorCategory represents the classification of an error.
type errorCategory string

const (
	errorCategoryAuth       errorCategory = "auth"
	errorCategoryNetwork    errorCategory = "network"
	errorCategoryNotFound   errorCategory = "not_found"
	errorCategoryValidation errorCategory = "validation"
	errorCategoryInternal   errorCategory = "internal"
)

// ErrorHandler provides error categorization and sanitization for MCP responses.
type ErrorHandler struct{}

// categorize classifies an error into a category based on its message.
func (ErrorHandler) categorize(err error) errorCategory {
	if err == nil {
		return errorCategoryInternal
	}

	msg := strings.ToLower(err.Error())

	if strings.Contains(msg, "401") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "authentication error") ||
		strings.Contains(msg, "unauthorized") {
		return errorCategoryAuth
	}

	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network") {
		return errorCategoryNetwork
	}

	if strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") {
		return errorCategoryNotFound
	}

	return errorCategoryInternal
}

// userFacingMessage returns a safe, actionable message for the given error category.
func (ErrorHandler) userFacingMessage(category errorCategory) string {
	switch category {
	case errorCategoryAuth:
		return "Authentication failed. Please use oreilly_reauthenticate to refresh your session."
	case errorCategoryNetwork:
		return "Network error occurred. Please check your connection and try again."
	case errorCategoryNotFound:
		return "The requested resource was not found. Please verify the ID and try again."
	case errorCategoryValidation:
		return "Invalid input. Please check your parameters and try again."
	case errorCategoryInternal:
		return "An internal error occurred. Please try again later."
	default:
		return "An unexpected error occurred. Please try again later."
	}
}

// errorDetail captures the result of error analysis.
type errorDetail struct {
	err      error
	category errorCategory
	message  string
}

// analyze categorizes the error and resolves the user-facing message.
func (h ErrorHandler) analyze(err error) errorDetail {
	cat := h.categorize(err)
	return errorDetail{err: err, category: cat, message: h.userFacingMessage(cat)}
}

// logError logs internal error details with category context.
func (ErrorHandler) logError(d errorDetail, logAttrs ...any) {
	attrs := make([]any, 0, 4+len(logAttrs))
	attrs = append(attrs, "error", d.err, "category", string(d.category))
	attrs = append(attrs, logAttrs...)
	slog.Error("Operation failed", attrs...)
}

// IsAuth returns true if the error is an authentication error.
func (h ErrorHandler) IsAuth(err error) bool {
	return h.categorize(err) == errorCategoryAuth
}

// ValidationMessage returns the user-facing message for validation errors.
func (h ErrorHandler) ValidationMessage() string {
	return h.userFacingMessage(errorCategoryValidation)
}

// Sanitize logs the internal error details and returns a user-facing message.
func (h ErrorHandler) Sanitize(err error, logAttrs ...any) string {
	d := h.analyze(err)
	h.logError(d, logAttrs...)
	return d.message
}

// ResourceContents creates a ReadResourceResult with a sanitized error message.
func (h ErrorHandler) ResourceContents(uri string, err error, logAttrs ...any) *mcp.ReadResourceResult {
	msg := h.Sanitize(err, logAttrs...)
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     fmt.Sprintf(`{"error": "%s"}`, msg),
		}},
	}
}
