package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorCategoryInternal,
		},
		{
			name:     "401 status code",
			err:      fmt.Errorf("API request failed with status 401"),
			expected: ErrorCategoryAuth,
		},
		{
			name:     "403 status code",
			err:      fmt.Errorf("API request failed with status 403"),
			expected: ErrorCategoryAuth,
		},
		{
			name:     "authentication error message",
			err:      fmt.Errorf("authentication error: token expired"),
			expected: ErrorCategoryAuth,
		},
		{
			name:     "unauthorized message",
			err:      fmt.Errorf("unauthorized access"),
			expected: ErrorCategoryAuth,
		},
		{
			name:     "timeout error",
			err:      fmt.Errorf("connection timeout after 30s"),
			expected: ErrorCategoryNetwork,
		},
		{
			name:     "connection refused",
			err:      fmt.Errorf("connection refused"),
			expected: ErrorCategoryNetwork,
		},
		{
			name:     "404 not found",
			err:      fmt.Errorf("API request failed with status 404"),
			expected: ErrorCategoryNotFound,
		},
		{
			name:     "not found message",
			err:      fmt.Errorf("resource not found"),
			expected: ErrorCategoryNotFound,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("something unexpected happened"),
			expected: ErrorCategoryInternal,
		},
		{
			name:     "wrapped auth error",
			err:      fmt.Errorf("search failed: %w", fmt.Errorf("401 unauthorized")),
			expected: ErrorCategoryAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		expectedMessage string
	}{
		{
			name:            "auth error returns user-facing message",
			err:             fmt.Errorf("API request failed with status 401: invalid token xyz123"),
			expectedMessage: userFacingErrorMessage(ErrorCategoryAuth),
		},
		{
			name:            "network error returns user-facing message",
			err:             fmt.Errorf("connection timeout: dial tcp 10.0.0.1:443"),
			expectedMessage: userFacingErrorMessage(ErrorCategoryNetwork),
		},
		{
			name:            "not found error returns user-facing message",
			err:             fmt.Errorf("API request failed with status 404: /api/v2/books/12345"),
			expectedMessage: userFacingErrorMessage(ErrorCategoryNotFound),
		},
		{
			name:            "internal error returns user-facing message",
			err:             fmt.Errorf("unexpected nil pointer at server.go:123"),
			expectedMessage: userFacingErrorMessage(ErrorCategoryInternal),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.err)
			assert.Equal(t, tt.expectedMessage, result)
		})
	}
}

func TestUserFacingErrorMessage(t *testing.T) {
	// Verify all categories have user-facing messages
	categories := []ErrorCategory{
		ErrorCategoryAuth,
		ErrorCategoryNetwork,
		ErrorCategoryNotFound,
		ErrorCategoryValidation,
		ErrorCategoryInternal,
	}

	for _, cat := range categories {
		msg := userFacingErrorMessage(cat)
		assert.NotEmpty(t, msg, "category %s should have a user-facing message", cat)
	}
}

func TestErrorResourceContents(t *testing.T) {
	uri := "oreilly://book-details/12345"
	err := errors.New("internal database error: connection pool exhausted")

	result := errorResourceContents(uri, err)

	assert.NotNil(t, result)
	assert.Len(t, result.Contents, 1)
	assert.Equal(t, uri, result.Contents[0].URI)
	assert.Equal(t, "application/json", result.Contents[0].MIMEType)
	// Should NOT contain internal details
	assert.NotContains(t, result.Contents[0].Text, "connection pool exhausted")
	// Should contain user-facing error
	assert.Contains(t, result.Contents[0].Text, "error")
}
