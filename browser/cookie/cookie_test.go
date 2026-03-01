package cookie

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerImpl_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	validExpiry := time.Now().Add(24 * time.Hour)
	expiredExpiry := time.Now().Add(-1 * time.Hour)

	cookies := []*http.Cookie{
		{
			Name:    "orm-jwt",
			Value:   "test-token",
			Domain:  ".oreilly.com",
			Path:    "/",
			Expires: validExpiry,
			Secure:  true,
		},
		{
			Name:    "groot_sessionid",
			Value:   "session-123",
			Domain:  ".oreilly.com",
			Path:    "/",
			Expires: expiredExpiry,
		},
	}

	err := cm.SaveCookiesFromData(cookies)
	require.NoError(t, err)

	// Verify file was created
	assert.True(t, cm.CookieFileExists())

	// Load into a new manager
	cm2 := NewCookieManager(tmpDir)
	err = cm2.LoadCookies()
	require.NoError(t, err)

	// Only valid (non-expired) cookies should be loaded
	u, _ := url.Parse("https://learning.oreilly.com/")
	loaded := cm2.GetCookiesForURL(u)
	assert.Len(t, loaded, 1)
	assert.Equal(t, "orm-jwt", loaded[0].Name)
	assert.Equal(t, "test-token", loaded[0].Value)
}

func TestManagerImpl_GetCookiesForURL(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	validExpiry := time.Now().Add(24 * time.Hour)
	expiredExpiry := time.Now().Add(-1 * time.Hour)

	cm.cookies = []*http.Cookie{
		{Name: "exact", Value: "v1", Domain: "learning.oreilly.com", Path: "/", Expires: validExpiry},
		{Name: "dot-domain", Value: "v2", Domain: ".oreilly.com", Path: "/", Expires: validExpiry},
		{Name: "other-domain", Value: "v3", Domain: "example.com", Path: "/", Expires: validExpiry},
		{Name: "expired", Value: "v4", Domain: ".oreilly.com", Path: "/", Expires: expiredExpiry},
		{Name: "secure-only", Value: "v5", Domain: ".oreilly.com", Path: "/", Secure: true, Expires: validExpiry},
		{Name: "path-specific", Value: "v6", Domain: ".oreilly.com", Path: "/api/v1", Expires: validExpiry},
	}

	tests := []struct {
		name          string
		url           string
		expectedNames []string
	}{
		{
			name:          "exact domain match",
			url:           "https://learning.oreilly.com/",
			expectedNames: []string{"exact", "dot-domain", "secure-only"},
		},
		{
			name:          "subdomain match with leading dot",
			url:           "https://api.oreilly.com/",
			expectedNames: []string{"dot-domain", "secure-only"},
		},
		{
			name:          "secure cookie excluded on HTTP",
			url:           "http://learning.oreilly.com/",
			expectedNames: []string{"exact", "dot-domain"},
		},
		{
			name:          "path specific match",
			url:           "https://learning.oreilly.com/api/v1/search",
			expectedNames: []string{"exact", "dot-domain", "secure-only", "path-specific"},
		},
		{
			name:          "path mismatch",
			url:           "https://learning.oreilly.com/other",
			expectedNames: []string{"exact", "dot-domain", "secure-only"},
		},
		{
			name:          "no domain match",
			url:           "https://other.example.org/",
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.url)
			require.NoError(t, err)

			cookies := cm.GetCookiesForURL(u)
			names := make([]string, len(cookies))
			for i, c := range cookies {
				names[i] = c.Name
			}
			assert.ElementsMatch(t, tt.expectedNames, names)
		})
	}
}

func TestManagerImpl_SetCookies(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	u, _ := url.Parse("https://learning.oreilly.com/")

	// Initial set
	err := cm.SetCookies(u, []*http.Cookie{
		{Name: "cookie1", Value: "value1", Domain: ".oreilly.com", Path: "/"},
	})
	require.NoError(t, err)

	cookies := cm.GetCookiesForURL(u)
	assert.Len(t, cookies, 1)
	assert.Equal(t, "value1", cookies[0].Value)

	// Upsert same cookie with new value
	err = cm.SetCookies(u, []*http.Cookie{
		{Name: "cookie1", Value: "updated", Domain: ".oreilly.com", Path: "/"},
	})
	require.NoError(t, err)

	cookies = cm.GetCookiesForURL(u)
	assert.Len(t, cookies, 1)
	assert.Equal(t, "updated", cookies[0].Value)

	// Domain/path defaults
	err = cm.SetCookies(u, []*http.Cookie{
		{Name: "no-domain", Value: "val"},
	})
	require.NoError(t, err)

	found := false
	for _, c := range cm.cookies {
		if c.Name == "no-domain" {
			found = true
			assert.Equal(t, "learning.oreilly.com", c.Domain)
			assert.Equal(t, "/", c.Path)
		}
	}
	assert.True(t, found, "no-domain cookie should exist")
}

func TestManagerImpl_DeleteCookieFile(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	// Create a cookie file
	err := cm.SaveCookiesFromData([]*http.Cookie{
		{Name: "test", Value: "val", Domain: ".oreilly.com", Path: "/"},
	})
	require.NoError(t, err)
	assert.True(t, cm.CookieFileExists())

	// Delete it
	err = cm.DeleteCookieFile()
	require.NoError(t, err)
	assert.False(t, cm.CookieFileExists())

	// Delete non-existent file should not error
	err = cm.DeleteCookieFile()
	assert.NoError(t, err)
}

func TestManagerImpl_LoadCookies_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	err := cm.LoadCookies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cookie file does not exist")
}

func TestManagerImpl_LoadCookies_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	err := os.WriteFile(filepath.Join(tmpDir, cookieFileName), []byte("invalid json"), 0600)
	require.NoError(t, err)

	err = cm.LoadCookies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestManagerImpl_LoadCookies_AllExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	// Save cookies that are already expired
	expiredCookies := []*http.Cookie{
		{
			Name:    "expired1",
			Value:   "val1",
			Domain:  ".oreilly.com",
			Path:    "/",
			Expires: time.Now().Add(-1 * time.Hour),
		},
	}
	err := cm.SaveCookiesFromData(expiredCookies)
	require.NoError(t, err)

	// Load into new manager - should fail because all cookies are expired
	cm2 := NewCookieManager(tmpDir)
	err = cm2.LoadCookies()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid cookies found")
}

func TestIsImportantCookie(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewCookieManager(tmpDir)

	tests := []struct {
		name     string
		cookie   string
		expected bool
	}{
		{"O'Reilly JWT", "orm-jwt", true},
		{"session ID", "groot_sessionid", true},
		{"refresh token", "orm-rt", true},
		{"Google Analytics _ga", "_ga", false},
		{"Google Analytics _gid", "_gid", false},
		{"Google Analytics _gat", "_gat", false},
		{"Google Tag Manager", "_gtm", false},
		{"Facebook Pixel", "_fbp", false},
		{"Hotjar", "_hjid", false},
		{"Hotjar sample", "_hjIncludedInPageviewSample", false},
		{"Optimizely", "optimizelyEndUserId", false},
		{"old GA __utma", "__utma", false},
		{"old GA __utmz", "__utmz", false},
		{"arbitrary cookie", "my_custom_cookie", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, cm.isImportantCookie(tt.cookie))
		})
	}
}
