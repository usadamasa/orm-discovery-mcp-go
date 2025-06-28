package browser

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const ormHome = "https://learning.oreilly.com/home/"

// NewBrowserClient は新しいブラウザクライアントを作成し、ログインを実行します
func NewBrowserClient(userID, password string, cookieManager cookie.Manager, debug bool, tmpDir string) (*BrowserClient, error) {
	if userID == "" || password == "" {
		return nil, fmt.Errorf("OREILLY_USER_ID and OREILLY_PASSWORD are required")
	}

	// ヘッドレスブラウザのコンテキストを作成
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, _ := chromedp.NewContext(allocCtx)

	// CookieJarを作成
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &BrowserClient{
		ctx:       ctx,
		cancel:    cancel,
		cookieJar: jar,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		tmpDir:    tmpDir,
		debug:     debug,
	}

	// Cookieの復元を試行
	if cookieManager != nil && cookieManager.CookieFileExists() {
		log.Printf("既存のCookieファイルが見つかりました。復元を試行します")
		if err := cookieManager.LoadCookies(&ctx); err != nil {
			log.Printf("Cookie復元に失敗しました: %v", err)
		} else {
			// ブラウザのCookieをHTTPクライアントに同期
			client.syncCookiesFromBrowser()

			// Cookieが有効かどうか検証
			if client.validateAuthentication(ctx) {
				log.Printf("Cookieを使用してログインが完了しました")
				client.cookieManager = cookieManager
				return client, nil
			}
			log.Printf("Cookieが無効でした。通常のログインを実行します")
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
		log.Printf("Cookieの保存に失敗しました: %v", err)
	}

	// ブラウザのCookieをHTTPクライアントに同期
	client.syncCookiesFromBrowser()

	log.Printf("ブラウザクライアントの初期化とログインが完了しました")
	return client, nil
}

// Close はブラウザクライアントをクリーンアップします
func (bc *BrowserClient) Close() {
	if bc.cancel != nil {
		bc.cancel()
	}
}

// login はO'Reillyにログインし、セッションCookieを取得します
func (bc *BrowserClient) login(userID, password string) error {
	log.Printf("O'Reillyへのログインを開始します: %s", userID)

	var cookies []*http.Cookie
	var divText string

	err := chromedp.Run(bc.ctx,
		// ログインページに移動
		chromedp.Navigate("https://www.oreilly.com/member/login/"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("ログインページに移動しました")
			return nil
		}),
		// メールアドレスの入力
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="email"]`, userID, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("メールアドレスを入力しました: %s", userID)
			bc.debugScreenshot(ctx, "orm_filled_email")
			log.Printf("Continueボタンをクリックしようとしています")
			return nil
		}),
		// Continueボタンをクリック
		chromedp.WaitVisible(`.orm-Button-root`, chromedp.ByQuery),
		chromedp.Click(`.orm-Button-root`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// クリック操作
			bc.debugScreenshot(ctx, "orm_clicked_continue")
			log.Printf("Continueボタンをクリックしました")
			return nil
		}),
		// リダイレクトまたはページ更新を待機
		chromedp.WaitVisible(`.sub-title`, chromedp.ByQuery),
		chromedp.Text(`.sub-title`, &divText, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			bc.debugScreenshot(ctx, "acm_after_redirected")
			log.Printf(".sub-title: %s", divText)
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err != nil {
				return err
			}
			if strings.Contains(currentURL, "idp.acm.org") {
				log.Printf("ACM IDPにリダイレクトされました")
			} else {
				log.Fatalf("想定されたログインフローが見つかりませんでした。現在のURL: %s", currentURL)
			}
			return nil
		}),
		// ACM IDPでログイン
		chromedp.ActionFunc(func(ctx context.Context) error {
			// @acm.orgを除いたユーザー名を取得
			username := strings.TrimSuffix(userID, "@acm.org")
			log.Printf("ACMユーザー名: %s", username)

			return chromedp.Run(ctx,
				// ユーザー名フィールドを待機
				chromedp.WaitVisible(`input[placeholder*="username"]`, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("ACMユーザー名フィールドが表示されました")
					return nil
				}),
				// ユーザー名を入力
				chromedp.Clear(`input[placeholder*="username"]`, chromedp.ByQuery),
				chromedp.SendKeys(`input[placeholder*="username"]`, username, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("ACMユーザー名を入力しました: %s", username)
					return nil
				}),
				// パスワードを入力
				chromedp.SendKeys(`input[placeholder*="password"]`, password, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("ACMパスワードを入力しました")
					return nil
				}),
				chromedp.ActionFunc(func(ctx context.Context) error {
					bc.debugScreenshot(ctx, "acm_filled")
					return nil
				}),
				// Sign inボタンをクリック
				chromedp.Click(`.btn`, chromedp.ByQuery),
				chromedp.ActionFunc(func(ctx context.Context) error {
					log.Printf("ACM Sign inボタンをクリックしました")
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
					log.Printf("URL取得エラー: %v", err)
					time.Sleep(2 * time.Second)
					continue
				}

				log.Printf("ログイン処理中のURL: %s", currentURL)

				// ログイン成功の判定
				if strings.Contains(currentURL, "learning.oreilly.com") ||
					strings.Contains(currentURL, "oreilly.com/home") ||
					strings.Contains(currentURL, "oreilly.com/member") {
					log.Printf("ログイン成功を確認しました")
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
			// Cookieを保存
			bc.cookies = cookies
			log.Printf("%d個のCookieを取得しました", len(cookies))
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
		log.Printf("認証検証中にエラーが発生しました: %v", err)
		return false
	}
	log.Printf("ログイン処理中のURL: %s", currentURL)

	// ログインページにリダイレクトされていないかチェック
	if currentURL != ormHome {
		log.Printf("認証が無効です。現在のURL: %s", currentURL)
		return false
	}

	log.Printf("認証検証成功: %s", pageTitle)
	return true
}

// syncCookiesFromBrowser はブラウザのCookieをHTTPクライアントのCookieJarに同期します
func (bc *BrowserClient) syncCookiesFromBrowser() {
	var cookies []*network.Cookie
	err := chromedp.Run(bc.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))

	if err != nil {
		log.Printf("ブラウザからのCookie取得に失敗しました: %v", err)
		return
	}

	// ブラウザのCookieをHTTPクライアントのCookieJarに追加
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

	for _, u := range urls {
		bc.cookieJar.SetCookies(u, httpCookies)
	}

	// デバッグログ
	if bc.debug {
		log.Printf("HTTPクライアントに%d個のCookieを同期しました", len(httpCookies))
		for _, c := range httpCookies {
			value := c.Value
			if len(value) > 20 {
				value = value[:20] + "..."
			}
			log.Printf("Cookie同期: %s=%s (Domain: %s, Path: %s)", c.Name, value, c.Domain, c.Path)
		}
	}

	// BrowserClientのcookiesフィールドも更新
	bc.cookies = httpCookies
}

// UpdateCookiesFromBrowser はAPI呼び出し前にCookieを最新状態に更新します
func (bc *BrowserClient) UpdateCookiesFromBrowser() {
	if bc.debug {
		log.Printf("API呼び出し前にCookieを更新中...")
	}
	bc.syncCookiesFromBrowser()
}
