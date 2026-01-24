//go:build e2e

package e2e

import (
	"testing"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// TestMCPServerInitialization tests that the browser client can be created
// with valid credentials.
func TestMCPServerInitialization(t *testing.T) {
	client := GetSharedClient()

	// Verify client is initialized
	if client == nil {
		t.Fatal("Shared browser client is nil")
	}

	t.Log("Browser client verified as initialized")
}

// TestServerWithRealAuthentication tests the full authentication flow
// including cookie restoration or fresh login.
func TestServerWithRealAuthentication(t *testing.T) {
	client := GetSharedClient()

	// Verify authentication by performing a simple search
	results, err := client.SearchContent("Go", nil)
	if err != nil {
		t.Fatalf("Search failed after authentication: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results, got empty")
	}

	t.Logf("Authentication verified: found %d results", len(results))
}

// TestServerBrowserClientLifecycle tests that the browser client
// can be properly closed without errors.
// Note: This test uses its own client to test the Close() behavior.
func TestServerBrowserClientLifecycle(t *testing.T) {
	cfg := GetSharedConfig()

	cookieManager := cookie.NewCookieManager(cfg.TmpDir)

	client, err := browser.NewBrowserClient(
		cfg.OReillyUserID,
		cfg.OReillyPassword,
		cookieManager,
		cfg.Debug,
		cfg.TmpDir,
	)
	if err != nil {
		t.Fatalf("Failed to create browser client: %v", err)
	}

	// Close should not panic
	client.Close()

	// Double close should also not panic
	client.Close()

	t.Log("Browser client lifecycle test passed")
}
