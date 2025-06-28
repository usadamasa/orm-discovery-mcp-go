package cookie

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const cookieFileName = "orm-mcp-go-cookies.json"

// Manager の前方宣言（main パッケージの構造体）
type Manager interface {
	SaveCookies(ctx *context.Context) error
	LoadCookies(ctx *context.Context) error
	CookieFileExists() bool
	DeleteCookieFile() error
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
	tmpDir   string
	filePath string
}

// NewCookieManager は新しいCookieManagerを作成する
func NewCookieManager(tmpDir string) *ManagerImpl {
	return &ManagerImpl{
		tmpDir:   tmpDir,
		filePath: filepath.Join(tmpDir, cookieFileName),
	}
}

// SaveCookies はブラウザのCookieをファイルに保存する
func (cm *ManagerImpl) SaveCookies(ctx *context.Context) error {
	var cookies []*network.Cookie
	err := chromedp.Run(*ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return fmt.Errorf("failed to get cookies from browser: %w", err)
	}

	// 重要なCookieのみをフィルタリング
	var filteredCookies []entry
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
		}
	}

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

	log.Printf("Saved %d cookies to %s", len(filteredCookies), cm.filePath)
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

	// ブラウザにCookieを設定
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

	err = chromedp.Run(*ctx, actions...)
	if err != nil {
		return fmt.Errorf("failed to set cookies in browser: %w", err)
	}

	log.Printf("Loaded %d cookies from %s", len(validCookies), cm.filePath)
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
