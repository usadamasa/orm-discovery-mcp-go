package main

import (
	"context"
	"encoding/json"
	"fmt"
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
func runSetupCookies() {
	fmt.Println("=== O'Reilly Cookie セットアップ ===")
	fmt.Println("通常のChrome/Safariで手動ログインして、Cookieを自動保存します。")
	fmt.Println()

	// XDGディレクトリを解決（OREILLY_USER_ID/PASSWORDは不要）
	xdgDirs, err := GetXDGDirs(os.Getenv("ORM_MCP_GO_DEBUG_DIR"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: XDGディレクトリの解決に失敗しました: %v\n", err)
		os.Exit(1)
	}

	if err := xdgDirs.EnsureExists(); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: XDGディレクトリの作成に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// Chrome実行ファイルを検索
	chromePath, err := findSystemChrome()
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		fmt.Fprintf(os.Stderr, "Google Chrome をインストールしてください: https://www.google.com/chrome/\n")
		os.Exit(1)
	}
	slog.Info("Chromeを発見しました", "path", chromePath)

	// セットアップ用データディレクトリ
	setupDataDir := xdgDirs.ChromeSetupDataDir()

	// Chrome を起動 (URLはchromedpでナビゲートするためここでは指定しない)
	// --no-first-run: 初回セットアップウィザードを抑制
	// --no-default-browser-check: デフォルトブラウザチェックを抑制
	cmd := exec.Command(
		chromePath,
		"--remote-debugging-port="+cdpDebugPort,
		"--user-data-dir="+setupDataDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: Chrome の起動に失敗しました: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				slog.Warn("Chromeプロセスの終了に失敗", "error", err)
			}
		}
		if err := os.RemoveAll(setupDataDir); err != nil {
			slog.Warn("セットアップデータディレクトリの削除に失敗", "path", setupDataDir, "error", err)
		}
	}()

	fmt.Printf("Chrome を起動しました (PID: %d)\n", cmd.Process.Pid)

	// CDP 接続待機
	fmt.Println("CDP サーバーへの接続を待機中...")
	wsURL, err := waitForCDP(cdpDebugPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: CDP接続に失敗しました: %v\n", err)
		fmt.Fprintf(os.Stderr, "ヒント: ポート %s が既に使用中の場合は、MCP サーバーを停止してから再試行してください\n", cdpDebugPort)
		os.Exit(1)
	}
	slog.Info("CDP接続確立", "ws_url", wsURL)

	// 既存 Chrome プロセスに接続
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	defer allocCancel()

	// chromedp.NewContext はブラウザ内に新しいタブを作成する。
	// このタブを直接ログインページにナビゲートすることで、
	// 監視対象タブ = ユーザーが操作するタブ が一致する。
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	// ログインページへナビゲート
	fmt.Printf("ログインページを開いています: %s\n\n", ormLoginURL)
	if err := chromedp.Run(ctx, chromedp.Navigate(ormLoginURL)); err != nil {
		slog.Warn("ログインページへのナビゲートに失敗しました。手動で開いてください", "url", ormLoginURL, "error", err)
		fmt.Fprintf(os.Stderr, "警告: ログインページを自動で開けませんでした。\n手動で %s を開いてログインしてください。\n\n", ormLoginURL)
	}

	// ログイン完了を待機
	fmt.Printf("ログイン完了を待機中... (最大 %.0f 分)\n", loginWaitTimeout.Minutes())
	fmt.Println("ブラウザで O'Reilly にログインしてください。")
	if err := waitForLoginCompletion(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: ログイン完了の待機に失敗しました: %v\n", err)
		os.Exit(1)
	}

	// Cookie を保存
	cookieManager := cookie.NewCookieManager(xdgDirs.CacheHome)
	if err := cookieManager.SaveCookies(&ctx); err != nil {
		fmt.Fprintf(os.Stderr, "エラー: Cookieの保存に失敗しました: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("✓ Cookieを保存しました: %s\n", xdgDirs.CookiePath())
	fmt.Println("次回から `orm-discovery-mcp-go` を実行すると、Cookieでログインできます。")
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
	cdpVersionURL := fmt.Sprintf("http://localhost:%s/json/version", port)

	for time.Now().Before(deadline) {
		wsURL, err := fetchCDPWebSocketURL(cdpVersionURL)
		if err == nil && wsURL != "" {
			return wsURL, nil
		}
		time.Sleep(cdpPollInterval)
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

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("CDP レスポンスのパースに失敗: %w", err)
	}

	return result.WebSocketDebuggerURL, nil
}

// waitForLoginCompletion は learning.oreilly.com への遷移を検出してログイン完了を判定します
func waitForLoginCompletion(ctx context.Context) error {
	deadline := time.Now().Add(loginWaitTimeout)

	for time.Now().Before(deadline) {
		var currentURL string
		if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			slog.Debug("URL取得エラー (継続)", "error", err)
			time.Sleep(loginPollInterval)
			continue
		}

		if strings.Contains(currentURL, ormLearningURLPart) {
			fmt.Println("✓ ログイン完了を確認しました")
			return nil
		}

		fmt.Printf("ログイン待機中... (現在のURL: %s)\n", currentURL)
		time.Sleep(loginPollInterval)
	}

	return fmt.Errorf("ログイン完了の待機がタイムアウトしました (%.0f 分)", loginWaitTimeout.Minutes())
}
