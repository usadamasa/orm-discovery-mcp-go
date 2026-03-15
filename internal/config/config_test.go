package config

import (
	"testing"
)

func TestLoadConfig_BindAddress_Default(t *testing.T) {
	// ORM_MCP_GO_DEBUG_DIR を設定してXDGディレクトリを一時ディレクトリに向ける
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", t.TempDir())
	// BIND_ADDRESS を未設定にする (t.Setenv は Cleanup で元に戻すが、明示的にクリアしたい)
	t.Setenv("BIND_ADDRESS", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Server.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %q, want %q", cfg.Server.BindAddress, "127.0.0.1")
	}
}

func TestLoadConfig_BindAddress_EnvOverride(t *testing.T) {
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", t.TempDir())
	t.Setenv("BIND_ADDRESS", "0.0.0.0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Server.BindAddress != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want %q", cfg.Server.BindAddress, "0.0.0.0")
	}
}
