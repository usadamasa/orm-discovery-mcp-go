package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppName はアプリケーション名（XDGサブディレクトリ名として使用）
const AppName = "orm-mcp-go"

// XDGDirs はXDG Base Directory Specification準拠のディレクトリパスを保持する
type XDGDirs struct {
	StateHome  string // ログ、Chrome一時データ、スクリーンショット用 ($XDG_STATE_HOME/orm-mcp-go)
	CacheHome  string // Cookie保存用（再生成可能なデータのため） ($XDG_CACHE_HOME/orm-mcp-go)
	ConfigHome string // 将来の設定ファイル用 ($XDG_CONFIG_HOME/orm-mcp-go)
}

// GetXDGDirs はXDGディレクトリパスを解決する
//
// 優先度:
//  1. debugDir (ORM_MCP_GO_DEBUG_DIR) → 設定時はStateHome/CacheHome/ConfigHome全てに使用（最優先）
//  2. XDG環境変数 (XDG_STATE_HOME, XDG_CACHE_HOME, XDG_CONFIG_HOME)
//  3. デフォルトパス (~/.local/state/orm-mcp-go など)
func GetXDGDirs(debugDir string) (*XDGDirs, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dirs := &XDGDirs{}

	// StateHome の解決
	dirs.StateHome = resolveXDGPath(
		"XDG_STATE_HOME",
		debugDir,
		filepath.Join(homeDir, ".local", "state"),
	)

	// CacheHome の解決
	dirs.CacheHome = resolveXDGPath(
		"XDG_CACHE_HOME",
		debugDir,
		filepath.Join(homeDir, ".cache"),
	)

	// ConfigHome の解決
	dirs.ConfigHome = resolveXDGPath(
		"XDG_CONFIG_HOME",
		debugDir,
		filepath.Join(homeDir, ".config"),
	)

	return dirs, nil
}

// resolveXDGPath はXDGパスを解決する
// 優先度: debugDir > XDG環境変数 > defaultBase
func resolveXDGPath(xdgEnvVar, debugDir, defaultBase string) string {
	// 1. debugDirが設定されている場合はそれを使用（デバッグ用、最優先）
	if debugDir != "" {
		return debugDir
	}

	// 2. XDG環境変数が設定されている場合はそれを使用
	if xdgPath := os.Getenv(xdgEnvVar); xdgPath != "" {
		return filepath.Join(xdgPath, AppName)
	}

	// 3. デフォルトパスを使用
	return filepath.Join(defaultBase, AppName)
}

// EnsureExists は全てのXDGディレクトリが存在することを確認し、存在しない場合は作成する
func (x *XDGDirs) EnsureExists() error {
	dirs := []string{x.StateHome, x.CacheHome, x.ConfigHome}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// CookiePath はCookieファイルのパスを返す
// CacheHomeに保存（再生成可能なデータのため）
// 注意: 実際のCookieファイル管理はbrowser/cookie/cookie.goで行われる
func (x *XDGDirs) CookiePath() string {
	return filepath.Join(x.CacheHome, "orm-mcp-go-cookies.json")
}

// ChromeDataDir はChrome一時データディレクトリのベースパスを返す
// 実際のディレクトリは chrome-user-data-{PID} の形式で作成される
// StateHomeに保存（セッション状態のため）
func (x *XDGDirs) ChromeDataDir() string {
	return x.StateHome
}

// ScreenshotDir はスクリーンショット保存ディレクトリのパスを返す
// StateHomeに保存（セッション状態のため）
func (x *XDGDirs) ScreenshotDir() string {
	return filepath.Join(x.StateHome, "screenshots")
}

// LogPath はログファイルのパスを返す
func (x *XDGDirs) LogPath() string {
	return filepath.Join(x.StateHome, "orm-mcp-go.log")
}

// ResearchHistoryPath は調査履歴ファイルのパスを返す
func (x *XDGDirs) ResearchHistoryPath() string {
	return filepath.Join(x.StateHome, "research-history.json")
}
