package mcputil

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorCategory represents the classification of an error.
type ErrorCategory string

const (
	ErrorCategoryAuth       ErrorCategory = "auth"
	ErrorCategoryNetwork    ErrorCategory = "network"
	ErrorCategoryNotFound   ErrorCategory = "not_found"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryInternal   ErrorCategory = "internal"
)

// CategorizeError classifies an error into a category based on its message.
func CategorizeError(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryInternal
	}

	msg := strings.ToLower(err.Error())

	// Auth errors
	if strings.Contains(msg, "401") ||
		strings.Contains(msg, "403") ||
		strings.Contains(msg, "authentication error") ||
		strings.Contains(msg, "unauthorized") {
		return ErrorCategoryAuth
	}

	// Network errors
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network") {
		return ErrorCategoryNetwork
	}

	// Not found errors
	if strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") {
		return ErrorCategoryNotFound
	}

	return ErrorCategoryInternal
}

// UserFacingErrorMessage returns a safe, actionable message for the given error category.
func UserFacingErrorMessage(category ErrorCategory) string {
	switch category {
	case ErrorCategoryAuth:
		return "Authentication failed. Please use oreilly_reauthenticate to refresh your session."
	case ErrorCategoryNetwork:
		return "Network error occurred. Please check your connection and try again."
	case ErrorCategoryNotFound:
		return "The requested resource was not found. Please verify the ID and try again."
	case ErrorCategoryValidation:
		return "Invalid input. Please check your parameters and try again."
	case ErrorCategoryInternal:
		return "An internal error occurred. Please try again later."
	default:
		return "An unexpected error occurred. Please try again later."
	}
}

// SanitizeError logs the internal error details and returns a user-facing message.
func SanitizeError(err error, logAttrs ...any) string {
	category := CategorizeError(err)
	attrs := make([]any, 0, 4+len(logAttrs))
	attrs = append(attrs, "error", err, "category", string(category))
	attrs = append(attrs, logAttrs...)
	slog.Error("Operation failed", attrs...)
	return UserFacingErrorMessage(category)
}

// ErrorResourceContents creates a ReadResourceResult with a sanitized error message.
func ErrorResourceContents(uri string, err error, logAttrs ...any) *mcp.ReadResourceResult {
	msg := SanitizeError(err, logAttrs...)
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     fmt.Sprintf(`{"error": "%s"}`, msg),
		}},
	}
}
