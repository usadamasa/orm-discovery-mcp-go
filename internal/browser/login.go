package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/cookie"
)

const (
	// CDPWaitTimeout はCDP接続待機タイムアウト
	CDPWaitTimeout    = 30 * time.Second
	cdpPollInterval   = 1 * time.Second
	cdpRequestTimeout = 3 * time.Second
)

// findAvailablePort はOS に空きポートを割り当ててもらい、そのポート番号を返す。
// 固定ポート (9222) では複数インスタンス起動時にポート衝突が起きるため、動的に割り当てる。
func findAvailablePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("空きポートの検索に失敗しました: %w", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			slog.Warn("リスナーのクローズに失敗", "error", err)
		}
	}()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", fmt.Errorf("ポート番号の取得に失敗しました: %w", err)
	}
	return port, nil
}

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

	if result.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("CDP エンドポイントが空の WebSocket URL を返しました")
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

// startChromeForLogin launches Chrome with a remote debugging port and returns the command,
// port, processDone channel, and any startup error.
func startChromeForLogin(chromePath, tempDir string) (*exec.Cmd, string, chan error, error) {
	port, err := findAvailablePort()
	if err != nil {
		return nil, "", nil, fmt.Errorf("CDPデバッグポートの取得に失敗しました: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return nil, "", nil, fmt.Errorf("一時ディレクトリの作成に失敗しました: %w", err)
	}

	cmd := exec.Command(
		chromePath,
		"--remote-debugging-port="+port,
		"--user-data-dir="+tempDir,
		"--no-first-run",
		"--no-default-browser-check",
	)
	if err := cmd.Start(); err != nil {
		return nil, "", nil, fmt.Errorf("chromeの起動に失敗しました: %w", err)
	}
	slog.Info("Chrome を起動しました", "pid", cmd.Process.Pid, "cdp_port", port)

	processDone := make(chan error, 1)
	go func() {
		processDone <- cmd.Wait()
	}()

	return cmd, port, processDone, nil
}

// extractAndValidateCookies extracts cookies from the CDP session and validates
// that authentication cookies (orm-jwt or groot_sessionid) are present.
// Returns nil cookies if validation fails (caller should continue polling).
func extractAndValidateCookies(ctx context.Context) ([]*http.Cookie, error) {
	var cookies []*http.Cookie
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
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
			if !c.Session && c.Expires > 0 {
				cookies[i].Expires = time.Unix(int64(c.Expires), 0)
			}
		}
		slog.Info("Cookieを取得しました", "count", len(cookies))
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("cookie取得に失敗しました: %w", err)
	}

	if len(cookies) == 0 {
		slog.Debug("Cookieが空です。ポーリングを継続します")
		return nil, nil
	}
	for _, c := range cookies {
		if c.Name == "orm-jwt" || c.Name == "groot_sessionid" {
			return cookies, nil
		}
	}
	slog.Debug("認証Cookie (orm-jwt/groot_sessionid) が見つかりません。ポーリングを継続します")
	return nil, nil
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

	cmd, port, processDone, err := startChromeForLogin(chromePath, tempDir)
	if err != nil {
		return nil, err
	}
	processExited := false

	defer func() {
		if !processExited {
			if killErr := cmd.Process.Kill(); killErr != nil {
				slog.Warn("Chrome プロセスの終了に失敗", "error", killErr)
			}
			<-processDone
		}
		if rmErr := os.RemoveAll(tempDir); rmErr != nil {
			slog.Warn("一時ディレクトリの削除に失敗", "path", tempDir, "error", rmErr)
		}
	}()

	wsURL, err := WaitForCDPWithTimeout(port, CDPWaitTimeout)
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

	if err := chromedp.Run(loginCtx, chromedp.Navigate("https://www.oreilly.com/member/login/")); err != nil {
		return nil, fmt.Errorf("ログインページへの遷移に失敗しました: %w", err)
	}

	slog.Info("ブラウザでO'Reillyにログインしてください。ログイン完了後、自動的に次の処理に進みます",
		"login_url", "https://www.oreilly.com/member/login/",
		"timeout_minutes", int(VisibleLoginTimeout.Minutes()),
	)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-loginCtx.Done():
			return nil, fmt.Errorf("手動ログインがタイムアウトしました（%.0f分）。再度お試しください", VisibleLoginTimeout.Minutes())
		case waitErr := <-processDone:
			processExited = true
			if waitErr == nil {
				return nil, fmt.Errorf("ログイン完了前にChromeが閉じられました。再度コマンドを実行してログインしてください")
			}
			return nil, fmt.Errorf("chromeが予期せず終了しました: %w", waitErr)
		case <-ticker.C:
			var currentURL string
			if err := chromedp.Run(loginCtx, chromedp.Location(&currentURL)); err != nil {
				slog.Debug("URL取得エラー (ポーリング中)", "error", err)
				continue
			}

			slog.Debug("ログイン待機中", "current_url", currentURL)

			if strings.Contains(currentURL, "learning.oreilly.com") {
				slog.Info("ログイン完了を確認しました", "url", currentURL)

				cookies, err := extractAndValidateCookies(loginCtx)
				if err != nil {
					return nil, err
				}
				if cookies == nil {
					continue
				}
				return cookies, nil
			}
		}
	}
}
