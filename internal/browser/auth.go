package browser

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/cookie"
)

// visibleLoginTempDir はビジブルログイン用の一時ディレクトリパスを返す。
// PIDベースのユニークパスにすることで、複数プロセス同時実行時のディレクトリ衝突を防ぐ。
func visibleLoginTempDir(stateDir string) string {
	return filepath.Join(stateDir, fmt.Sprintf("chrome-setup-%d", os.Getpid()))
}

// errUnauthenticated は HTTP 401/403 による認証失敗を表すセンチネルエラー。
// ネットワークエラーとは区別され、Cookie の削除判断に使用される。
var errUnauthenticated = errors.New("認証されていません (401/403)")

const ormHome = "https://learning.oreilly.com/home/"

// NewBrowserClient は新しいブラウザクライアントを作成します。
// Cookie が無効またはない場合は、ビジブルブラウザを起動してユーザーに手動ログインを促します。
// stateDir: XDG StateHome (Chrome一時データ用)
func NewBrowserClient(cookieManager cookie.Manager, debug bool, stateDir string) (*BrowserClient, error) {
	client := &BrowserClient{
		httpClient: &http.Client{
			Timeout: APIOperationTimeout,
			Transport: &GzipTransport{
				Transport: http.DefaultTransport,
			},
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		stateDir:  stateDir,
		debug:     debug,
	}

	// Cookieの復元を試行
	if cookieManager.CookieFileExists() {
		slog.Info("既存のCookieファイルが見つかりました。復元を試行します")
		if err := cookieManager.LoadCookies(); err != nil {
			slog.Warn("Cookie復元に失敗しました", "error", err)
		} else {
			// cookie.Managerをクライアントに設定
			client.cookieManager = cookieManager

			// HTTPリクエストでCookieが有効かどうか検証（chromedp不要）
			if client.validateAuthenticationViaHTTP() == nil {
				slog.Info("Cookieを使用してログインが完了しました")
				return client, nil
			}
			slog.Info("Cookieが無効でした。ビジブルブラウザでログインを実行します")
		}
	}

	// ビジブルブラウザでログインを実行
	client.cookieManager = cookieManager
	if err := RunVisibleLogin(visibleLoginTempDir(stateDir), cookieManager); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	slog.Info("ブラウザクライアントの初期化とログインが完了しました")
	return client, nil
}

// Close はブラウザクライアントをクリーンアップします
func (bc *BrowserClient) Close() {
	// httpClient と cookieManager はクリーンアップ不要
}

// Reauthenticate はCookie有効期限切れ時にビジブルブラウザを起動して再認証します
func (bc *BrowserClient) Reauthenticate() error {
	slog.Info("Cookie有効期限切れ検出: ビジブルブラウザで再認証を開始します")

	if err := RunVisibleLogin(visibleLoginTempDir(bc.stateDir), bc.cookieManager); err != nil {
		return fmt.Errorf("再認証に失敗しました: %w", err)
	}

	// ログイン成功後にCookieの有効性をHTTPで検証する
	if err := bc.validateAuthenticationViaHTTP(); err != nil {
		return fmt.Errorf("再認証後のCookie検証に失敗しました: %w", err)
	}

	slog.Info("再認証が完了しました")
	return nil
}

// validateAuthenticationViaHTTP はHTTPリクエストでCookieの有効性を検証します。
// chromedpを使用せずにHTTPクライアントで認証を検証する。
// 戻り値:
//   - nil: 認証成功 (200)
//   - errUnauthenticated: 401/403 による認証失敗 (Cookie を削除すべき)
//   - その他エラー: ネットワーク障害や予期しないレスポンス (Cookie は保持すべき)
func (bc *BrowserClient) validateAuthenticationViaHTTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), AuthValidationTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ormHome, nil)
	if err != nil {
		return fmt.Errorf("認証検証リクエスト作成に失敗: %w", err)
	}

	// ヘッダー設定
	req.Header.Set("User-Agent", bc.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	// Cookie を設定
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
		return fmt.Errorf("認証検証リクエストに失敗: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Warn("レスポンスボディのクローズに失敗", "error", cerr)
		}
	}()

	// 401/403 は認証失敗 (Cookie が無効)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		slog.Info("認証が無効です", "status", resp.StatusCode)
		return errUnauthenticated
	}

	// 200 は認証成功
	if resp.StatusCode == http.StatusOK {
		slog.Info("HTTP認証検証成功", "status", resp.StatusCode)
		return nil
	}

	// それ以外のステータスコード（302リダイレクトなど）はネットワーク的な問題として扱う
	slog.Warn("予期しないステータスコード", "status", resp.StatusCode)
	return fmt.Errorf("予期しないステータスコード: %d", resp.StatusCode)
}

// CheckAndResetAuth はCookieの有効性を検証し、期限切れの場合はCookieファイルを削除します。
// 認証済みの場合は nil を返します。
// 401/403 が確定した場合のみ stale Cookie を削除してエラーを返します。
// ネットワークエラーの場合は Cookie を保持したままエラーを返します。
func (bc *BrowserClient) CheckAndResetAuth() error {
	err := bc.validateAuthenticationViaHTTP()
	if err == nil {
		return nil
	}
	if errors.Is(err, errUnauthenticated) {
		// 401/403 が確定した場合のみ Cookie を削除
		if delErr := bc.cookieManager.DeleteCookieFile(); delErr != nil {
			slog.Warn("期限切れCookieの削除に失敗", "error", delErr)
		}
	}
	// ネットワークエラーの場合は Cookie を保持したままエラーを返す
	return fmt.Errorf("cookieが無効です。再認証が必要です: %w", err)
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
			slog.Debug("API呼び出し準備", "url", req.URL.String(), "cookie_count", len(cookies)) // #nosec G706 -- debug log, values are from internal API client
			if referer != "" {
				slog.Debug("Referer設定", "referer", referer)
			}
			for _, cookie := range cookies {
				value := cookie.Value
				if len(value) > 20 {
					value = value[:20] + "..."
				}
				slog.Debug("送信Cookie", "name", cookie.Name, "value", value, "domain", cookie.Domain, "path", cookie.Path) // #nosec G706 -- debug log, cookie values are from local cookie jar
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, contentURL, nil)
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
