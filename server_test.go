package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
