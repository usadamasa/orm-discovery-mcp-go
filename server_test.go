package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
			result := extractProductIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractProductIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
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
			product, chapter := extractProductIDAndChapterFromURI(tt.uri)
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
			result := extractQuestionIDFromURI(tt.uri)
			if result != tt.expected {
				t.Errorf("extractQuestionIDFromURI(%q) = %q, want %q", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestOriginValidationMiddleware_NoOrigin_Allowed(t *testing.T) {
	handler := originValidationMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestOriginValidationMiddleware_LocalhostOrigin_Allowed(t *testing.T) {
	handler := originValidationMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		origin string
	}{
		{"localhost with port", "http://localhost:3000"},
		{"localhost without port", "http://localhost"},
		{"127.0.0.1 with port", "http://127.0.0.1:8080"},
		{"127.0.0.1 without port", "http://127.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("origin=%q: status = %d, want %d", tt.origin, rec.Code, http.StatusOK)
			}
		})
	}
}

func TestOriginValidationMiddleware_EvilOrigin_Rejected(t *testing.T) {
	handler := originValidationMiddleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestOriginValidationMiddleware_AllowedOrigins(t *testing.T) {
	allowedOrigins := []string{"https://my-app.example.com", "https://dev.example.com"}
	handler := originValidationMiddleware(allowedOrigins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{"allowed origin 1", "https://my-app.example.com", http.StatusOK},
		{"allowed origin 2", "https://dev.example.com", http.StatusOK},
		{"not allowed origin", "https://other.example.com", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("origin=%q: status = %d, want %d", tt.origin, rec.Code, tt.wantStatus)
			}
		})
	}
}
