package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

const (
	// CDPDebugPort はChrome DevTools Protocol のデバッグポート
	CDPDebugPort = "9222"
	// CDPWaitTimeout はCDP接続待機タイムアウト
	CDPWaitTimeout    = 30 * time.Second
	cdpPollInterval   = 1 * time.Second
	cdpRequestTimeout = 3 * time.Second
)

// FindSystemChrome はシステム Chrome のパスを返す (macOS / Linux のみ)
func FindSystemChrome() (string, error) {
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

	return "", fmt.Errorf("chromeが見つかりませんでした (macOS/Linux)")
}

// WaitForCDPWithTimeout は指定したタイムアウトで CDP WebSocket URL が利用可能になるまで待機する
func WaitForCDPWithTimeout(port string, timeout time.Duration) (string, error) {
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

// fetchCDPWebSocketURL は CDP エンドポイントから WebSocket URL を取得する
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

// oreillyDomainURLs は Cookie を設定する O'Reilly ドメインのリスト
var oreillyDomainURLs = []*url.URL{
	{Scheme: "https", Host: "learning.oreilly.com"},
	{Scheme: "https", Host: "www.oreilly.com"},
	{Scheme: "https", Host: "oreilly.com"},
}

// RunVisibleLogin は Chrome をネイティブ起動してユーザーの手動ログインを待ち、
// 取得した Cookie を cookie.Manager に保存する。
func RunVisibleLogin(tempDir string, cm cookie.Manager) error {
	cookies, err := runVisibleLogin(tempDir)
	if err != nil {
		return err
	}
	if err := cm.SaveCookiesFromData(cookies); err != nil {
		return fmt.Errorf("cookieの保存に失敗しました: %w", err)
	}
	for _, u := range oreillyDomainURLs {
		if err := cm.SetCookies(u, cookies); err != nil {
			slog.Warn("cookie.ManagerへのCookie設定に失敗", "url", u.String(), "error", err)
		}
	}
	return nil
}

// runVisibleLogin はビジブルChromeを起動し、ユーザーが手動ログインするまで待機する。
// ログイン完了は learning.oreilly.com への URL 遷移で検知する。
// exec.Command + NewRemoteAllocator を使用することで Akamai のボット検知を回避する。
func runVisibleLogin(tempDir string) ([]*http.Cookie, error) {
	slog.Info("ビジブルブラウザを起動します。ブラウザでO'Reillyにログインしてください",
		"url", "https://www.oreilly.com/member/login/",
		"timeout", VisibleLoginTimeout,
	)

	chromePath, err := FindSystemChrome()
	if err != nil {
		return nil, fmt.Errorf("chromeの検索に失敗しました: %w", err)
	}

	// 一時プロファイルで Chrome を起動する
	// ログイン後に Chrome を終了し一時ディレクトリを削除する
	cmd := exec.Command(
		chromePath,
		"--remote-debugging-port="+CDPDebugPort,
		"--user-data-dir="+tempDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("chromeの起動に失敗しました: %w", err)
	}
	slog.Info("Chrome を起動しました", "pid", cmd.Process.Pid)
	defer func() {
		if killErr := cmd.Process.Kill(); killErr != nil {
			slog.Warn("Chrome プロセスの終了に失敗", "error", killErr)
		}
		// Kill後にWaitを呼んでゾンビプロセスを回収する
		// Kill後のWait失敗 (例: "signal: killed") は正常なため Debug レベルで記録
		if waitErr := cmd.Wait(); waitErr != nil {
			slog.Debug("Chromeプロセスのwait結果", "error", waitErr)
		}
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			slog.Warn("一時ディレクトリの削除に失敗", "path", tempDir, "error", rmErr)
		}
	}()

	// CDP 接続待機
	wsURL, err := WaitForCDPWithTimeout(CDPDebugPort, CDPWaitTimeout)
	if err != nil {
		return nil, fmt.Errorf("CDP 接続に失敗しました: %w", err)
	}
	slog.Info("CDP 接続確立", "ws_url", wsURL)

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)
	defer allocCancel()

	chromeCtx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	loginCtx, loginCancel := context.WithTimeout(chromeCtx, VisibleLoginTimeout)
	defer loginCancel()

	// ログインページに遷移
	if err := chromedp.Run(loginCtx, chromedp.Navigate("https://www.oreilly.com/member/login/")); err != nil {
		return nil, fmt.Errorf("ログインページへの遷移に失敗しました: %w", err)
	}

	slog.Info("ブラウザでO'Reillyにログインしてください。ログイン完了後、自動的に次の処理に進みます",
		"login_url", "https://www.oreilly.com/member/login/",
		"timeout_minutes", int(VisibleLoginTimeout.Minutes()),
	)

	// 2秒間隔でポーリング: learning.oreilly.com への遷移を検知
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-loginCtx.Done():
			return nil, fmt.Errorf("手動ログインがタイムアウトしました（%.0f分）。再度お試しください", VisibleLoginTimeout.Minutes())
		case <-ticker.C:
			var currentURL string
			if err := chromedp.Run(loginCtx, chromedp.Location(&currentURL)); err != nil {
				slog.Debug("URL取得エラー (ポーリング中)", "error", err)
				continue
			}

			slog.Debug("ログイン待機中", "current_url", currentURL)

			if strings.Contains(currentURL, "learning.oreilly.com") {
				slog.Info("ログイン完了を確認しました", "url", currentURL)

				// Cookie を取得
				var cookies []*http.Cookie
				err := chromedp.Run(loginCtx, chromedp.ActionFunc(func(ctx context.Context) error {
					cookiesResp, err := network.GetCookies().Do(ctx)
					if err != nil {
						return err
					}
					cookies = make([]*http.Cookie, len(cookiesResp))
					for i, c := range cookiesResp {
						cookies[i] = &http.Cookie{
							Name:     c.Name,
							Value:    c.Value,
							Domain:   c.Domain,
							Path:     c.Path,
							Secure:   c.Secure,
							HttpOnly: c.HTTPOnly,
						}
					}
					slog.Info("Cookieを取得しました", "count", len(cookies))
					return nil
				}))
				if err != nil {
					return nil, fmt.Errorf("cookie取得に失敗しました: %w", err)
				}
				return cookies, nil
			}
		}
	}
}
