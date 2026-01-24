package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetXDGDirs_Defaults(t *testing.T) {
	// 環境変数をクリア
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", "")

	dirs, err := GetXDGDirs("")
	if err != nil {
		t.Fatalf("GetXDGDirs() returned error: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	expectedStateHome := filepath.Join(homeDir, ".local", "state", AppName)
	expectedCacheHome := filepath.Join(homeDir, ".cache", AppName)
	expectedConfigHome := filepath.Join(homeDir, ".config", AppName)

	if dirs.StateHome != expectedStateHome {
		t.Errorf("StateHome = %q, want %q", dirs.StateHome, expectedStateHome)
	}
	if dirs.CacheHome != expectedCacheHome {
		t.Errorf("CacheHome = %q, want %q", dirs.CacheHome, expectedCacheHome)
	}
	if dirs.ConfigHome != expectedConfigHome {
		t.Errorf("ConfigHome = %q, want %q", dirs.ConfigHome, expectedConfigHome)
	}
}

func TestGetXDGDirs_WithXDGEnvVars(t *testing.T) {
	// 一時ディレクトリを作成してXDG環境変数をセット
	tmpDir := t.TempDir()

	customState := filepath.Join(tmpDir, "custom-state")
	customCache := filepath.Join(tmpDir, "custom-cache")
	customConfig := filepath.Join(tmpDir, "custom-config")

	t.Setenv("XDG_STATE_HOME", customState)
	t.Setenv("XDG_CACHE_HOME", customCache)
	t.Setenv("XDG_CONFIG_HOME", customConfig)
	t.Setenv("ORM_MCP_GO_DEBUG_DIR", "")

	dirs, err := GetXDGDirs("")
	if err != nil {
		t.Fatalf("GetXDGDirs() returned error: %v", err)
	}

	expectedStateHome := filepath.Join(customState, AppName)
	expectedCacheHome := filepath.Join(customCache, AppName)
	expectedConfigHome := filepath.Join(customConfig, AppName)

	if dirs.StateHome != expectedStateHome {
		t.Errorf("StateHome = %q, want %q", dirs.StateHome, expectedStateHome)
	}
	if dirs.CacheHome != expectedCacheHome {
		t.Errorf("CacheHome = %q, want %q", dirs.CacheHome, expectedCacheHome)
	}
	if dirs.ConfigHome != expectedConfigHome {
		t.Errorf("ConfigHome = %q, want %q", dirs.ConfigHome, expectedConfigHome)
	}
}

func TestGetXDGDirs_WithDebugDir(t *testing.T) {
	// XDG環境変数をクリアして、デバッグ用ORM_MCP_GO_DEBUG_DIRを使用
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	tmpDir := t.TempDir()
	debugDir := filepath.Join(tmpDir, "debug-dir")

	dirs, err := GetXDGDirs(debugDir)
	if err != nil {
		t.Fatalf("GetXDGDirs() returned error: %v", err)
	}

	// デバッグモードでは全てのパスがdebugDirを使用
	if dirs.StateHome != debugDir {
		t.Errorf("StateHome = %q, want %q", dirs.StateHome, debugDir)
	}
	if dirs.CacheHome != debugDir {
		t.Errorf("CacheHome = %q, want %q", dirs.CacheHome, debugDir)
	}
	if dirs.ConfigHome != debugDir {
		t.Errorf("ConfigHome = %q, want %q", dirs.ConfigHome, debugDir)
	}
}

func TestGetXDGDirs_DebugDirPriorityOverXDG(t *testing.T) {
	// XDG環境変数とデバッグディレクトリの両方が設定されている場合、デバッグディレクトリが優先
	tmpDir := t.TempDir()

	customState := filepath.Join(tmpDir, "custom-state")
	customCache := filepath.Join(tmpDir, "custom-cache")
	debugDir := filepath.Join(tmpDir, "debug-dir")

	t.Setenv("XDG_STATE_HOME", customState)
	t.Setenv("XDG_CACHE_HOME", customCache)
	t.Setenv("XDG_CONFIG_HOME", "")

	dirs, err := GetXDGDirs(debugDir)
	if err != nil {
		t.Fatalf("GetXDGDirs() returned error: %v", err)
	}

	// デバッグディレクトリが設定されている場合はそちらが優先（XDGより優先）
	if dirs.StateHome != debugDir {
		t.Errorf("StateHome = %q, want %q (debugDir should take priority over XDG)", dirs.StateHome, debugDir)
	}
	if dirs.CacheHome != debugDir {
		t.Errorf("CacheHome = %q, want %q (debugDir should take priority over XDG)", dirs.CacheHome, debugDir)
	}
	if dirs.ConfigHome != debugDir {
		t.Errorf("ConfigHome = %q, want %q (debugDir should take priority over XDG)", dirs.ConfigHome, debugDir)
	}
}

func TestXDGDirs_EnsureExists(t *testing.T) {
	tmpDir := t.TempDir()

	dirs := &XDGDirs{
		StateHome:  filepath.Join(tmpDir, "state", AppName),
		CacheHome:  filepath.Join(tmpDir, "cache", AppName),
		ConfigHome: filepath.Join(tmpDir, "config", AppName),
	}

	// ディレクトリがまだ存在しないことを確認
	for _, dir := range []string{dirs.StateHome, dirs.CacheHome, dirs.ConfigHome} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("Directory should not exist yet: %s", dir)
		}
	}

	// EnsureExists を呼び出し
	if err := dirs.EnsureExists(); err != nil {
		t.Fatalf("EnsureExists() returned error: %v", err)
	}

	// ディレクトリが作成されたことを確認
	for _, dir := range []string{dirs.StateHome, dirs.CacheHome, dirs.ConfigHome} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory was not created: %s, error: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Path is not a directory: %s", dir)
		}
		// パーミッションが0700であることを確認
		perm := info.Mode().Perm()
		if perm != 0700 {
			t.Errorf("Directory permission = %o, want 0700", perm)
		}
	}
}

func TestXDGDirs_EnsureExists_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	dirs := &XDGDirs{
		StateHome:  filepath.Join(tmpDir, "state", AppName),
		CacheHome:  filepath.Join(tmpDir, "cache", AppName),
		ConfigHome: filepath.Join(tmpDir, "config", AppName),
	}

	// 最初にディレクトリを作成
	for _, dir := range []string{dirs.StateHome, dirs.CacheHome, dirs.ConfigHome} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	// 2回目の呼び出しでもエラーにならないことを確認
	if err := dirs.EnsureExists(); err != nil {
		t.Errorf("EnsureExists() should not return error for existing dirs: %v", err)
	}
}

func TestXDGDirs_CookiePath(t *testing.T) {
	dirs := &XDGDirs{
		StateHome:  "/test/state/orm-mcp-go",
		CacheHome:  "/test/cache/orm-mcp-go",
		ConfigHome: "/test/config/orm-mcp-go",
	}

	// CookieはCacheHomeに保存（ファイル名はcookie/cookie.goと一致させる）
	expected := "/test/cache/orm-mcp-go/orm-mcp-go-cookies.json"
	if got := dirs.CookiePath(); got != expected {
		t.Errorf("CookiePath() = %q, want %q", got, expected)
	}
}

func TestXDGDirs_ChromeDataDir(t *testing.T) {
	dirs := &XDGDirs{
		StateHome:  "/test/state/orm-mcp-go",
		CacheHome:  "/test/cache/orm-mcp-go",
		ConfigHome: "/test/config/orm-mcp-go",
	}

	// StateHomeを使用することを確認(セッション状態のため)
	got := dirs.ChromeDataDir()
	if got != "/test/state/orm-mcp-go" {
		t.Errorf("ChromeDataDir() = %q, want %q", got, "/test/state/orm-mcp-go")
	}
}

func TestXDGDirs_ScreenshotDir(t *testing.T) {
	dirs := &XDGDirs{
		StateHome:  "/test/state/orm-mcp-go",
		CacheHome:  "/test/cache/orm-mcp-go",
		ConfigHome: "/test/config/orm-mcp-go",
	}

	// StateHomeを使用することを確認(セッション状態のため)
	expected := "/test/state/orm-mcp-go/screenshots"
	if got := dirs.ScreenshotDir(); got != expected {
		t.Errorf("ScreenshotDir() = %q, want %q", got, expected)
	}
}

func TestXDGDirs_LogPath(t *testing.T) {
	dirs := &XDGDirs{
		StateHome:  "/test/state/orm-mcp-go",
		CacheHome:  "/test/cache/orm-mcp-go",
		ConfigHome: "/test/config/orm-mcp-go",
	}

	expected := "/test/state/orm-mcp-go/orm-mcp-go.log"
	if got := dirs.LogPath(); got != expected {
		t.Errorf("LogPath() = %q, want %q", got, expected)
	}
}
