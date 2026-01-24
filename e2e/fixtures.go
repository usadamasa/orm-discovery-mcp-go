//go:build e2e

// Package e2e provides end-to-end testing for the MCP server.
package e2e

import (
	"os"
)

// Test fixtures: Well-known book IDs for testing
const (
	// TestBookID is a well-known O'Reilly book for testing (Learning Go)
	TestBookID = "9781098166298"

	// TestChapterName is a well-known chapter name for testing
	TestChapterName = "ch01"

	// TestSearchQuery is a search query for testing
	TestSearchQuery = "Go programming"
)

// TestConfig holds configuration for E2E tests loaded from environment variables.
type TestConfig struct {
	OReillyUserID   string
	OReillyPassword string
	Debug           bool
	TmpDir          string
}

// LoadTestConfig loads test configuration from environment variables.
// Returns nil if required environment variables are not set.
func LoadTestConfig() *TestConfig {
	userID := os.Getenv("OREILLY_USER_ID")
	password := os.Getenv("OREILLY_PASSWORD")

	if userID == "" || password == "" {
		return nil
	}

	tmpDir := os.Getenv("ORM_MCP_GO_TMP_DIR")
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	return &TestConfig{
		OReillyUserID:   userID,
		OReillyPassword: password,
		Debug:           os.Getenv("ORM_MCP_GO_DEBUG") == "true",
		TmpDir:          tmpDir,
	}
}
