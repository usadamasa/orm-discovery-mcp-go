//go:build e2e

package e2e

import (
	"log"
	"os"
	"testing"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

var sharedClient *browser.BrowserClient
var sharedConfig *TestConfig

func TestMain(m *testing.M) {
	cfg := LoadTestConfig()
	sharedConfig = cfg

	// Cookie不在時はE2Eテストを失敗させる (APIへの疎通確認が目的のため)
	cookieManager := cookie.NewCookieManager(cfg.TmpDir)
	if !cookieManager.CookieFileExists() {
		log.Fatalf("Cookie not found at %s. E2E tests require authentication. Run 'bin/orm-discovery-mcp-go --login' first, then set ORM_MCP_GO_TMP_DIR to the cookie directory.", cfg.TmpDir)
	}

	// Create shared client (only once for all tests)
	client, err := browser.NewBrowserClient(
		cookieManager,
		cfg.Debug,
		cfg.TmpDir,
	)
	if err != nil {
		log.Fatalf("Failed to create shared browser client: %v", err)
	}
	sharedClient = client

	// Run tests
	code := m.Run()

	// Cleanup
	sharedClient.Close()

	os.Exit(code)
}

// GetSharedClient returns the shared browser client for tests.
func GetSharedClient() *browser.BrowserClient {
	return sharedClient
}

// GetSharedConfig returns the shared test configuration.
func GetSharedConfig() *TestConfig {
	return sharedConfig
}
