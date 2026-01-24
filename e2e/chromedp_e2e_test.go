//go:build e2e

package e2e

import (
	"os"
	"testing"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// TestChromeDP_BrowserLifecycle tests ChromeDP browser startup and shutdown.
func TestChromeDP_BrowserLifecycle(t *testing.T) {
	client := GetSharedClient()

	// Verify client is initialized
	if client == nil {
		t.Fatal("Shared browser client is nil")
	}

	t.Log("ChromeDP browser lifecycle test passed (verified shared client)")
}

// TestChromeDPLogin tests the complete login flow through ChromeDP.
// Note: This test requires a fresh client to test the login flow.
func TestChromeDPLogin(t *testing.T) {
	cfg := GetSharedConfig()

	// Use a fresh cookie manager without cached cookies
	// to force a fresh login
	freshLoginDir := cfg.TmpDir + "/fresh-login-test"
	if err := os.MkdirAll(freshLoginDir, 0755); err != nil {
		t.Fatalf("Failed to create fresh login directory: %v", err)
	}
	cookieManager := cookie.NewCookieManager(freshLoginDir)

	client, err := browser.NewBrowserClient(
		cfg.OReillyUserID,
		cfg.OReillyPassword,
		cookieManager,
		cfg.Debug,
		cfg.TmpDir,
	)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	defer client.Close()

	// Verify login succeeded by performing a search
	results, err := client.SearchContent("test", nil)
	if err != nil {
		t.Fatalf("Search after login failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results after login")
	}

	t.Logf("Login verification: found %d results", len(results))
}

// TestCookieRestoration tests that cookies are properly saved and restored.
// Note: This test requires fresh clients to test the cookie save/restore flow.
func TestCookieRestoration(t *testing.T) {
	cfg := GetSharedConfig()

	// Use a shared cookie directory for this test
	cookieDir := cfg.TmpDir + "/cookie-restoration-test"
	if err := os.MkdirAll(cookieDir, 0755); err != nil {
		t.Fatalf("Failed to create cookie directory: %v", err)
	}
	cookieManager := cookie.NewCookieManager(cookieDir)

	// First: Create client and login (saves cookies)
	client1, err := browser.NewBrowserClient(
		cfg.OReillyUserID,
		cfg.OReillyPassword,
		cookieManager,
		cfg.Debug,
		cfg.TmpDir,
	)
	if err != nil {
		t.Fatalf("First login failed: %v", err)
	}
	client1.Close()
	t.Log("First login completed, cookies saved")

	// Verify cookies were saved
	if !cookieManager.CookieFileExists() {
		t.Fatal("Expected cookie file to exist after first login")
	}

	// Second: Create a new client with the same cookie manager
	// This should restore cookies instead of logging in fresh
	cookieManager2 := cookie.NewCookieManager(cookieDir)
	client2, err := browser.NewBrowserClient(
		cfg.OReillyUserID,
		cfg.OReillyPassword,
		cookieManager2,
		cfg.Debug,
		cfg.TmpDir,
	)
	if err != nil {
		t.Fatalf("Second client creation with restored cookies failed: %v", err)
	}
	defer client2.Close()

	// Verify authentication works with restored cookies
	results, err := client2.SearchContent("Go", nil)
	if err != nil {
		t.Fatalf("Search with restored cookies failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results with restored cookies")
	}

	t.Logf("Cookie restoration verified: found %d results", len(results))
}

// TestChromeDP_ReauthenticationFlow tests the reauthentication mechanism.
func TestChromeDP_ReauthenticationFlow(t *testing.T) {
	cfg := GetSharedConfig()
	client := GetSharedClient()

	// Trigger reauthentication (simulates cookie expiration handling)
	err := client.ReauthenticateIfNeeded(cfg.OReillyUserID, cfg.OReillyPassword)
	if err != nil {
		t.Fatalf("Reauthentication failed: %v", err)
	}

	// Verify reauthentication succeeded
	results, err := client.SearchContent("Python", nil)
	if err != nil {
		t.Fatalf("Search after reauthentication failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results after reauthentication")
	}

	t.Logf("Reauthentication verified: found %d results", len(results))
}
