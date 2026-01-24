package cookie

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	cookieFileName = "orm-mcp-go-cookies.json"
	// CookieOperationTimeout はCookie操作のタイムアウト時間
	CookieOperationTimeout = 10 * time.Second
)

// Manager の前方宣言（main パッケージの構造体）
type Manager interface {
	SaveCookies(ctx *context.Context) error
	LoadCookies(ctx *context.Context) error
	CookieFileExists() bool
	DeleteCookieFile() error
	GetCookiesForURL(url *url.URL) []*http.Cookie
	SetCookies(url *url.URL, cookies []*http.Cookie) error
}

// entry はCookieの情報を保持する構造体
type entry struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires"`
	HTTPOnly bool      `json:"httpOnly"`
	Secure   bool      `json:"secure"`
}

// cookieCache はCookieキャッシュファイルの構造体
type cookieCache struct {
	Cookies []entry   `json:"cookies"`
	SavedAt time.Time `json:"saved_at"`
}

// ManagerImpl はCookieの保存と復元を管理する
type ManagerImpl struct {
	cacheDir string
	filePath string
	cookies  []*http.Cookie
}

// NewCookieManager は新しいCookieManagerを作成する
// cacheDir: XDG CacheHome ディレクトリ（例: ~/.cache/orm-mcp-go）
func NewCookieManager(cacheDir string) *ManagerImpl {
	return &ManagerImpl{
		cacheDir: cacheDir,
		filePath: filepath.Join(cacheDir, cookieFileName),
		cookies:  make([]*http.Cookie, 0),
	}
}

// SaveCookies はブラウザのCookieをファイルに保存する
func (cm *ManagerImpl) SaveCookies(ctx *context.Context) error {
	// Cookie保存操作にタイムアウトを設定
	saveCtx, saveCancel := context.WithTimeout(*ctx, CookieOperationTimeout)
	defer saveCancel()

	var cookies []*network.Cookie
	err := chromedp.Run(saveCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return fmt.Errorf("failed to get cookies from browser: %w", err)
	}

	// 重要なCookieのみをフィルタリング
	var filteredCookies []entry
	var httpCookies []*http.Cookie
	for _, cookie := range cookies {
		if cm.isImportantCookie(cookie.Name) {
			cookieData := entry{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   cookie.Domain,
				Path:     cookie.Path,
				HTTPOnly: cookie.HTTPOnly,
				Secure:   cookie.Secure,
			}
			if cookie.Expires != 0 {
				cookieData.Expires = time.Unix(int64(cookie.Expires), 0)
			}
			filteredCookies = append(filteredCookies, cookieData)

			// http.Cookieとしても保存
			httpCookie := &http.Cookie{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Domain:   cookie.Domain,
				Path:     cookie.Path,
				HttpOnly: cookie.HTTPOnly,
				Secure:   cookie.Secure,
			}
			if cookie.Expires != 0 {
				httpCookie.Expires = time.Unix(int64(cookie.Expires), 0)
			}
			httpCookies = append(httpCookies, httpCookie)
		}
	}

	// 内部のクッキーストレージを更新
	cm.cookies = httpCookies

	cache := cookieCache{
		Cookies: filteredCookies,
		SavedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	err = os.WriteFile(cm.filePath, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	slog.Info("Cookieを保存しました", "count", len(filteredCookies), "file_path", cm.filePath)
	return nil
}

// LoadCookies はファイルからCookieを読み込んでブラウザに設定する
func (cm *ManagerImpl) LoadCookies(ctx *context.Context) error {
	if !cm.CookieFileExists() {
		return fmt.Errorf("cookie file does not exist: %s", cm.filePath)
	}

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		return fmt.Errorf("failed to read cookies file: %w", err)
	}

	var cache cookieCache
	err = json.Unmarshal(data, &cache)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cookies: %w", err)
	}

	// Cookieの有効期限をチェック
	var validCookies []entry
	now := time.Now()
	for _, cookie := range cache.Cookies {
		if cookie.Expires.IsZero() || cookie.Expires.After(now) {
			validCookies = append(validCookies, cookie)
		}
	}

	if len(validCookies) == 0 {
		return fmt.Errorf("no valid cookies found")
	}

	// http.Cookieとして内部ストレージに保存
	var httpCookies []*http.Cookie
	for _, cookie := range validCookies {
		httpCookie := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			HttpOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		}
		if !cookie.Expires.IsZero() {
			httpCookie.Expires = cookie.Expires
		}
		httpCookies = append(httpCookies, httpCookie)
	}
	cm.cookies = httpCookies

	// ブラウザにCookieを設定 (タイムアウト付き)
	loadCtx, loadCancel := context.WithTimeout(*ctx, CookieOperationTimeout)
	defer loadCancel()

	var actions []chromedp.Action
	for _, cookie := range validCookies {
		var expires *cdp.TimeSinceEpoch
		if !cookie.Expires.IsZero() {
			exp := cdp.TimeSinceEpoch(cookie.Expires)
			expires = &exp
		}

		actions = append(actions, network.SetCookie(cookie.Name, cookie.Value).
			WithDomain(cookie.Domain).
			WithPath(cookie.Path).
			WithExpires(expires).
			WithHTTPOnly(cookie.HTTPOnly).
			WithSecure(cookie.Secure))
	}

	err = chromedp.Run(loadCtx, actions...)
	if err != nil {
		return fmt.Errorf("failed to set cookies in browser: %w", err)
	}

	slog.Info("Cookieを読み込みました", "count", len(validCookies), "file_path", cm.filePath)
	return nil
}

// CookieFileExists はCookieファイルが存在するかどうかをチェックする
func (cm *ManagerImpl) CookieFileExists() bool {
	_, err := os.Stat(cm.filePath)
	return err == nil
}

// DeleteCookieFile はCookieファイルを削除する
func (cm *ManagerImpl) DeleteCookieFile() error {
	if !cm.CookieFileExists() {
		return nil
	}
	return os.Remove(cm.filePath)
}

// GetCookiesForURL は指定されたURLに対して適切なCookieを返す
func (cm *ManagerImpl) GetCookiesForURL(url *url.URL) []*http.Cookie {
	var result []*http.Cookie
	now := time.Now()

	for _, cookie := range cm.cookies {
		// 期限切れチェック
		if !cookie.Expires.IsZero() && cookie.Expires.Before(now) {
			continue
		}

		// ドメインマッチング
		if !cm.cookieMatchesDomain(cookie, url.Host) {
			continue
		}

		// パスマッチング
		if !cm.cookieMatchesPath(cookie, url.Path) {
			continue
		}

		// Secure属性チェック
		if cookie.Secure && url.Scheme != "https" {
			continue
		}

		result = append(result, cookie)
	}

	return result
}

// SetCookies は指定されたURLに対してCookieを設定する
func (cm *ManagerImpl) SetCookies(url *url.URL, cookies []*http.Cookie) error {
	// 既存のクッキーをマップに変換（名前とドメインをキーとして使用）
	existingCookies := make(map[string]*http.Cookie)
	for _, cookie := range cm.cookies {
		key := cookie.Name + "|" + cookie.Domain
		existingCookies[key] = cookie
	}

	// 新しいクッキーを追加または更新
	for _, newCookie := range cookies {
		// ドメインが設定されていない場合はURLのホストを使用
		if newCookie.Domain == "" {
			newCookie.Domain = url.Host
		}
		// パスが設定されていない場合はデフォルトパスを使用
		if newCookie.Path == "" {
			newCookie.Path = "/"
		}

		key := newCookie.Name + "|" + newCookie.Domain
		existingCookies[key] = newCookie
	}

	// マップから配列に戻す
	cm.cookies = make([]*http.Cookie, 0, len(existingCookies))
	for _, cookie := range existingCookies {
		cm.cookies = append(cm.cookies, cookie)
	}

	return nil
}

// cookieMatchesDomain はクッキーがドメインにマッチするかをチェック
func (cm *ManagerImpl) cookieMatchesDomain(cookie *http.Cookie, host string) bool {
	domain := cookie.Domain
	if domain == "" {
		return false
	}

	// ドメインが"."で始まる場合（例：.oreilly.com）
	if strings.HasPrefix(domain, ".") {
		// サブドメインマッチング
		return strings.HasSuffix(host, domain) || host == domain[1:]
	}

	// 完全一致
	return host == domain
}

// cookieMatchesPath はクッキーがパスにマッチするかをチェック
func (cm *ManagerImpl) cookieMatchesPath(cookie *http.Cookie, path string) bool {
	cookiePath := cookie.Path
	if cookiePath == "" {
		cookiePath = "/"
	}

	// パスプレフィックスマッチング
	if !strings.HasPrefix(path, cookiePath) {
		return false
	}

	// 完全一致またはスラッシュで区切られている
	return len(path) == len(cookiePath) || cookiePath[len(cookiePath)-1] == '/' || path[len(cookiePath)] == '/'
}

// isImportantCookie は保存すべき重要なCookieかどうかを判定する
func (cm *ManagerImpl) isImportantCookie(name string) bool {
	// 除外すべきCookieのリスト（一般的な分析・トラッキング系）
	excludedCookies := []string{
		"_ga",                                            // Google Analytics
		"_gid",                                           // Google Analytics
		"_gat",                                           // Google Analytics
		"_gtm",                                           // Google Tag Manager
		"_fbp",                                           // Facebook Pixel
		"_hjid",                                          // Hotjar
		"_hjIncludedInPageviewSample",                    // Hotjar
		"optimizelyEndUserId",                            // Optimizely
		"__utma", "__utmb", "__utmc", "__utmt", "__utmz", // Old Google Analytics
	}

	// 除外対象かチェック
	for _, excluded := range excludedCookies {
		if name == excluded {
			return false
		}
	}

	// O'Reilly関連の全Cookieを保存する
	// 認証、セッション、設定、機能関連のCookieを幅広く含める
	return true
}
