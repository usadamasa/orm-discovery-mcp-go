package browser

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const ormHome = "https://learning.oreilly.com/home/"

// GzipTransport is a custom transport that automatically handles gzip decompression
type GzipTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface with automatic gzip decompression
func (g *GzipTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := g.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Check if response is gzip compressed
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("failed to create gzip reader: %w", err)
		}

		// Create a new response with decompressed body
		resp.Body = &gzipReadCloser{
			Reader: gzipReader,
			Closer: resp.Body,
		}

		// Remove Content-Encoding header since we've decompressed
		resp.Header.Del("Content-Encoding")
	}

	return resp, nil
}

// gzipReadCloser wraps a gzip.Reader and ensures both gzip reader and original body are closed
type gzipReadCloser struct {
	io.Reader
	Closer io.Closer
}

func (grc *gzipReadCloser) Close() error {
	// Close the gzip reader first
	if gzipReader, ok := grc.Reader.(*gzip.Reader); ok {
		if err := gzipReader.Close(); err != nil {
			slog.Warn("gzipリーダーのクローズに失敗", "error", err)
		}
	}

	// Then close the original response body
	if grc.Closer != nil {
		return grc.Closer.Close()
	}

	return nil
}

// NewBrowserClient は新しいブラウザクライアントを作成し、ログインを実行します
func NewBrowserClient(userID, password string, cookieManager cookie.Manager, debug bool, tmpDir string) (*BrowserClient, error) {
	if userID == "" || password == "" {
		return nil, fmt.Errorf("OREILLY_USER_ID and OREILLY_PASSWORD are required")
	}

	// ヘッドレスブラウザのコンテキストを作成
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(filepath.Join(tmpDir, "chrome-user-data")),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	client := &BrowserClient{
		ctx:         ctx,
		ctxCancel:   ctxCancel,
		allocCancel: allocCancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &GzipTransport{
				Transport: http.DefaultTransport,
			},
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		tmpDir:    tmpDir,
		debug:     debug,
	}

	// Cookieの復元を試行
	if cookieManager.CookieFileExists() {
		slog.Info("既存のCookieファイルが見つかりました。復元を試行します")
		if err := cookieManager.LoadCookies(&ctx); err != nil {
			slog.Warn("Cookie復元に失敗しました", "error", err)
		} else {
			// ブラウザのCookieをHTTPクライアントに同期
			client.syncCookiesFromBrowser()

			// Cookieが有効かどうか検証
			if client.validateAuthentication(ctx) {
				slog.Info("Cookieを使用してログインが完了しました")
				client.cookieManager = cookieManager

				// デバッグモードでなければ、ブラウザをクローズ
				if !debug {
					slog.Info("非デバッグモード: ブラウザコンテキストをクローズします")
					client.Close()
				}

				return client, nil
			}
			slog.Info("Cookieが無効でした。通常のログインを実行します")
		}
	}

	// 通常のログインを実行
	if err := client.login(userID, password); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// ログイン成功後にCookieを保存
	client.cookieManager = cookieManager
	if err := cookieManager.SaveCookies(&ctx); err != nil {
		slog.Warn("Cookieの保存に失敗しました", "error", err)
	}

	// ブラウザのCookieをHTTPクライアントに同期
	client.syncCookiesFromBrowser()

	slog.Info("ブラウザクライアントの初期化とログインが完了しました")

	// デバッグモードでなければ、ブラウザをクローズ
	if !debug {
		slog.Info("非デバッグモード: ブラウザコンテキストをクローズします")
		client.Close()
	}

	return client, nil
}

// Close はブラウザクライアントをクリーンアップします
func (bc *BrowserClient) Close() {
	// 正しい順序でクリーンアップ: ctx → allocator
	if bc.ctxCancel != nil {
		bc.ctxCancel()
	}
	if bc.allocCancel != nil {
		bc.allocCancel()
	}
}

// ReauthenticateIfNeeded はCookie有効期限切れ時にブラウザを再起動して再認証します
func (bc *BrowserClient) ReauthenticateIfNeeded(userID, password string) error {
	slog.Info("Cookie有効期限切れ検出: 再認証を開始します")

	// 一時的にブラウザを起動
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(filepath.Join(bc.tmpDir, "chrome-user-data")),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent(bc.userAgent),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	defer ctxCancel()

	// ブラウザコンテキストを一時的に更新
	bc.ctx = ctx
	bc.ctxCancel = ctxCancel
	bc.allocCancel = allocCancel

	// ログイン実行
	if err := bc.login(userID, password); err != nil {
		return fmt.Errorf("再認証に失敗しました: %w", err)
	}

	// Cookie保存
	if err := bc.cookieManager.SaveCookies(&ctx); err != nil {
		slog.Warn("Cookieの保存に失敗しました", "error", err)
	}

	// ブラウザのCookieをHTTPクライアントに同期
	bc.syncCookiesFromBrowser()

	// 非デバッグモード時はすぐにクローズ
	if !bc.debug {
		bc.Close()
	}

	slog.Info("再認証が完了しました")
	return nil
}

// login はO'Reillyにログインし、セッションCookieを取得します
func (bc *BrowserClient) login(userID, password string) error {
	slog.Info("O'Reillyへのログインを開始します", "user_id", userID)

	var cookies []*http.Cookie
	var divText string

	err := chromedp.Run(bc.ctx,
		// ログインページに移動
		chromedp.Navigate("https://www.oreilly.com/member/login/"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Debug("ログインページに移動しました")
			return nil
		}),
		// メールアドレスの入力
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, userID, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			slog.Debug("メールアドレスを入力しました", "user_id", userID)
			bc.debugScreenshot(ctx, "orm_filled_email")
			slog.Debug("Continueボタンをクリックしようとしています")
			return nil
		}),
		// Continueボタンをクリック
		chromedp.WaitVisible(`.orm-Button-root`, chromedp.ByQuery),
		chromedp.Click(`.orm-Button-root`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// クリック操作
			bc.debugScreenshot(ctx, "orm_clicked_continue")
			slog.Debug("Continueボタンをクリックしました")
			return nil
		}),
		// リダイレクトまたはページ更新を待機
		chromedp.WaitVisible(`.sub-title`, chromedp.ByQuery),
		chromedp.Text(`.sub-title`, &divText, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			bc.debugScreenshot(ctx, "acm_after_redirected")
			slog.Debug(".sub-title取得", "text", divText)
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err != nil {
				return err
			}
			if strings.Contains(currentURL, "idp.acm.org") {
				slog.Info("ACM IDPにリダイレクトされました", "url", currentURL)
			} else {
				slog.Error("想定されたログインフローが見つかりませんでした", "current_url", currentURL)
				return fmt.Errorf("想定されたログインフローが見つかりませんでした。現在のURL: %s", currentURL)
			}
			return nil
		}),
		// ACM IDPでログイン
		chromedp.ActionFunc(func(ctx context.Context) error {
			// @acm.orgを除いたユーザー名を取得
			username := strings.TrimSuffix(userID, "@acm.org")
			slog.Debug("ACMユーザー名を取得", "username", username)

			return chromedp.Run(ctx,
				// ユーザー名フィールドを待機
				chromedp.WaitVisible(`input[placeholder*="username"]`, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					slog.Debug("ACMユーザー名フィールドが表示されました")
					return nil
				}),
				// ユーザー名を入力
				chromedp.Clear(`input[placeholder*="username"]`, chromedp.ByQuery),
				chromedp.SendKeys(`input[placeholder*="username"]`, username, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					slog.Debug("ACMユーザー名を入力しました", "username", username)
					return nil
				}),
				// パスワードを入力
				chromedp.SendKeys(`input[placeholder*="password"]`, password, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					slog.Debug("ACMパスワードを入力しました")
					return nil
				}),
				chromedp.ActionFunc(func(ctx context.Context) error {
					bc.debugScreenshot(ctx, "acm_filled")
					return nil
				}),
				// Sign inボタンをクリック
				chromedp.Click(`.btn`, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					slog.Debug("ACM Sign inボタンをクリックしました")
					return nil
				}),
			)
		}),
		// ログイン完了まで待機
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 最大60秒待機（時間を延長）
			timeout := time.Now().Add(60 * time.Second)
			bc.debugScreenshot(ctx, "acm_login_completed")
			for time.Now().Before(timeout) {
				var currentURL string
				if err := chromedp.Location(&currentURL).Do(ctx); err != nil {
					slog.Debug("URL取得エラー", "error", err)
					time.Sleep(2 * time.Second)
					continue
				}

				slog.Debug("ログイン処理中", "url", currentURL)

				// ログイン成功の判定
				if strings.Contains(currentURL, "learning.oreilly.com") ||
					strings.Contains(currentURL, "oreilly.com/home") ||
					strings.Contains(currentURL, "oreilly.com/member") {
					slog.Info("ログイン成功を確認しました", "final_url", currentURL)
					return nil
				}

				// エラーページの確認
				if strings.Contains(currentURL, "error") || strings.Contains(currentURL, "denied") {
					return fmt.Errorf("ログインエラーページが検出されました: %s", currentURL)
				}

				time.Sleep(2 * time.Second)
			}

			return fmt.Errorf("ログインがタイムアウトしました（60秒）")
		}),

		// Cookieを取得
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookiesResp, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			// cdproto.Cookieから標準のhttp.Cookieに変換
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
		}),
	)

	if err != nil {
		return fmt.Errorf("ログイン処理でエラーが発生しました: %w", err)
	}

	return nil
}

// validateAuthentication はCookieが有効かどうかを検証します
func (bc *BrowserClient) validateAuthentication(ctx context.Context) bool {
	var pageTitle string

	var currentURL string
	err := chromedp.Run(bc.ctx,
		// 認証が必要なページにアクセス
		chromedp.Navigate(ormHome),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Title(&pageTitle),
		chromedp.Location(&currentURL),
	)
	bc.debugScreenshot(ctx, "validate_saved_cookie_authentication")

	if err != nil {
		slog.Warn("認証検証中にエラーが発生しました", "error", err)
		return false
	}
	slog.Debug("認証検証中", "url", currentURL, "title", pageTitle)

	// ログインページにリダイレクトされていないかチェック
	if currentURL != ormHome {
		slog.Info("認証が無効です", "current_url", currentURL, "expected_url", ormHome)
		return false
	}

	slog.Info("認証検証成功", "title", pageTitle)
	return true
}

// syncCookiesFromBrowser はブラウザのCookieをcookie.Managerに同期します
func (bc *BrowserClient) syncCookiesFromBrowser() {
	var cookies []*network.Cookie
	err := chromedp.Run(bc.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))

	if err != nil {
		slog.Warn("ブラウザからのCookie取得に失敗しました", "error", err)
		return
	}

	// ブラウザのCookieをcookie.Managerに設定
	var httpCookies []*http.Cookie
	for _, c := range cookies {
		httpCookie := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}

		if c.Expires != 0 {
			httpCookie.Expires = time.Unix(int64(c.Expires), 0)
		}

		httpCookies = append(httpCookies, httpCookie)
	}

	// O'Reilly関連のURLでCookieを設定
	urls := []*url.URL{
		{Scheme: "https", Host: "learning.oreilly.com"},
		{Scheme: "https", Host: "www.oreilly.com"},
		{Scheme: "https", Host: "oreilly.com"},
	}

	if bc.cookieManager != nil {
		for _, u := range urls {
			if err := bc.cookieManager.SetCookies(u, httpCookies); err != nil {
				slog.Warn("cookie.ManagerへのCookie設定に失敗", "url", u.String(), "error", err)
			}
		}
	}

	// デバッグログ
	if bc.debug {
		slog.Debug("cookie.ManagerにCookieを同期しました", "count", len(httpCookies))
		for _, c := range httpCookies {
			value := c.Value
			if len(value) > 20 {
				value = value[:20] + "..."
			}
			slog.Debug("Cookie同期", "name", c.Name, "value", value, "domain", c.Domain, "path", c.Path)
		}
	}

}

// CreateRequestEditor creates a standardized RequestEditorFn for API calls
func (bc *BrowserClient) CreateRequestEditor() func(ctx context.Context, req *http.Request) error {
	return bc.createRequestEditorInternal("")
}

// CreateRequestEditorWithReferer creates a standardized RequestEditorFn with custom Referer
func (bc *BrowserClient) CreateRequestEditorWithReferer(referer string) func(ctx context.Context, req *http.Request) error {
	return bc.createRequestEditorInternal(referer)
}

// createRequestEditorInternal is the internal implementation for creating request editors
func (bc *BrowserClient) createRequestEditorInternal(referer string) func(ctx context.Context, req *http.Request) error {
	return func(ctx context.Context, req *http.Request) error {
		// Set comprehensive browser-matching headers
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "ja,en-US;q=0.7,en;q=0.3")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
		req.Header.Set("Content-Type", "application/json")

		// Set Referer only if provided
		if referer != "" {
			req.Header.Set("Referer", referer)
		}

		req.Header.Set("Origin", "https://learning.oreilly.com")
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Priority", "u=0")
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("User-Agent", bc.userAgent)

		// Get cookies from cookie.Manager and set them manually
		cookies := bc.cookieManager.GetCookiesForURL(req.URL)

		// Set Cookie header manually
		if len(cookies) > 0 {
			var cookieValues []string
			for _, cookie := range cookies {
				cookieValues = append(cookieValues, cookie.Name+"="+cookie.Value)
			}
			req.Header.Set("Cookie", strings.Join(cookieValues, "; "))
		}

		// Debug logging for cookie transmission
		if bc.debug {
			slog.Debug("API呼び出し準備", "url", req.URL.String(), "cookie_count", len(cookies))
			if referer != "" {
				slog.Debug("Referer設定", "referer", referer)
			}
			for _, cookie := range cookies {
				value := cookie.Value
				if len(value) > 20 {
					value = value[:20] + "..."
				}
				slog.Debug("送信Cookie", "name", cookie.Name, "value", value, "domain", cookie.Domain, "path", cookie.Path)
			}
		}

		return nil
	}
}

// GetContentFromURL retrieves HTML/XHTML content from the specified URL with authentication
func (bc *BrowserClient) GetContentFromURL(contentURL string) (string, error) {
	// Determine content type from URL
	contentType := "HTML"
	if strings.HasSuffix(contentURL, ".xhtml") {
		contentType = "XHTML"
	} else if strings.Contains(contentURL, "/files/html/") {
		contentType = "HTML (nested path)"
	}

	slog.Info("コンテンツを取得しています", "type", contentType, "url", contentURL)

	req, err := http.NewRequest("GET", contentURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for HTML response (try different accept headers)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", bc.userAgent)

	// Add authentication cookies manually using cookie.Manager
	cookies := bc.cookieManager.GetCookiesForURL(req.URL)
	if len(cookies) > 0 {
		var cookieValues []string
		for _, cookie := range cookies {
			cookieValues = append(cookieValues, cookie.Name+"="+cookie.Value)
		}
		req.Header.Set("Cookie", strings.Join(cookieValues, "; "))
	}

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("レスポンスボディのクローズに失敗", "error", err)
		}
	}()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("authentication error: status %d (cookies may have expired)", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("content request failed with status %d", resp.StatusCode)
	}

	// Handle gzip compression
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer func() {
			if err := gzipReader.Close(); err != nil {
				slog.Warn("gzipリーダーのクローズに失敗", "error", err)
			}
		}()
		reader = gzipReader
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read content body: %w", err)
	}

	return string(bodyBytes), nil
}
