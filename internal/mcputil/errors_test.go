package mcputil

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_IsAuth(t *testing.T) {
	h := ErrorHandler{}
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"401 status code", fmt.Errorf("API request failed with status 401"), true},
		{"403 status code", fmt.Errorf("API request failed with status 403"), true},
		{"authentication error message", fmt.Errorf("authentication error: token expired"), true},
		{"unauthorized message", fmt.Errorf("unauthorized access"), true},
		{"timeout error", fmt.Errorf("connection timeout after 30s"), false},
		{"generic error", fmt.Errorf("something unexpected happened"), false},
		{"wrapped auth error", fmt.Errorf("search failed: %w", fmt.Errorf("401 unauthorized")), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, h.IsAuth(tt.err))
		})
	}
}

func TestErrorHandler_Sanitize(t *testing.T) {
	h := ErrorHandler{}
	tests := []struct {
		name        string
		err         error
		shouldMatch string
	}{
		{"auth error", fmt.Errorf("API request failed with status 401: invalid token xyz123"), h.ValidationMessage()},
		{"network error", fmt.Errorf("connection timeout: dial tcp 10.0.0.1:443"), "Network error"},
		{"not found error", fmt.Errorf("API request failed with status 404: /api/v2/books/12345"), "not found"},
		{"internal error", fmt.Errorf("unexpected nil pointer at server.go:123"), "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Sanitize(tt.err)
			assert.NotEmpty(t, result)
			// Sanitized message should not contain raw error details
			assert.NotContains(t, result, "xyz123")
			assert.NotContains(t, result, "10.0.0.1")
			assert.NotContains(t, result, "server.go:123")
		})
	}
}

func TestErrorHandler_ValidationMessage(t *testing.T) {
	h := ErrorHandler{}
	msg := h.ValidationMessage()
	assert.NotEmpty(t, msg)
	assert.Contains(t, msg, "Invalid input")
}

func TestErrorHandler_ResourceContents(t *testing.T) {
	h := ErrorHandler{}
	uri := "oreilly://book-details/12345"
	err := errors.New("internal database error: connection pool exhausted")

	result := h.ResourceContents(uri, err)

	assert.NotNil(t, result)
	assert.Len(t, result.Contents, 1)
	assert.Equal(t, uri, result.Contents[0].URI)
	assert.Equal(t, "application/json", result.Contents[0].MIMEType)
	assert.NotContains(t, result.Contents[0].Text, "connection pool exhausted")
	assert.Contains(t, result.Contents[0].Text, "error")
}

func TestErrorHandler_Categorize_Coverage(t *testing.T) {
	h := ErrorHandler{}
	// Verify different error categories produce different sanitized messages
	authMsg := h.Sanitize(fmt.Errorf("401 unauthorized"))
	netMsg := h.Sanitize(fmt.Errorf("connection timeout"))
	notFoundMsg := h.Sanitize(fmt.Errorf("404 not found"))
	internalMsg := h.Sanitize(fmt.Errorf("something broke"))

	// All messages should be distinct
	msgs := []string{authMsg, netMsg, notFoundMsg, internalMsg}
	seen := make(map[string]bool)
	for _, m := range msgs {
		assert.False(t, seen[m], "duplicate message: %s", m)
		seen[m] = true
	}
}
