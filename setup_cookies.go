package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

const (
	cdpDebugPort       = "9222"
	cdpWaitTimeout     = 30 * time.Second
	loginWaitTimeout   = 5 * time.Minute
	loginPollInterval  = 2 * time.Second
	cdpPollInterval    = 1 * time.Second
	cdpRequestTimeout  = 3 * time.Second
	ormLoginURL        = "https://www.oreilly.com/member/login/"
	ormLearningURLPart = "learning.oreilly.com"
)

// runSetupCookies は手動ログインからCookieを保存するセットアップフローを実行します
// OREILLY_USER_ID / OREILLY_PASSWORD は不要（手動ログインのため）
// CLI から呼ばれるエントリポイント (stdout に出力)
func runSetupCookies() error {
	return runSetupCookiesWithOutput(os.Stdout)
}

// runSetupCookiesWithOutput は出力先を指定して実行します
// サーバー内部から呼ぶ際は stderr を渡すことで stdio モードの MCP stream を汚染しません
func runSetupCookiesWithOutput(out io.Writer) error {
	fmt.Fprintln(out, "=== O'Reilly Cookie セットアップ ===")
	fmt.Fprintln(out)

	// XDGディレクトリを解決（OREILLY_USER_ID/PASSWORDは不要）
	xdgDirs, err := GetXDGDirs(os.Getenv("ORM_MCP_GO_DEBUG_DIR"))
	if err != nil {
		return fmt.Errorf("XDGディレクトリの解決に失敗しました: %w", err)
	}

	if err := xdgDirs.EnsureExists(); err != nil {
		return fmt.Errorf("XDGディレクトリの作成に失敗しました: %w", err)
	}

	chromePath, err := findSystemChrome()
	if err != nil {
		return fmt.Errorf("%w\nGoogle Chrome をインストールしてください: https://www.google.com/chrome/", err)
	}
	slog.Info("Chromeを発見しました", "path", chromePath)

	// 一時プロファイルで Chrome を起動する
	// ログイン後に Chrome を終了し一時ディレクトリを削除する
	tempDir := xdgDirs.ChromeSetupDataDir()
	fmt.Fprintln(out, "Chrome を起動してログインページを開きます。ログインするとCookieを自動保存します。")

	cmd := exec.Command(
		chromePath,
		"--remote-debugging-port="+cdpDebugPort,
		"--user-data-dir="+tempDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("chrome の起動に失敗しました: %w", err)
	}
	fmt.Fprintf(out, "Chrome を起動しました (PID: %d)\n", cmd.Process.Pid)
	defer func() {
		if killErr := cmd.Process.Kill(); killErr != nil {
			slog.Warn("Chrome プロセスの終了に失敗", "error", killErr)
		}
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			slog.Warn("一時ディレクトリの削除に失敗", "path", tempDir, "error", rmErr)
		}
	}()

	// CDP 接続待機
	fmt.Fprintln(out, "CDP サーバーへの接続を待機中...")
	wsURL, err := waitForCDP(cdpDebugPort)
	if err != nil {
		return fmt.Errorf("CDP 接続に失敗しました: %w", err)
	}
	slog.Info("CDP 接続確立", "ws_url", wsURL)

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	defer allocCancel()

	// chromedp.NewContext はブラウザ内に新しいタブを作成する。
	// このタブを直接ログインページにナビゲートすることで、
	// 監視対象タブ = ユーザーが操作するタブ が一致する。
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	// ログインページへナビゲート
	fmt.Fprintf(out, "ログインページを開いています: %s\n\n", ormLoginURL)
	if err := chromedp.Run(ctx, chromedp.Navigate(ormLoginURL)); err != nil {
		return fmt.Errorf("ログインページへのナビゲートに失敗しました: %w", err)
	}

	// ログイン完了を待機
	fmt.Fprintf(out, "ログイン完了を待機中... (最大 %.0f 分)\n", loginWaitTimeout.Minutes())
	fmt.Fprintln(out, "ブラウザで O'Reilly にログインしてください。")
	if err := waitForLoginCompletion(ctx, out); err != nil {
		return fmt.Errorf("ログイン完了の待機に失敗しました: %w", err)
	}

	// Cookie を保存
	cookieManager := cookie.NewCookieManager(xdgDirs.CacheHome)
	if err := cookieManager.SaveCookies(&ctx); err != nil {
		return fmt.Errorf("cookieの保存に失敗しました: %w", err)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "✓ Cookieを保存しました: %s\n", xdgDirs.CookiePath())
	fmt.Fprintln(out, "次回から `orm-discovery-mcp-go` を実行すると、Cookieでログインできます。")
	return nil
}

// findSystemChrome はシステムにインストールされている Chrome の実行ファイルパスを返します
func findSystemChrome() (string, error) {
	// macOS
	macOSPath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(macOSPath); err == nil {
		return macOSPath, nil
	}

	// Linux (PATH 検索)
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium-browser", "chromium"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Windows
	windowsPath := `C:\Program Files\Google\Chrome\Application\chrome.exe`
	if _, err := os.Stat(windowsPath); err == nil {
		return windowsPath, nil
	}

	return "", fmt.Errorf("このシステムにGoogle Chromeが見つかりませんでした (macOS/Linux/Windows を検索しましたが見つかりません)")
}

// waitForCDP はデフォルト30秒タイムアウトで CDP WebSocket URL が利用可能になるまで待機します
func waitForCDP(port string) (string, error) {
	return waitForCDPWithTimeout(port, cdpWaitTimeout)
}

// waitForCDPWithTimeout は指定したタイムアウトで CDP WebSocket URL が利用可能になるまで待機します
func waitForCDPWithTimeout(port string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	// Chrome は IPv4 127.0.0.1 でリッスンするため localhost (IPv6 [::1]) ではなく明示的に指定する
	cdpVersionURL := fmt.Sprintf("http://127.0.0.1:%s/json/version", port)
	var lastErr error

	for time.Now().Before(deadline) {
		wsURL, err := fetchCDPWebSocketURL(cdpVersionURL)
		if err == nil && wsURL != "" {
			return wsURL, nil
		}
		lastErr = err
		time.Sleep(cdpPollInterval)
	}

	if lastErr != nil {
		return "", fmt.Errorf("CDP サーバーへの接続がタイムアウトしました (ポート %s): %w", port, lastErr)
	}
	return "", fmt.Errorf("CDP サーバーへの接続がタイムアウトしました (ポート %s)", port)
}

// fetchCDPWebSocketURL は CDP エンドポイントから WebSocket URL を取得します
func fetchCDPWebSocketURL(cdpVersionURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cdpRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cdpVersionURL, nil)
	if err != nil {
		return "", fmt.Errorf("リクエスト作成に失敗: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("CDP エンドポイントへの接続に失敗: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("CDP レスポンスボディのクローズに失敗", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CDP エンドポイントが予期しないステータスを返しました: %d", resp.StatusCode)
	}

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("CDP レスポンスのパースに失敗: %w", err)
	}

	return result.WebSocketDebuggerURL, nil
}

// waitForLoginCompletion は learning.oreilly.com への遷移を検出してログイン完了を判定します
func waitForLoginCompletion(ctx context.Context, out io.Writer) error {
	deadline := time.Now().Add(loginWaitTimeout)

	for time.Now().Before(deadline) {
		var currentURL string
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			slog.Debug("URL取得エラー (継続)", "error", err)
			time.Sleep(loginPollInterval)
			continue
		}

		if strings.Contains(currentURL, ormLearningURLPart) {
			fmt.Fprintln(out, "✓ ログイン完了を確認しました")
			return nil
		}

		fmt.Fprintf(out, "ログイン待機中... (現在のURL: %s)\n", currentURL)
		time.Sleep(loginPollInterval)
	}

	return fmt.Errorf("ログイン完了の待機がタイムアウトしました (%.0f 分)", loginWaitTimeout.Minutes())
}
