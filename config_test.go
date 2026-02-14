package main

import (
	"testing"
)

func TestLoadConfig_BindAddress_Default(t *testing.T) {
	// 必須環境変数を設定
	t.Setenv("OREILLY_USER_ID", "test@example.com")
	t.Setenv("OREILLY_PASSWORD", "testpass")
	// ORM_MCP_GO_DEBUG_DIR を設定してXDGディレクトリを一時ディレクトリに向ける
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", t.TempDir())
	// BIND_ADDRESS を未設定にする (t.Setenv は Cleanup で元に戻すが、明示的にクリアしたい)
	t.Setenv("BIND_ADDRESS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %q, want %q", cfg.BindAddress, "127.0.0.1")
	}
}

func TestLoadConfig_BindAddress_EnvOverride(t *testing.T) {
	t.Setenv("OREILLY_USER_ID", "test@example.com")
	t.Setenv("OREILLY_PASSWORD", "testpass")
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", t.TempDir())
	t.Setenv("BIND_ADDRESS", "0.0.0.0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want %q", cfg.BindAddress, "0.0.0.0")
	}
}
