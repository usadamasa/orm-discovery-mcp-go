package main

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

// categorizeError classifies an error into a category based on its message.
func categorizeError(err error) ErrorCategory {
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

// userFacingErrorMessage returns a safe, actionable message for the given error category.
func userFacingErrorMessage(category ErrorCategory) string {
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

// sanitizeError logs the internal error details and returns a user-facing message.
func sanitizeError(err error, logAttrs ...any) string {
	category := categorizeError(err)
	attrs := append([]any{"error", err, "category", string(category)}, logAttrs...)
	slog.Error("Operation failed", attrs...)
	return userFacingErrorMessage(category)
}

// errorResourceContents creates a ReadResourceResult with a sanitized error message.
func errorResourceContents(uri string, err error, logAttrs ...any) *mcp.ReadResourceResult {
	msg := sanitizeError(err, logAttrs...)
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     fmt.Sprintf(`{"error": "%s"}`, msg),
		}},
	}
}
