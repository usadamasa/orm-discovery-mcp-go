// Package chromedp provides lifecycle management for ChromeDP browser instances.
package chromedp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/chromedp"
)

// Timeout constants for ChromeDP operations
const (
	ExecAllocatorTimeout = 45 * time.Second
)

// Manager manages ChromeDP browser lifecycle with process-specific user data directories.
//
// Invariants:
// - chromeDataDir always has format "chrome-user-data-{PID}" where {PID} is os.Getpid()
// - All three cancel functions (ctxCancel, allocCancel, baseCancel) are from the same context chain
// - ctx is valid only until Close() is called
// - Close() should be called only once (subsequent calls are silently ignored)
type Manager struct {
	tmpDir        string
	chromeDataDir string
	allocCtx      context.Context
	allocCancel   context.CancelFunc
	ctx           context.Context
	ctxCancel     context.CancelFunc
	baseCancel    context.CancelFunc
	closed        bool
}

// NewManager creates a new ChromeDP lifecycle manager.
// It automatically cleans up old Chrome data directories from previous processes.
func NewManager(tmpDir string, debug bool) (*Manager, error) {
	// 古いChromeデータディレクトリのクリーンアップ
	// クリーンアップ失敗は致命的ではないが、ログに記録する
	if err := cleanupOldChromeDataDirs(tmpDir); err != nil {
		slog.Warn("古いChromeデータディレクトリのクリーンアップに失敗しましたが、続行します",
			"path", tmpDir,
			"error", err)
	}

	// プロセス固有のユーザーデータディレクトリを生成
	chromeDataDir := filepath.Join(tmpDir, fmt.Sprintf("chrome-user-data-%d", os.Getpid()))
	slog.Debug("Chromeユーザーデータディレクトリ", "path", chromeDataDir)

	// ChromeDPオプションの設定
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(chromeDataDir),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// タイムアウト付きコンテキストでブラウザを起動
	baseCtx, baseCancel := context.WithTimeout(context.Background(), ExecAllocatorTimeout)

	// コンテキストの作成
	allocCtx, allocCancel := chromedp.NewExecAllocator(baseCtx, opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	return &Manager{
		tmpDir:        tmpDir,
		chromeDataDir: chromeDataDir,
		allocCtx:      allocCtx,
		allocCancel:   allocCancel,
		ctx:           ctx,
		ctxCancel:     ctxCancel,
		baseCancel:    baseCancel,
	}, nil
}

// Context returns the ChromeDP context for browser operations.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Close closes the browser and cleans up the user data directory.
// Subsequent calls are silently ignored.
func (m *Manager) Close() {
	if m.closed {
		slog.Debug("Manager is already closed")
		return
	}
	m.closed = true

	if m.ctxCancel != nil {
		m.ctxCancel()
	}
	if m.allocCancel != nil {
		m.allocCancel()
	}
	if m.baseCancel != nil {
		m.baseCancel()
	}

	// プロセス固有のChromeデータディレクトリを削除
	if m.chromeDataDir != "" {
		if err := os.RemoveAll(m.chromeDataDir); err != nil {
			slog.Warn("Chromeデータディレクトリの削除に失敗", "path", m.chromeDataDir, "error", err)
		}
	}
}

// cleanupOldChromeDataDirs removes old Chrome data directories from terminated processes.
// Returns an error if the tmpDir cannot be read.
func cleanupOldChromeDataDirs(tmpDir string) error {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		slog.Info("一時ディレクトリの読み取りに失敗", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to read tmpDir: %w", err)
	}

	currentPID := os.Getpid()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// chrome-user-data-{pid} 形式のディレクトリを検出
		if !strings.HasPrefix(name, "chrome-user-data-") {
			continue
		}

		// PIDを抽出
		pidStr := strings.TrimPrefix(name, "chrome-user-data-")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// PIDが数値でない場合は削除対象
			slog.Debug("不正な形式のChromeデータディレクトリ", "name", name)
			dirPath := filepath.Join(tmpDir, name)
			if removeErr := os.RemoveAll(dirPath); removeErr != nil {
				slog.Warn("不正な形式のChromeデータディレクトリの削除に失敗", "path", dirPath, "error", removeErr)
			}
			continue
		}

		// 現在のプロセスIDのディレクトリはスキップ
		if pid == currentPID {
			continue
		}

		// プロセスが実行中かつorm-discovery-mcp-goかどうかを確認
		if isOrmMcpGoProcess(pid) {
			slog.Debug("orm-discovery-mcp-goプロセスが実行中のためスキップ", "pid", pid)
			continue
		}

		// 実行中でない、または別のプロセスのディレクトリを削除
		dirPath := filepath.Join(tmpDir, name)
		slog.Info("古いChromeデータディレクトリを削除します", "path", dirPath, "pid", pid)
		if err := os.RemoveAll(dirPath); err != nil {
			slog.Warn("古いChromeデータディレクトリの削除に失敗", "path", dirPath, "error", err)
		}
	}

	// 旧形式ディレクトリ: SingletonLockがない場合のみ削除
	cleanupLegacyChromeDataDir(tmpDir)

	return nil
}

// isOrmMcpGoProcess checks if the given PID is a running orm-discovery-mcp-go process.
func isOrmMcpGoProcess(pid int) bool {
	// まずプロセスが存在するか確認
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// シグナル0で存在確認
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}

	// プロセス名を確認(macOS/Linux対応)
	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := os.ReadFile(commPath)
	if err != nil {
		// macOSでは/procがないため、psコマンドで確認
		return isOrmMcpGoProcessByPS(pid)
	}

	processName := strings.TrimSpace(string(data))
	return isMatchingProcessName(processName)
}

// isOrmMcpGoProcessByPS checks process name using ps command (for macOS).
func isOrmMcpGoProcessByPS(pid int) bool {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		slog.Debug("プロセス確認コマンド失敗", "pid", pid, "error", err)
		return false
	}
	processName := strings.TrimSpace(string(output))
	return isMatchingProcessName(processName)
}

// isMatchingProcessName checks if the process name matches orm-discovery-mcp-go binary.
// Uses filepath.Base to handle full paths and exact matching to avoid false positives.
func isMatchingProcessName(processName string) bool {
	baseName := filepath.Base(processName)
	// Exact match for the binary name
	return baseName == "orm-discovery-mcp-go"
}

// cleanupLegacyChromeDataDir safely removes the legacy fixed Chrome data directory.
func cleanupLegacyChromeDataDir(tmpDir string) {
	oldDir := filepath.Join(tmpDir, "chrome-user-data")
	lockFile := filepath.Join(oldDir, "SingletonLock")

	// ディレクトリが存在しない場合は何もしない
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return
	}

	// SingletonLockが存在する場合は削除しない(使用中の可能性)
	if _, err := os.Stat(lockFile); err == nil {
		slog.Warn("旧形式のChromeデータディレクトリにSingletonLockが存在するため削除をスキップ",
			"path", oldDir,
			"hint", "手動で削除してください: rm -rf "+oldDir)
		return
	}

	// SingletonLockがない場合は安全に削除
	slog.Info("旧形式のChromeデータディレクトリを削除します", "path", oldDir)
	if err := os.RemoveAll(oldDir); err != nil {
		slog.Warn("旧形式のChromeデータディレクトリの削除に失敗", "path", oldDir, "error", err)
	}
}
