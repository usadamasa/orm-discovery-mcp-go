package browser

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"net/http"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// NewBrowserClient は新しいブラウザクライアントを作成し、ログインを実行します
func NewBrowserClient(userID, password string) (*BrowserClient, error) {
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
	ctx, _ := chromedp.NewContext(allocCtx)

	client := &BrowserClient{
		ctx:    ctx,
		cancel: cancel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}

	// ログインを実行
	if err := client.login(userID, password); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to login: %w", err)
	}

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

	err := chromedp.Run(bc.ctx,
		// ログインページに移動
		chromedp.Navigate("https://www.oreilly.com/member/login/"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("ログインページに移動しました")
			return nil
		}),
		chromedp.WaitVisible(`input[name="email"]`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("メールアドレス入力フィールドが表示されました")
			return nil
		}),

		// 第1段階: メールアドレスを入力してContinueボタンをクリック
		chromedp.SendKeys(`input[name="email"]`, userID, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("メールアドレスを入力しました: %s", userID)
			return nil
		}),

		// Continueボタンをクリック（第1段階）
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("Continueボタンをクリックしようとしています")
			return nil
		}),
		// JavaScriptを使用してボタンをクリック
		chromedp.Evaluate(`
			const button = document.querySelector('button[type="submit"]');
			if (button) {
				button.click();
				console.log('Continueボタンをクリックしました');
			} else {
				console.log('Continueボタンが見つかりません');
			}
		`, nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("JavaScriptでContinueボタンをクリックしました")
			return nil
		}),

		// リダイレクトまたはページ更新を待機
		chromedp.Sleep(5*time.Second), // より長い待機時間
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("リダイレクト後のURL: %s", currentURL)
			}
			return nil
		}),

		// 複数のケースに対応する処理
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err != nil {
				return err
			}

			// ケース1: ACMのIDPにリダイレクトされた場合
			if strings.Contains(currentURL, "idp.acm.org") {
				log.Printf("ACM IDPにリダイレクトされました")

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
					// Sign inボタンをクリック
					chromedp.Click(`button:contains("Sign in"), input[value="Sign in"]`, chromedp.ByQuery),
					chromedp.ActionFunc(func(ctx context.Context) error {
						log.Printf("ACM Sign inボタンをクリックしました")
						return nil
					}),
					// ACMログイン完了を待機
					chromedp.Sleep(5*time.Second),
				)
			}

			// ケース2: 同じO'Reillyページでパスワード入力フィールドが表示された場合
			if strings.Contains(currentURL, "oreilly.com/member/login") {
				log.Printf("O'Reillyページでパスワード入力フィールドを確認します")

				// パスワード入力フィールドが存在するかチェック
				var passwordExists bool
				if err := chromedp.Evaluate(`!!document.querySelector('input[name="password"]')`, &passwordExists).Do(ctx); err == nil && passwordExists {
					log.Printf("O'Reillyページでパスワード入力フィールドが見つかりました")

					return chromedp.Run(ctx,
						// パスワードを入力
						chromedp.SendKeys(`input[name="password"]`, password, chromedp.ByQuery),
						chromedp.ActionFunc(func(ctx context.Context) error {
							log.Printf("O'Reillyページでパスワードを入力しました")
							return nil
						}),
						// Sign Inボタンをクリック
						chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
						chromedp.ActionFunc(func(ctx context.Context) error {
							log.Printf("O'ReillyページでSign Inボタンをクリックしました")
							return nil
						}),
					)
				}
			}

			log.Printf("想定されたログインフローが見つかりませんでした。現在のURL: %s", currentURL)
			return nil // エラーにせず、次のステップに進む
		}),

		// ログイン完了まで待機（ホームページまたは学習プラットフォームページ）
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 最大60秒待機（時間を延長）
			timeout := time.Now().Add(60 * time.Second)
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

		// 学習プラットフォームに移動して確実にログイン状態を確立
		chromedp.Navigate("https://learning.oreilly.com/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second), // より長い待機時間

		// ログイン状態を確認し、必要に応じて再ログイン
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("警告: 学習プラットフォームへの直接アクセスに失敗しました")
				return err
			}
			log.Printf("学習プラットフォームアクセス後のURL: %s", currentURL)

			// ログインページにリダイレクトされている場合の処理
			if strings.Contains(currentURL, "/member/login") || strings.Contains(currentURL, "/login") {
				log.Printf("ログインページにリダイレクトされました。直接学習プラットフォームにアクセスを試行します")

				// 複数のURLパターンを試行
				url := "https://learning.oreilly.com/home/"

				log.Printf("URL試行: %s", url)
				if err := chromedp.Run(ctx,
					chromedp.Navigate(url),
					chromedp.WaitVisible(`body`, chromedp.ByQuery),
					chromedp.Sleep(3*time.Second),
				); err == nil {
					var newURL string
					if err := chromedp.Location(&newURL).Do(ctx); err == nil {
						log.Printf("新しいURL: %s", newURL)
						if !strings.Contains(newURL, "/login") {
							log.Printf("学習プラットフォームへのアクセスに成功しました")
							return nil
						}
					}
				}

				log.Printf("学習プラットフォームへのアクセスに成功しました")
			}
			return nil
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
			return nil
		}),
	)

	if err != nil {
		return fmt.Errorf("ログイン処理でエラーが発生しました: %w", err)
	}

	// Cookieを保存
	bc.cookies = cookies
	log.Printf("ログインが完了し、%d個のCookieを取得しました", len(cookies))

	// 重要なCookieをログ出力（デバッグ用）
	for _, cookie := range cookies {
		if strings.Contains(cookie.Name, "jwt") ||
			strings.Contains(cookie.Name, "session") ||
			strings.Contains(cookie.Name, "auth") {
			log.Printf("重要なCookie取得: %s", cookie.Name)
		}
	}

	return nil
}

// GetCookieString はHTTPリクエスト用のCookie文字列を返します
func (bc *BrowserClient) GetCookieString() string {
	var cookieStrings []string
	for _, cookie := range bc.cookies {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
	}
	return strings.Join(cookieStrings, "; ")
}

// GetJWTToken はJWTトークンを取得します
func (bc *BrowserClient) GetJWTToken() string {
	for _, cookie := range bc.cookies {
		if cookie.Name == "orm-jwt" {
			return cookie.Value
		}
	}
	return ""
}

// GetSessionID はセッションIDを取得します
func (bc *BrowserClient) GetSessionID() string {
	for _, cookie := range bc.cookies {
		if cookie.Name == "groot_sessionid" {
			return cookie.Value
		}
	}
	return ""
}

// GetRefreshToken はリフレッシュトークンを取得します
func (bc *BrowserClient) GetRefreshToken() string {
	for _, cookie := range bc.cookies {
		if cookie.Name == "orm-rt" {
			return cookie.Value
		}
	}
	return ""
}

// RefreshSession はセッションを更新します
func (bc *BrowserClient) RefreshSession() error {
	log.Printf("セッションの更新を開始します")

	var cookies []*http.Cookie

	err := chromedp.Run(bc.ctx,
		// 学習プラットフォームのホームページにアクセス
		chromedp.Navigate("https://learning.oreilly.com/home/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// 更新されたCookieを取得
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
			return nil
		}),
	)

	if err != nil {
		return fmt.Errorf("セッション更新でエラーが発生しました: %w", err)
	}

	// Cookieを更新
	bc.cookies = cookies
	log.Printf("セッションが更新され、%d個のCookieを取得しました", len(cookies))

	return nil
}
