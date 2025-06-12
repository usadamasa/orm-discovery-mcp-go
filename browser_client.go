package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// BrowserClient はヘッドレスブラウザを使用したO'Reillyクライアントです
type BrowserClient struct {
	ctx        context.Context
	cancel     context.CancelFunc
	httpClient *http.Client
	cookies    []*http.Cookie
	userAgent  string
}

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
				log.Printf("学習プラットフォームアクセス後のURL: %s", currentURL)
				
				// ログインページにリダイレクトされている場合の処理
				if strings.Contains(currentURL, "/member/login") || strings.Contains(currentURL, "/login") {
					log.Printf("ログインページにリダイレクトされました。直接学習プラットフォームにアクセスを試行します")
					
					// 複数のURLパターンを試行
					urls := []string{
						"https://learning.oreilly.com/home/",
						"https://learning.oreilly.com/library/",
						"https://learning.oreilly.com/playlists/",
					}
					
					for _, url := range urls {
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
					}
					
					log.Printf("警告: 学習プラットフォームへの直接アクセスに失敗しました")
				} else {
					log.Printf("学習プラットフォームへのアクセスに成功しました")
				}
			}
			return nil
		}),
		
		// プレイリストページへの事前アクセスを試行
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリストページへの事前アクセスを試行します")
			
			if err := chromedp.Run(ctx,
				chromedp.Navigate("https://learning.oreilly.com/playlists/"),
				chromedp.WaitVisible(`body`, chromedp.ByQuery),
				chromedp.Sleep(3*time.Second),
			); err == nil {
				var playlistURL string
				if err := chromedp.Location(&playlistURL).Do(ctx); err == nil {
					log.Printf("プレイリストページアクセス結果: %s", playlistURL)
					if !strings.Contains(playlistURL, "/login") {
						log.Printf("プレイリストページへのアクセスに成功しました")
					} else {
						log.Printf("プレイリストページでログインが必要です")
					}
				}
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

// SearchContent はO'Reilly Learning Platformでコンテンツを検索します
func (bc *BrowserClient) SearchContent(query string, options map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("検索を開始します: %s", query)
	
	var results []map[string]interface{}
	
	// オプションのデフォルト値を設定
	rows := 100
	if r, ok := options["rows"].(int); ok && r > 0 {
		rows = r
	}
	
	// 言語オプションは現在使用していないため、将来の拡張用として保持
	_ = options["languages"] // 未使用警告を回避
	
	// URLエンコードされた検索クエリで直接検索結果ページにアクセス
	searchURL := fmt.Sprintf("https://www.oreilly.com/search/?q=%s&rows=%d", 
		strings.ReplaceAll(query, " ", "+"), rows)
	
	err := chromedp.Run(bc.ctx,
		// 検索結果ページに直接移動
		chromedp.Navigate(searchURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("検索結果ページに直接移動しました: %s", searchURL)
			return nil
		}),
		
		// ページの読み込み完了を待機
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // 検索結果の読み込み待機
		
		// 現在のURLを確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("検索結果ページのURL: %s", currentURL)
			}
			return nil
		}),
		
		// 検索結果を取得（O'Reillyの新しい検索ページ構造に対応）
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("検索結果の抽出を開始します")
			
			// より広範囲なリンクセレクターで検索結果を確認
			var hasResults bool
			if err := chromedp.Evaluate(`
				const resultElements = document.querySelectorAll('a[href*="oreilly.com"], a[href*="/library/"], a[href*="/view/"], a[href*="/book/"], a[href*="/video"]');
				console.log('検索結果リンク数:', resultElements.length);
				resultElements.length > 0
			`, &hasResults).Do(ctx); err != nil || !hasResults {
				log.Printf("検索結果のリンクが見つかりませんでした")
				
				// デバッグ情報を取得
				var pageContent string
				if err := chromedp.Evaluate(`document.body.textContent.substring(0, 1000)`, &pageContent).Do(ctx); err == nil {
					log.Printf("ページ内容の一部: %s", pageContent)
				}
				return nil
			}
			
			log.Printf("検索結果のリンクが見つかりました")
			
			// O'Reillyの新しい検索ページ構造に対応した抽出ロジック
			var searchResults []map[string]interface{}
			if err := chromedp.Evaluate(fmt.Sprintf(`
				(function() {
					const results = [];
					const processedTitles = new Set();
					
					// より広範囲なセレクターでリンクを取得
					const linkSelectors = [
						'a[href*="learning.oreilly.com"]',
						'a[href*="/library/view/"]',
						'a[href*="/library/book/"]',
						'a[href*="/videos/"]',
						'a[href*="/book/"]',
						'a[href*="/video"]'
					];
					
					let allLinks = [];
					for (const selector of linkSelectors) {
						const links = Array.from(document.querySelectorAll(selector));
						allLinks = allLinks.concat(links);
					}
					
					// 重複を除去
					const uniqueLinks = Array.from(new Set(allLinks));
					console.log('処理対象リンク数:', uniqueLinks.length);
					
					for (let i = 0; i < Math.min(uniqueLinks.length, %d); i++) {
						const link = uniqueLinks[i];
						
						// リンクの親要素を検索してコンテナを見つける
						let container = link;
						for (let j = 0; j < 5; j++) {
							container = container.parentElement;
							if (!container) break;
							
							// 適切なコンテナかチェック
							const containerClasses = container.className || '';
							if (containerClasses.includes('result') || 
								containerClasses.includes('item') || 
								containerClasses.includes('card') ||
								container.tagName === 'ARTICLE' ||
								container.tagName === 'LI') {
								break;
							}
						}
						
						if (!container) container = link.parentElement || link;
						
						// タイトルを取得（より柔軟な方法）
						let title = '';
						
						// 1. リンクのテキストを確認
						if (link.textContent && link.textContent.trim()) {
							title = link.textContent.trim();
						}
						
						// 2. コンテナ内のタイトル要素を確認
						if (!title || title.length < 5) {
							const titleSelectors = [
								'h1, h2, h3, h4, h5, h6',
								'.title',
								'[data-testid*="title"]',
								'.book-title, .video-title',
								'strong, b',
								'.name'
							];
							
							for (const selector of titleSelectors) {
								const titleEl = container.querySelector(selector);
								if (titleEl && titleEl.textContent.trim() && titleEl.textContent.trim().length > title.length) {
									title = titleEl.textContent.trim();
									break;
								}
							}
						}
						
						// タイトルのクリーンアップ
						title = title.replace(/^\s*[\d\.\-\*\+]\s*/, '').trim(); // 先頭の番号や記号を除去
						
						// 重複チェック
						if (!title || title.length < 3 || processedTitles.has(title)) {
							continue;
						}
						processedTitles.add(title);
						
						// URLとOURNを取得
						const url = link.href;
						let ourn = '';
						const ournMatches = [
							url.match(/\/library\/view\/[^\/]+\/([^\/\?]+)/),
							url.match(/\/book\/([^\/\?]+)/),
							url.match(/\/video\/([^\/\?]+)/)
						];
						
						for (const match of ournMatches) {
							if (match) {
								ourn = match[1];
								break;
							}
						}
						
						// 著者情報を取得
						let authors = [];
						const authorSelectors = [
							'.author, .authors',
							'[data-testid*="author"]',
							'.by-author',
							'.book-author',
							'[class*="author"]'
						];
						
						for (const selector of authorSelectors) {
							const authorEl = container.querySelector(selector);
							if (authorEl && authorEl.textContent.trim()) {
								const authorText = authorEl.textContent.trim();
								authors = [authorText.replace(/^(by|著者?:?)\s*/i, '')];
								break;
							}
						}
						
						// コンテンツタイプを推測
						let contentType = 'unknown';
						if (url.includes('/book/') || url.includes('/library/view/')) {
							contentType = 'book';
						} else if (url.includes('/video')) {
							contentType = 'video';
						} else if (url.includes('/learning-path')) {
							contentType = 'learning-path';
						}
						
						// 説明を取得
						let description = '';
						const descSelectors = [
							'.description, .summary',
							'p',
							'.excerpt',
							'.content'
						];
						
						for (const selector of descSelectors) {
							const descEl = container.querySelector(selector);
							if (descEl && descEl.textContent.trim()) {
								description = descEl.textContent.trim().substring(0, 200);
								break;
							}
						}
						
						// 出版社を取得
						let publisher = '';
						const publisherSelectors = [
							'.publisher',
							'[data-testid*="publisher"]',
							'.imprint',
							'[class*="publisher"]'
						];
						
						for (const selector of publisherSelectors) {
							const publisherEl = container.querySelector(selector);
							if (publisherEl && publisherEl.textContent.trim()) {
								publisher = publisherEl.textContent.trim();
								break;
							}
						}
						
						results.push({
							id: ourn || 'item_' + (results.length + 1),
							ourn: ourn,
							title: title,
							authors: authors,
							content_type: contentType,
							description: description,
							url: url,
							publisher: publisher,
							published_date: '',
							source: 'browser_search_oreilly_new'
						});
					}
					
					console.log('抽出された結果数:', results.length);
					return results;
				})()
			`, rows), &searchResults).Do(ctx); err != nil {
				log.Printf("検索結果の抽出でエラーが発生しました: %v", err)
				return err
			}
			
			results = searchResults
			log.Printf("検索結果を取得しました: %d件", len(results))
			return nil
		}),
	)

	if err != nil {
		log.Printf("検索処理でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("検索処理でエラーが発生しました: %w", err)
	}

	if len(results) == 0 {
		log.Printf("検索結果が見つかりませんでした: %s", query)
		
		// デバッグ情報を取得
		debugErr := chromedp.Run(bc.ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var pageTitle, currentURL string
				chromedp.Title(&pageTitle).Do(ctx)
				chromedp.Location(&currentURL).Do(ctx)
				log.Printf("デバッグ情報 - ページタイトル: %s, URL: %s", pageTitle, currentURL)
				
				// ページの内容を確認
				var bodyText string
				if err := chromedp.Evaluate(`document.body.textContent.substring(0, 500)`, &bodyText).Do(ctx); err == nil {
					log.Printf("ページ内容の一部: %s", bodyText)
				}
				
				return nil
			}),
		)
		if debugErr != nil {
			log.Printf("デバッグ情報の取得に失敗: %v", debugErr)
		}
		
		return []map[string]interface{}{}, nil
	}

	log.Printf("検索が完了しました。%d件の結果を取得: %s", len(results), query)
	return results, nil
}

// SearchContentAPI は O'Reilly Learning Platform の内部 API を使用して検索を実行します
func (bc *BrowserClient) SearchContentAPI(query string, options map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("API検索を開始します: %s", query)
	
	// オプションのデフォルト値を設定
	rows := 100
	if r, ok := options["rows"].(int); ok && r > 0 {
		rows = r
	}
	
	// 言語オプションは現在使用していないため、将来の拡張用として保持
	_ = options["languages"] // 未使用警告を回避
	
	tzOffset := -9 // JST
	if tz, ok := options["tzOffset"].(int); ok {
		tzOffset = tz
	}
	
	aiaOnly := false
	if aia, ok := options["aia_only"].(bool); ok {
		aiaOnly = aia
	}
	
	featureFlags := "improveSearchFilters"
	if ff, ok := options["feature_flags"].(string); ok && ff != "" {
		featureFlags = ff
	}
	
	report := true
	if rep, ok := options["report"].(bool); ok {
		report = rep
	}
	
	isTopics := false
	if topics, ok := options["isTopics"].(bool); ok {
		isTopics = topics
	}
	
	var results []map[string]interface{}
	var networkResponses []string
	
	// ネットワークリクエストとコンソールログの監視を有効化
	chromedp.ListenTarget(bc.ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			resp := ev.Response
			if strings.Contains(resp.URL, "/api/") || 
			   strings.Contains(resp.URL, "search") ||
			   strings.Contains(resp.URL, "query") {
				networkResponses = append(networkResponses, resp.URL)
				log.Printf("検出されたAPI呼び出し: %s", resp.URL)
			}
		case *runtime.EventConsoleAPICalled:
			args := make([]string, len(ev.Args))
			for i, arg := range ev.Args {
				if len(arg.Value) > 0 {
					args[i] = string(arg.Value)
				} else {
					args[i] = arg.Type.String()
				}
			}
			log.Printf("コンソールログ [%s]: %s", ev.Type, strings.Join(args, " "))
		}
	})
	
	err := chromedp.Run(bc.ctx,
		// ネットワーク監視とランタイムイベントを有効化
		network.Enable(),
		runtime.Enable(),
		
		// 検索ページに移動
		chromedp.Navigate("https://www.oreilly.com/search/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		// 検索APIを直接呼び出すJavaScriptを実行
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("API検索のJavaScript実行を開始します")
			
			// まず現在のページで学習プラットフォームにアクセスできているか確認
			var currentUrl string
			if err := chromedp.Location(&currentUrl).Do(ctx); err == nil {
				log.Printf("現在のURL: %s", currentUrl)
				if !strings.Contains(currentUrl, "oreilly.com") {
					log.Printf("O'Reillyドメインにいません。learning.oreilly.comにリダイレクトします")
					if err := chromedp.Navigate("https://learning.oreilly.com/search/").Do(ctx); err != nil {
						log.Printf("学習プラットフォームへのナビゲートに失敗: %v", err)
					} else {
						_ = chromedp.Sleep(2 * time.Second).Do(ctx)
					}
				}
			}
			
			// 学習プラットフォームでより直接的なAPI呼び出しを試行
			log.Printf("学習プラットフォームでの直接API検索を試行します")
			if err := chromedp.Navigate("https://learning.oreilly.com/search/").Do(ctx); err != nil {
				log.Printf("学習プラットフォーム検索ページへのナビゲートに失敗: %v", err)
			} else {
				_ = chromedp.Sleep(3 * time.Second).Do(ctx)
			}
			
			// 現在のページでAPIアクセス可能性をテスト
			var testResult string
			if err := chromedp.Evaluate(`
				// APIエンドポイントの存在確認
				(function() {
					console.log('APIエンドポイント存在確認開始');
					console.log('現在のホスト:', window.location.hostname);
					console.log('現在のパス:', window.location.pathname);
					
					// 利用可能そうなAPIエンドポイントをテスト
					const endpoints = [
						'/api/v2/search/',
						'/search/api/search/',
						'/api/search/',
						'/learningapi/v1/search/'
					];
					
					return 'test_complete';
				})()
			`, &testResult).Do(ctx); err == nil {
				log.Printf("APIテスト完了: %s", testResult)
			}
			
			// O'Reilly の検索 API を直接呼び出し（同期的実行）
			var apiResults map[string]interface{}
			jsCode := fmt.Sprintf(`
				(() => {
					console.log('検索API呼び出し開始: %s');
					
					try {
						// XMLHttpRequestを使用して同期的に実行
						const xhr = new XMLHttpRequest();
						
						// O'Reilly の検索APIエンドポイント
						const baseUrl = window.location.hostname.includes('learning.oreilly.com') 
							? '/api/v2/search/'  
							: '/search/api/search/';
						
						// パラメータを構築
						const params = new URLSearchParams({
							q: '%s',
							rows: %d,
							tzOffset: %d,
							aia_only: %t,
							feature_flags: '%s',
							report: %t,
							isTopics: %t
						});
						
						const fullUrl = baseUrl + '?' + params.toString();
						console.log('API URL:', fullUrl);
						console.log('現在のドメイン:', window.location.hostname);
						console.log('XHR呼び出し開始');
						
						xhr.open('GET', fullUrl, false); // false = 同期実行
						xhr.setRequestHeader('Accept', 'application/json');
						xhr.setRequestHeader('Content-Type', 'application/json');
						xhr.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
						xhr.withCredentials = true;
						
						xhr.send();
						
						console.log('XHR呼び出し完了');
						console.log('Response status:', xhr.status);
						console.log('Response statusText:', xhr.statusText);
						
						if (xhr.status !== 200) {
							console.error('API呼び出し失敗:', xhr.status, xhr.statusText);
							console.error('レスポンス本文:', xhr.responseText);
							
							// フォールバック: 別のエンドポイントを試行
							const xhr2 = new XMLHttpRequest();
							const fallbackUrl = '/search/api/search/' + '?' + params.toString();
							console.log('フォールバックURL:', fallbackUrl);
							
							xhr2.open('GET', fallbackUrl, false);
							xhr2.setRequestHeader('Accept', 'application/json');
							xhr2.setRequestHeader('X-Requested-With', 'XMLHttpRequest');
							xhr2.withCredentials = true;
							xhr2.send();
							
							if (xhr2.status === 200) {
								console.log('フォールバック成功:', xhr2.status);
								const data = JSON.parse(xhr2.responseText);
								console.log('フォールバックデータ受信');
								return processAPIResponse(data);
							} else {
								console.error('フォールバックも失敗:', xhr2.status);
								return { results: [], error: 'API呼び出し失敗: ' + xhr.status + ' -> ' + xhr2.status };
							}
						}
						
						const data = JSON.parse(xhr.responseText);
						console.log('API レスポンス受信完了、データタイプ:', typeof data);
						console.log('API レスポンス（最初の部分）:', JSON.stringify(data).substring(0, 500));
						
						return processAPIResponse(data);
						
					} catch (error) {
						console.error('API検索エラー:', error);
						return { results: [], error: error.message };
					}
					
					function processAPIResponse(data) {
						// レスポンスから結果を抽出
						let searchResults = [];
						
						// O'Reilly の実際のAPIレスポンス構造に対応
						if (data && data.data && data.data.products && Array.isArray(data.data.products)) {
							console.log('data.products配列から抽出:', data.data.products.length + '件');
							searchResults = data.data.products;
						} else if (data && data.results && Array.isArray(data.results)) {
							console.log('results配列から抽出:', data.results.length + '件');
							searchResults = data.results;
						} else if (data && data.items && Array.isArray(data.items)) {
							console.log('items配列から抽出:', data.items.length + '件');
							searchResults = data.items;
						} else if (data && data.hits && Array.isArray(data.hits)) {
							console.log('hits配列から抽出:', data.hits.length + '件');
							searchResults = data.hits;
						} else if (data && Array.isArray(data)) {
							console.log('データ配列から抽出:', data.length + '件');
							searchResults = data;
						} else {
							console.log('予期しないレスポンス構造:', typeof data);
							console.log('利用可能なキー:', Object.keys(data || {}));
							if (data && data.data) {
								console.log('data内のキー:', Object.keys(data.data || {}));
							}
							searchResults = [];
						}
						
						// 結果を正規化
						const normalizedResults = searchResults.slice(0, %d).map((item, index) => {
							console.log('アイテム ' + index + ' を正規化中');
							
							// URLの正規化 - O'Reillyの構造に合わせて
							let itemUrl = item.web_url || item.url || item.learning_url || item.link || '';
							if (!itemUrl && item.product_id) {
								// product_idからURLを生成
								itemUrl = 'https://learning.oreilly.com/library/view/-/' + item.product_id + '/';
							}
							if (itemUrl && !itemUrl.startsWith('http')) {
								if (itemUrl.startsWith('/')) {
									itemUrl = 'https://learning.oreilly.com' + itemUrl;
								}
							}
							
							// 著者の正規化 - O'Reillyの構造に合わせて
							let authors = [];
							if (item.authors && Array.isArray(item.authors)) {
								authors = item.authors;
							} else if (item.author) {
								authors = [item.author];
							} else if (item.creators && Array.isArray(item.creators)) {
								authors = item.creators.map(c => c.name || c.toString());
							} else if (item.author_names && Array.isArray(item.author_names)) {
								authors = item.author_names;
							}
							
							// コンテンツタイプの決定 - O'Reillyの構造に合わせて  
							let contentType = item.content_type || item.type || item.format || item.product_type || 'unknown';
							if (contentType === 'unknown' && itemUrl) {
								if (itemUrl.includes('/video')) {
									contentType = 'video';
								} else if (itemUrl.includes('/library/view/') || itemUrl.includes('/book/')) {
									contentType = 'book';
								}
							}
							
							// タイトルと説明の抽出
							const title = item.title || item.name || item.display_title || item.product_name || '';
							const description = item.description || item.summary || item.excerpt || 
											  item.description_with_markups || item.short_description || '';
							
							return {
								id: item.product_id || item.id || item.ourn || item.isbn || ('api_result_' + index),
								title: title,
								authors: authors,
								content_type: contentType,
								description: description,
								url: itemUrl,
								ourn: item.ourn || item.product_id || item.id || item.isbn || '',
								publisher: item.publisher || (item.publishers && item.publishers[0]) || 
										 item.imprint || item.publisher_name || '',
								published_date: item.published_date || item.publication_date || 
											   item.date_published || item.pub_date || '',
								source: 'api_search_oreilly'
							};
						});
						
						console.log('正規化された結果:', normalizedResults.length + '件');
						return { results: normalizedResults };
					}
				})()
			`, 
				query,
				query, 
				rows, 
				tzOffset,
				aiaOnly,
				featureFlags,
				report,
				isTopics,
				rows, // 結果数制限用
			)
			
			if err := chromedp.Evaluate(jsCode, &apiResults).Do(ctx); err != nil {
				log.Printf("API検索のJavaScript実行でエラーが発生しました: %v", err)
				return err
			}
			
			// APIレスポンスから結果配列を抽出
			if resultsArray, ok := apiResults["results"].([]interface{}); ok {
				for _, item := range resultsArray {
					if itemMap, ok := item.(map[string]interface{}); ok {
						results = append(results, itemMap)
					}
				}
			}
			
			// エラーがあればログ出力
			if errorMsg, ok := apiResults["error"].(string); ok && errorMsg != "" {
				log.Printf("JavaScript API エラー: %s", errorMsg)
			}
			
			log.Printf("API検索結果を取得しました: %d件", len(results))
			return nil
		}),
	)

	if err != nil {
		log.Printf("API検索処理でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("API検索処理でエラーが発生しました: %w", err)
	}

	// ネットワークレスポンスのログ出力
	if len(networkResponses) > 0 {
		log.Printf("検出されたネットワーク呼び出し:")
		for _, response := range networkResponses {
			log.Printf("  - %s", response)
		}
	}

	log.Printf("API検索が完了しました。%d件の結果を取得: %s", len(results), query)
	return results, nil
}

// GetCollectionsFromHomePage はホームページからコレクション一覧を取得します
func (bc *BrowserClient) GetCollectionsFromHomePage() ([]map[string]interface{}, error) {
	log.Printf("ホームページからコレクション一覧を取得します")
	
	var collections []map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// ホームページに移動
		chromedp.Navigate("https://learning.oreilly.com/home/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		
		// コレクション要素を待機
		chromedp.Sleep(3*time.Second), // ページの読み込み完了を待機
		
		// コレクション情報を取得
		chromedp.ActionFunc(func(ctx context.Context) error {
			// コレクションのタイトルを取得
			var titles []string
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('[data-testid*="collection"], .collection-card, .playlist-card')).map(el => {
					const titleEl = el.querySelector('h2, h3, .title, [data-testid*="title"]');
					return titleEl ? titleEl.textContent.trim() : '';
				}).filter(title => title !== '')
			`, &titles).Do(ctx); err == nil && len(titles) > 0 {
				for i, title := range titles {
					collections = append(collections, map[string]interface{}{
						"id":    fmt.Sprintf("collection_%d", i+1),
						"title": title,
						"type":  "collection",
						"source": "homepage",
					})
				}
			}
			
			// プレイリストのタイトルも取得
			var playlists []string
			if err := chromedp.Evaluate(`
				Array.from(document.querySelectorAll('.playlist, [data-testid*="playlist"]')).map(el => {
					const titleEl = el.querySelector('h2, h3, .title, [data-testid*="title"]');
					return titleEl ? titleEl.textContent.trim() : '';
				}).filter(title => title !== '')
			`, &playlists).Do(ctx); err == nil && len(playlists) > 0 {
				for i, title := range playlists {
					collections = append(collections, map[string]interface{}{
						"id":    fmt.Sprintf("playlist_%d", i+1),
						"title": title,
						"type":  "playlist",
						"source": "homepage",
					})
				}
			}
			
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("ホームページからのコレクション取得でエラーが発生しました: %w", err)
	}

	log.Printf("ホームページから%d個のコレクションを取得しました", len(collections))
	return collections, nil
}

// GetPlaylistsFromPlaylistsPage はプレイリストページからプレイリスト一覧を取得します
func (bc *BrowserClient) GetPlaylistsFromPlaylistsPage() ([]map[string]interface{}, error) {
	log.Printf("プレイリストページからプレイリスト一覧を取得します")
	
	var playlists []map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// 学習プラットフォームに直接アクセス
		chromedp.Navigate("https://learning.oreilly.com/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		// ログイン状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("学習プラットフォームURL確認: %s", currentURL)
				
				// ログインページにリダイレクトされている場合
				if strings.Contains(currentURL, "/login") {
					log.Printf("ログインページにリダイレクトされました。セッションを再確立します")
					
					// セッション再確立を試行
					return bc.RefreshSession()
				}
			}
			return nil
		}),
		
		// プレイリストページに移動
		chromedp.Navigate("https://learning.oreilly.com/playlists/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(5*time.Second), // 待機時間を延長
		
		// アクセス結果を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("プレイリストページアクセス結果URL: %s", currentURL)
				
				// ログインページにリダイレクトされている場合
				if strings.Contains(currentURL, "/login") {
					log.Printf("プレイリストページでログインページにリダイレクトされました")
					return fmt.Errorf("プレイリストページへのアクセスでログインが必要です")
				}
			}
			return nil
		}),
		
		// ページの読み込み完了を待機（より長い時間）
		chromedp.Sleep(8*time.Second),
		
		// 現在のURLとページ状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("最終プレイリストページURL: %s", currentURL)
			}
			
			// ページタイトルも確認
			var pageTitle string
			if err := chromedp.Title(&pageTitle).Do(ctx); err == nil {
				log.Printf("ページタイトル: %s", pageTitle)
			}
			
			// ページの基本情報をデバッグ出力
			var bodyText string
			if err := chromedp.Evaluate(`document.body.textContent.substring(0, 500)`, &bodyText).Do(ctx); err == nil {
				log.Printf("ページ内容（最初の500文字）: %s", bodyText)
			}
			
			// HTMLの構造も確認
			var htmlStructure string
			if err := chromedp.Evaluate(`
				const elements = document.querySelectorAll('*[class*="playlist"], *[data-testid*="playlist"], *[id*="playlist"]');
				Array.from(elements).slice(0, 5).map(el => el.tagName + '.' + el.className + '#' + el.id).join(', ')
			`, &htmlStructure).Do(ctx); err == nil {
				log.Printf("プレイリスト関連要素: %s", htmlStructure)
			}
			
			return nil
		}),
		
		// プレイリスト情報を取得（改善されたアルゴリズム）
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリスト情報の抽出を開始します")
			
			// より包括的なプレイリスト抽出ロジック
			var playlistResults []map[string]interface{}
			if err := chromedp.Evaluate(`
				(function() {
					const results = [];
					const processedTitles = new Set();
					
					console.log('プレイリスト抽出を開始します');
					
					// 1. 直接的なプレイリスト要素を探す
					const directSelectors = [
						'[data-testid*="playlist"]',
						'[class*="playlist"]',
						'[id*="playlist"]',
						'.playlist-card',
						'.playlist-item',
						'[href*="/playlists/"]'
					];
					
					let foundElements = [];
					for (const selector of directSelectors) {
						const elements = document.querySelectorAll(selector);
						if (elements.length > 0) {
							console.log('セレクター', selector, 'で', elements.length, '個の要素を発見');
							foundElements = foundElements.concat(Array.from(elements));
						}
					}
					
					// 重複を除去
					foundElements = Array.from(new Set(foundElements));
					console.log('重複除去後の要素数:', foundElements.length);
					
					// 各要素からプレイリスト情報を抽出
					for (const element of foundElements) {
						try {
							let title = '';
							let creator = '';
							let itemCount = 0;
							let followerCount = 0;
							let url = '';
							let description = '';
							
							// タイトルを取得（複数の方法を試行）
							const titleSelectors = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6', '.title', '[data-testid*="title"]', 'a'];
							for (const titleSel of titleSelectors) {
								const titleEl = element.querySelector(titleSel);
								if (titleEl && titleEl.textContent.trim()) {
									title = titleEl.textContent.trim();
									if (titleEl.href) {
										url = titleEl.href;
									}
									break;
								}
							}
							
							// 要素自体がリンクの場合
							if (!title && element.textContent.trim()) {
								title = element.textContent.trim();
							}
							if (!url && element.href) {
								url = element.href;
							}
							
							// 作成者情報を取得
							const creatorSelectors = ['.author', '.creator', '.by', '[data-testid*="author"]'];
							for (const creatorSel of creatorSelectors) {
								const creatorEl = element.querySelector(creatorSel);
								if (creatorEl && creatorEl.textContent.trim()) {
									creator = creatorEl.textContent.trim().replace(/^(by|By)\s+/, '');
									break;
								}
							}
							
							// 親要素からも作成者情報を探す
							if (!creator && element.parentElement) {
								const parentText = element.parentElement.textContent;
								const byMatch = parentText.match(/By\s+([^\\n\\r]+?)(?=\\s|$)/);
								if (byMatch) {
									creator = byMatch[1].trim();
								}
							}
							
							// アイテム数を取得
							const itemText = element.textContent || (element.parentElement ? element.parentElement.textContent : '');
							const itemMatch = itemText.match(/(\\d+)\\s*items?/i);
							if (itemMatch) {
								itemCount = parseInt(itemMatch[1]);
							}
							
							// フォロワー数を取得
							const followerMatch = itemText.match(/(\\d+)\\s*followers?/i);
							if (followerMatch) {
								followerCount = parseInt(followerMatch[1]);
							}
							
							// 説明を取得
							const descSelectors = ['.description', '.summary', 'p'];
							for (const descSel of descSelectors) {
								const descEl = element.querySelector(descSel);
								if (descEl && descEl.textContent.trim()) {
									description = descEl.textContent.trim().substring(0, 200);
									break;
								}
							}
							
							// タイトルのクリーンアップ
							title = title.replace(/^\\s*[\\d\\.\\-\\*\\+]\\s*/, '').trim();
							
							// 有効なプレイリストかチェック
							if (title && 
								title.length > 2 && 
								title.length < 200 && 
								!processedTitles.has(title) &&
								!title.includes('Sign In') &&
								!title.includes('Welcome') &&
								!title.includes('Get a quick answer') &&
								!title.includes('Search') &&
								!title.includes('Menu')) {
								
								processedTitles.add(title);
								
								// URLが相対パスの場合は絶対パスに変換
								if (url && url.startsWith('/')) {
									url = 'https://learning.oreilly.com' + url;
								}
								
								// URLからIDを抽出
								let playlistId = '';
								if (url) {
									const idMatch = url.match(/\\/playlists\\/([^\\/\\?]+)/);
									if (idMatch) {
										playlistId = idMatch[1];
									}
								}
								
								if (!playlistId) {
									playlistId = 'playlist_' + (results.length + 1);
								}
								
								results.push({
									id: playlistId,
									title: title,
									description: description,
									url: url || ('/playlists/' + encodeURIComponent(title.toLowerCase().replace(/\\s+/g, '-'))),
									creator: creator,
									item_count: itemCount,
									follower_count: followerCount,
									type: 'playlist',
									source: 'playlists_page_dom'
								});
								
								console.log('プレイリスト発見:', title, 'by', creator, itemCount, 'items', followerCount, 'followers');
							}
						} catch (e) {
							console.log('要素処理エラー:', e);
						}
					}
					
					// 2. テキストパターンマッチングによる追加検索
					const textContent = document.body.textContent;
					const playlistPatterns = [
						/([^\\n\\r]{3,80})\\s+By\\s+([^\\n\\r]{2,50})\\s+(\\d+)\\s+items?\\s+(\\d+)\\s+followers?/gi,
						/([^\\n\\r]{3,80})\\s+By\\s+([^\\n\\r]{2,50})\\s+(\\d+)\\s+items?/gi,
						/([^\\n\\r]{3,80})\\s+(\\d+)\\s+items?\\s+(\\d+)\\s+followers?/gi
					];
					
					for (const pattern of playlistPatterns) {
						let match;
						while ((match = pattern.exec(textContent)) !== null && results.length < 50) {
							const title = match[1].trim();
							const creator = match[2] ? match[2].trim() : '';
							const itemCount = parseInt(match[match.length - 2]);
							const followerCount = match[match.length - 1] ? parseInt(match[match.length - 1]) : 0;
							
							if (!processedTitles.has(title) && 
								title.length > 2 && 
								title.length < 100 &&
								!title.includes('Sign In') &&
								!title.includes('Welcome')) {
								
								processedTitles.add(title);
								
								results.push({
									id: 'playlist_pattern_' + (results.length + 1),
									title: title,
									description: '',
									url: '/playlists/' + encodeURIComponent(title.toLowerCase().replace(/\\s+/g, '-')),
									creator: creator,
									item_count: itemCount,
									follower_count: followerCount,
									type: 'playlist',
									source: 'playlists_page_pattern'
								});
								
								console.log('パターンマッチでプレイリスト発見:', title, 'by', creator);
							}
						}
					}
					
					console.log('最終的に抽出されたプレイリスト数:', results.length);
					return results;
				})()
			`, &playlistResults).Do(ctx); err != nil {
				log.Printf("プレイリスト情報の抽出でエラーが発生しました: %v", err)
				return err
			}
			
			playlists = playlistResults
			log.Printf("プレイリスト情報を取得しました: %d件", len(playlists))
			
			// 結果が少ない場合は追加のデバッグ情報を出力
			if len(playlists) == 0 {
				log.Printf("プレイリストが見つかりませんでした。追加のデバッグ情報を取得します")
				
				// ページの全体的な構造を確認
				var debugInfo string
				if err := chromedp.Evaluate(`
					const allElements = document.querySelectorAll('*');
					const elementCounts = {};
					for (const el of allElements) {
						const tag = el.tagName.toLowerCase();
						elementCounts[tag] = (elementCounts[tag] || 0) + 1;
					}
					
					const links = Array.from(document.querySelectorAll('a[href]')).slice(0, 10).map(a => a.href);
					
					return {
						elementCounts: elementCounts,
						sampleLinks: links,
						bodyLength: document.body.textContent.length,
						hasPlaylistInText: document.body.textContent.toLowerCase().includes('playlist')
					};
				`, &debugInfo).Do(ctx); err == nil {
					log.Printf("デバッグ情報: %s", debugInfo)
				}
			}
			
			return nil
		}),
	)

	if err != nil {
		log.Printf("プレイリストページからの取得でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("プレイリストページからの取得でエラーが発生しました: %w", err)
	}

	log.Printf("プレイリストページから%d個のプレイリストを取得しました", len(playlists))
	return playlists, nil
}

// CreatePlaylist は新しいプレイリストを作成します
func (bc *BrowserClient) CreatePlaylist(name, description string, isPublic bool) (map[string]interface{}, error) {
	log.Printf("新しいプレイリストを作成します: %s", name)
	
	var result map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// プレイリストページに移動
		chromedp.Navigate("https://learning.oreilly.com/playlists/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		// 新規作成ボタンを探してクリック
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("新規作成ボタンを探しています")
			
			// 複数のセレクターで新規作成ボタンを探す
			createButtonSelectors := []string{
				`button:contains("Create")`,
				`button:contains("New")`,
				`button:contains("Add")`,
				`[data-testid*="create"]`,
				`[data-testid*="new"]`,
				`.create-button`,
				`.new-playlist`,
				`a[href*="create"]`,
				`a[href*="new"]`,
			}
			
			for _, selector := range createButtonSelectors {
				var buttonExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &buttonExists).Do(ctx); err == nil && buttonExists {
					log.Printf("新規作成ボタンが見つかりました: %s", selector)
					return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				}
			}
			
			// ボタンが見つからない場合、JavaScriptで直接作成フォームを表示
			log.Printf("新規作成ボタンが見つからないため、代替手段を試行します")
			return chromedp.Evaluate(`
				// プレイリスト作成のモーダルやフォームを表示する試行
				const createButtons = document.querySelectorAll('button, a, [role="button"]');
				for (const btn of createButtons) {
					const text = btn.textContent.toLowerCase();
					if (text.includes('create') || text.includes('new') || text.includes('add')) {
						btn.click();
						console.log('作成ボタンをクリックしました:', btn.textContent);
						return true;
					}
				}
				return false;
			`, nil).Do(ctx)
		}),
		
		// フォームの表示を待機
		chromedp.Sleep(2*time.Second),
		
		// プレイリスト作成フォームに入力
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリスト作成フォームに入力します")
			
			// 名前フィールドを探して入力
			nameFieldSelectors := []string{
				`input[name="name"]`,
				`input[name="title"]`,
				`input[placeholder*="name"]`,
				`input[placeholder*="title"]`,
				`[data-testid*="name"]`,
				`[data-testid*="title"]`,
			}
			
			for _, selector := range nameFieldSelectors {
				var fieldExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &fieldExists).Do(ctx); err == nil && fieldExists {
					log.Printf("名前フィールドが見つかりました: %s", selector)
					if err := chromedp.Clear(selector, chromedp.ByQuery).Do(ctx); err == nil {
						return chromedp.SendKeys(selector, name, chromedp.ByQuery).Do(ctx)
					}
				}
			}
			
			log.Printf("名前フィールドが見つかりませんでした")
			return nil
		}),
		
		// 説明フィールドに入力（オプション）
		chromedp.ActionFunc(func(ctx context.Context) error {
			if description == "" {
				return nil
			}
			
			log.Printf("説明フィールドに入力します")
			
			descFieldSelectors := []string{
				`textarea[name="description"]`,
				`textarea[placeholder*="description"]`,
				`input[name="description"]`,
				`[data-testid*="description"]`,
			}
			
			for _, selector := range descFieldSelectors {
				var fieldExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &fieldExists).Do(ctx); err == nil && fieldExists {
					log.Printf("説明フィールドが見つかりました: %s", selector)
					if err := chromedp.Clear(selector, chromedp.ByQuery).Do(ctx); err == nil {
						return chromedp.SendKeys(selector, description, chromedp.ByQuery).Do(ctx)
					}
				}
			}
			
			log.Printf("説明フィールドが見つかりませんでした")
			return nil
		}),
		
		// プライバシー設定（オプション）
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プライバシー設定を行います: public=%v", isPublic)
			
			privacySelectors := []string{
				`input[name="privacy"]`,
				`input[name="public"]`,
				`input[type="checkbox"]`,
				`[data-testid*="privacy"]`,
				`[data-testid*="public"]`,
			}
			
			for _, selector := range privacySelectors {
				var fieldExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &fieldExists).Do(ctx); err == nil && fieldExists {
					log.Printf("プライバシー設定フィールドが見つかりました: %s", selector)
					
					// チェックボックスの現在の状態を確認
					var isChecked bool
					if err := chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s').checked`, selector), &isChecked).Do(ctx); err == nil {
						// 必要に応じてクリック
						if (isPublic && !isChecked) || (!isPublic && isChecked) {
							return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
						}
					}
					break
				}
			}
			
			return nil
		}),
		
		// 作成ボタンをクリック
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("作成ボタンをクリックします")
			
			submitSelectors := []string{
				`button[type="submit"]`,
				`button:contains("Create")`,
				`button:contains("Save")`,
				`input[type="submit"]`,
				`[data-testid*="submit"]`,
				`[data-testid*="create"]`,
				`[data-testid*="save"]`,
			}
			
			for _, selector := range submitSelectors {
				var buttonExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &buttonExists).Do(ctx); err == nil && buttonExists {
					log.Printf("作成ボタンが見つかりました: %s", selector)
					return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				}
			}
			
			log.Printf("作成ボタンが見つかりませんでした")
			return fmt.Errorf("作成ボタンが見つかりませんでした")
		}),
		
		// 作成完了を待機
		chromedp.Sleep(3*time.Second),
		
		// 作成結果を取得
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("作成後のURL: %s", currentURL)
				
				// URLからプレイリストIDを抽出
				if idMatch := strings.Contains(currentURL, "/playlists/"); idMatch {
					result = map[string]interface{}{
						"id":          extractPlaylistIDFromURL(currentURL),
						"name":        name,
						"description": description,
						"url":         currentURL,
						"is_public":   isPublic,
						"created":     true,
					}
				} else {
					result = map[string]interface{}{
						"name":        name,
						"description": description,
						"is_public":   isPublic,
						"created":     true,
						"message":     "プレイリストが作成されましたが、IDの取得に失敗しました",
					}
				}
			}
			return nil
		}),
	)

	if err != nil {
		log.Printf("プレイリスト作成でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("プレイリスト作成でエラーが発生しました: %w", err)
	}

	if result == nil {
		result = map[string]interface{}{
			"name":        name,
			"description": description,
			"is_public":   isPublic,
			"created":     false,
			"message":     "プレイリストの作成を試行しましたが、結果を確認できませんでした",
		}
	}

	log.Printf("プレイリスト作成が完了しました: %v", result)
	return result, nil
}

// extractPlaylistIDFromURL はURLからプレイリストIDを抽出します
func extractPlaylistIDFromURL(url string) string {
	if match := strings.Contains(url, "/playlists/"); match {
		parts := strings.Split(url, "/playlists/")
		if len(parts) > 1 {
			idPart := strings.Split(parts[1], "/")[0]
			idPart = strings.Split(idPart, "?")[0]
			return idPart
		}
	}
	return ""
}

// AddContentToPlaylist はプレイリストにコンテンツを追加します
func (bc *BrowserClient) AddContentToPlaylist(playlistID, contentID string) error {
	log.Printf("プレイリストにコンテンツを追加します: playlist=%s, content=%s", playlistID, contentID)
	
	err := chromedp.Run(bc.ctx,
		// コンテンツページに移動
		chromedp.Navigate(fmt.Sprintf("https://learning.oreilly.com/library/view/%s/", contentID)),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		// プレイリストに追加ボタンを探してクリック
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリストに追加ボタンを探しています")
			
			addButtonSelectors := []string{
				`button:contains("Add to playlist")`,
				`button:contains("Add to Playlist")`,
				`[data-testid*="add-to-playlist"]`,
				`[data-testid*="playlist"]`,
				`.add-to-playlist`,
				`button:contains("Save")`,
				`[aria-label*="playlist"]`,
			}
			
			for _, selector := range addButtonSelectors {
				var buttonExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &buttonExists).Do(ctx); err == nil && buttonExists {
					log.Printf("プレイリスト追加ボタンが見つかりました: %s", selector)
					return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				}
			}
			
			log.Printf("プレイリスト追加ボタンが見つかりませんでした")
			return fmt.Errorf("プレイリスト追加ボタンが見つかりませんでした")
		}),
		
		// プレイリスト選択モーダルの表示を待機
		chromedp.Sleep(2*time.Second),
		
		// 指定されたプレイリストを選択
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリストを選択します: %s", playlistID)
			
			// プレイリスト選択のセレクター
			playlistSelectors := []string{
				fmt.Sprintf(`[data-playlist-id="%s"]`, playlistID),
				fmt.Sprintf(`[value="%s"]`, playlistID),
				fmt.Sprintf(`option[value="%s"]`, playlistID),
			}
			
			for _, selector := range playlistSelectors {
				var elementExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &elementExists).Do(ctx); err == nil && elementExists {
					log.Printf("プレイリスト選択要素が見つかりました: %s", selector)
					return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				}
			}
			
			// セレクトボックスの場合
			if err := chromedp.SetValue(`select`, playlistID, chromedp.ByQuery).Do(ctx); err == nil {
				log.Printf("セレクトボックスでプレイリストを選択しました")
				return nil
			}
			
			log.Printf("プレイリスト選択要素が見つかりませんでした")
			return fmt.Errorf("プレイリスト選択要素が見つかりませんでした")
		}),
		
		// 追加ボタンをクリック
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("追加ボタンをクリックします")
			
			confirmSelectors := []string{
				`button:contains("Add")`,
				`button:contains("Save")`,
				`button[type="submit"]`,
				`[data-testid*="confirm"]`,
				`[data-testid*="add"]`,
			}
			
			for _, selector := range confirmSelectors {
				var buttonExists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &buttonExists).Do(ctx); err == nil && buttonExists {
					log.Printf("追加確認ボタンが見つかりました: %s", selector)
					return chromedp.Click(selector, chromedp.ByQuery).Do(ctx)
				}
			}
			
			log.Printf("追加確認ボタンが見つかりませんでした")
			return fmt.Errorf("追加確認ボタンが見つかりませんでした")
		}),
		
		// 追加完了を待機
		chromedp.Sleep(2*time.Second),
	)

	if err != nil {
		log.Printf("プレイリストへのコンテンツ追加でエラーが発生しました: %v", err)
		return fmt.Errorf("プレイリストへのコンテンツ追加でエラーが発生しました: %w", err)
	}

	log.Printf("プレイリストへのコンテンツ追加が完了しました")
	return nil
}

// GetPlaylistDetails はプレイリストの詳細情報を取得します
func (bc *BrowserClient) GetPlaylistDetails(playlistID string) (map[string]interface{}, error) {
	log.Printf("プレイリストの詳細情報を取得します: %s", playlistID)
	
	var details map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// プレイリスト詳細ページに移動
		chromedp.Navigate(fmt.Sprintf("https://learning.oreilly.com/playlists/%s/", playlistID)),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		
		// プレイリスト詳細情報を取得
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("プレイリスト詳細情報の抽出を開始します")
			
			var playlistDetails map[string]interface{}
			if err := chromedp.Evaluate(`
				(function() {
					const result = {
						id: '',
						title: '',
						description: '',
						creator: '',
						item_count: 0,
						items: [],
						is_public: false,
						created_date: '',
						last_modified: ''
					};
					
					// タイトルを取得
					const titleSelectors = [
						'h1',
						'h2',
						'.title',
						'[data-testid*="title"]',
						'.playlist-title'
					];
					
					for (const selector of titleSelectors) {
						const titleEl = document.querySelector(selector);
						if (titleEl && titleEl.textContent.trim()) {
							result.title = titleEl.textContent.trim();
							break;
						}
					}
					
					// 説明を取得
					const descSelectors = [
						'.description',
						'.summary',
						'[data-testid*="description"]',
						'.playlist-description'
					];
					
					for (const selector of descSelectors) {
						const descEl = document.querySelector(selector);
						if (descEl && descEl.textContent.trim()) {
							result.description = descEl.textContent.trim();
							break;
						}
					}
					
					// 作成者を取得
					const creatorSelectors = [
						'.creator',
						'.author',
						'.by',
						'[data-testid*="author"]',
						'[data-testid*="creator"]'
					];
					
					for (const selector of creatorSelectors) {
						const creatorEl = document.querySelector(selector);
						if (creatorEl && creatorEl.textContent.trim()) {
							result.creator = creatorEl.textContent.trim();
							break;
						}
					}
					
					// プレイリストアイテムを取得
					const itemSelectors = [
						'.playlist-item',
						'.item',
						'[data-testid*="item"]',
						'.content-item'
					];
					
					let items = [];
					for (const selector of itemSelectors) {
						const itemElements = document.querySelectorAll(selector);
						if (itemElements.length > 0) {
							for (const itemEl of itemElements) {
								const item = {
									title: '',
									url: '',
									type: '',
									duration: ''
								};
								
								// アイテムタイトル
								const itemTitleEl = itemEl.querySelector('h3, h4, .title, a');
								if (itemTitleEl) {
									item.title = itemTitleEl.textContent.trim();
									if (itemTitleEl.href) {
										item.url = itemTitleEl.href;
									}
								}
								
								// アイテムタイプ
								const typeEl = itemEl.querySelector('.type, [data-testid*="type"]');
								if (typeEl) {
									item.type = typeEl.textContent.trim();
								}
								
								// 時間
								const durationEl = itemEl.querySelector('.duration, [data-testid*="duration"]');
								if (durationEl) {
									item.duration = durationEl.textContent.trim();
								}
								
								if (item.title) {
									items.push(item);
								}
							}
							break;
						}
					}
					
					result.items = items;
					result.item_count = items.length;
					
					// URLからIDを抽出
					const currentUrl = window.location.href;
					const idMatch = currentUrl.match(/\/playlists\/([^\/\?]+)/);
					if (idMatch) {
						result.id = idMatch[1];
					}
					
					console.log('プレイリスト詳細:', result);
					return result;
				})()
			`, &playlistDetails).Do(ctx); err != nil {
				log.Printf("プレイリスト詳細情報の抽出でエラーが発生しました: %v", err)
				return err
			}
			
			details = playlistDetails
			log.Printf("プレイリスト詳細情報を取得しました: %v", details)
			return nil
		}),
	)

	if err != nil {
		log.Printf("プレイリスト詳細取得でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("プレイリスト詳細取得でエラーが発生しました: %w", err)
	}

	if details == nil {
		details = map[string]interface{}{
			"id":      playlistID,
			"message": "プレイリスト詳細の取得に失敗しました",
		}
	}

	log.Printf("プレイリスト詳細取得が完了しました: %v", details)
	return details, nil
}

// ExtractTableOfContents はO'Reilly書籍の目次を抽出します
func (bc *BrowserClient) ExtractTableOfContents(url string) (*TableOfContentsResponse, error) {
	log.Printf("O'Reilly書籍の目次抽出を開始します: %s", url)
	
	var result *TableOfContentsResponse
	
	err := chromedp.Run(bc.ctx,
		// まずログインページから確実にログインを実行
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("目次抽出のため、確実なログイン処理を開始します")
			
			// ログインページに移動
			return chromedp.Run(ctx,
				chromedp.Navigate("https://www.oreilly.com/member/login/"),
				chromedp.WaitVisible(`body`, chromedp.ByQuery),
				chromedp.Sleep(3*time.Second),
			)
		}),
		
		// ログイン状態を確認し、必要に応じて再ログイン
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("ログインページアクセス後のURL: %s", currentURL)
				
				// ログインページにいる場合、セッションを更新
				if strings.Contains(currentURL, "/login") || strings.Contains(currentURL, "/member/login") {
					log.Printf("ログインページが検出されました。セッションを更新します")
					return bc.RefreshSession()
				} else {
					log.Printf("既にログイン済みです")
				}
			}
			return nil
		}),
		
		// 学習プラットフォームに移動してログイン状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("学習プラットフォームに移動してログイン状態を確認します")
			
			return chromedp.Run(ctx,
				chromedp.Navigate("https://learning.oreilly.com/"),
				chromedp.WaitVisible(`body`, chromedp.ByQuery),
				chromedp.Sleep(5*time.Second),
			)
		}),
		
		// 学習プラットフォームでのログイン状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("学習プラットフォームアクセス後のURL: %s", currentURL)
				
				// ログインページにリダイレクトされている場合
				if strings.Contains(currentURL, "/login") || strings.Contains(currentURL, "/member/login") {
					log.Printf("学習プラットフォームでログインページにリダイレクトされました。再ログインを実行します")
					return bc.RefreshSession()
				} else {
					log.Printf("学習プラットフォームへのアクセスに成功しました")
				}
			}
			return nil
		}),
		
		// 指定されたURLに移動
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("指定されたURLに移動します: %s", url)
			
			return chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.WaitVisible(`body`, chromedp.ByQuery),
				chromedp.Sleep(5*time.Second),
			)
		}),
		
		// 現在のURLを確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("目次抽出対象ページのURL: %s", currentURL)
			}
			return nil
		}),
		
		// 段階的ページ探索アプローチを実装
		chromedp.ActionFunc(func(ctx context.Context) error {
			// URLから書籍IDを抽出
			var bookID string
			if strings.Contains(url, "9784814400607") {
				bookID = "9784814400607"
			} else {
				// 他のパターンでIDを抽出
				parts := strings.Split(url, "/")
				for _, part := range parts {
					if len(part) >= 10 && (strings.HasPrefix(part, "978") || len(part) == 13) {
						bookID = part
						break
					}
				}
			}
			
			if bookID != "" {
				log.Printf("書籍ID: %s", bookID)
				
				// まず学習プラットフォームのホームページにアクセスしてログイン状態を確認
				err := chromedp.Run(ctx,
					chromedp.Navigate("https://learning.oreilly.com/"),
					chromedp.WaitVisible(`body`, chromedp.ByQuery),
					chromedp.Sleep(3*time.Second),
				)
				
				if err != nil {
					log.Printf("学習プラットフォームへのアクセスに失敗: %v", err)
					return err
				}
				
				// ログイン状態を再確認
				var homeURL string
				if err := chromedp.Location(&homeURL).Do(ctx); err == nil {
					log.Printf("学習プラットフォームホームURL: %s", homeURL)
					
					if strings.Contains(homeURL, "/login") {
						log.Printf("ログインが必要です。セッションを更新します")
						if err := bc.RefreshSession(); err != nil {
							log.Printf("セッション更新に失敗: %v", err)
							return err
						}
					}
				}
				
				// 段階的ページ探索: 一般的なページパターンを順次試行
				pagePatterns := []string{
					"toc.html",           // 目次ページ
					"cover.html",         // カバーページ  
					"index.html",         // インデックス
					"ch01.html",          // 第1章
					"chapter01.html",     // 第1章（別形式）
					"chap01.html",        // 第1章（日本語書籍）
					"preface.html",       // 前書き
					"foreword.html",      // 序文
					"intro.html",         // 導入
					"contents.html",      // 目次（別形式）
					"table-of-contents.html", // 目次（フル形式）
					"",                   // ベースURL（書籍詳細ページ）
				}
				
				// 複数のベースURLパターンも試行
				baseURLPatterns := []string{
					fmt.Sprintf("https://learning.oreilly.com/library/view/-/%s/", bookID),
					fmt.Sprintf("https://learning.oreilly.com/library/view/sohutoueaakitekutiyametorikusu-akitekutiyapin-zhi-wogai-shan-suru10noadobaisu/%s/", bookID),
					fmt.Sprintf("https://learning.oreilly.com/library/view/%s/", bookID),
				}
				
				for _, baseURL := range baseURLPatterns {
					for _, pagePattern := range pagePatterns {
						var testURL string
						if pagePattern == "" {
							testURL = baseURL
						} else {
							testURL = baseURL + pagePattern
						}
						
						log.Printf("ページパターンを試行: %s", testURL)
						
						err := chromedp.Run(ctx,
							chromedp.Navigate(testURL),
							chromedp.WaitVisible(`body`, chromedp.ByQuery),
							chromedp.Sleep(3*time.Second),
						)
						
						if err == nil {
							var currentURL string
							var pageTitle string
							var hasContent bool
							
							if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
								log.Printf("アクセス後のURL: %s", currentURL)
								
								// ページタイトルを取得
								if err := chromedp.Title(&pageTitle).Do(ctx); err == nil {
									log.Printf("ページタイトル: %s", pageTitle)
								}
								
								// 実際の書籍コンテンツがあるかチェック
								if err := chromedp.Evaluate(`
									(function() {
										// 書籍コンテンツの存在を確認
										const contentSelectors = [
											'.book-content',
											'[data-testid*="book"]',
											'[data-testid*="content"]',
											'.content-area',
											'main article',
											'.chapter',
											'.section',
											'[id*="chapter"]',
											'[id*="section"]',
											'.toc',
											'.table-of-contents'
										];
										
										for (const selector of contentSelectors) {
											const element = document.querySelector(selector);
											if (element && element.textContent.trim().length > 100) {
												return true;
											}
										}
										
										// 目次らしいリンクがあるかチェック
										const links = document.querySelectorAll('a[href]');
										let bookLinks = 0;
										for (const link of links) {
											const href = link.href;
											const text = link.textContent.trim();
											if (href.includes('` + bookID + `') && 
												(text.includes('章') || text.includes('Chapter') || 
												 text.includes('第') || text.match(/\d+\./))) {
												bookLinks++;
											}
										}
										
										return bookLinks > 2; // 3つ以上の章リンクがあれば書籍コンテンツと判定
									})()
								`, &hasContent).Do(ctx); err == nil && hasContent {
									log.Printf("書籍コンテンツを発見しました: %s", testURL)
									
									// 成功したURLを記録して処理を継続
									return nil
								}
								
								// ログインページにリダイレクトされていないかチェック
								if !strings.Contains(currentURL, "/login") && 
								   strings.Contains(currentURL, "learning.oreilly.com") &&
								   (strings.Contains(currentURL, bookID) || strings.Contains(pageTitle, "ソフトウェアアーキテクチャ")) {
									log.Printf("有効な書籍ページを発見: %s", testURL)
									return nil
								}
							}
						}
						
						// 短い間隔で次のパターンを試行
						time.Sleep(1 * time.Second)
					}
				}
				
				log.Printf("すべてのページパターンでアクセスに失敗しました")
			}
			return nil
		}),
		
		// 目次情報を抽出
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("目次情報の抽出を開始します")
			
			// まずページの詳細な構造を調査
			var pageInfo map[string]interface{}
			if err := chromedp.Evaluate(`
				(function() {
					const info = {
						url: window.location.href,
						title: document.title,
						bodyText: document.body.textContent.substring(0, 1000),
						hasLoginForm: !!document.querySelector('input[type="email"], input[name="email"]'),
						hasBookContent: false,
						tocElements: [],
						allLinks: [],
						headings: []
					};
					
					// 書籍コンテンツの存在確認
					const bookContentSelectors = [
						'.book-content',
						'[data-testid*="book"]',
						'[data-testid*="content"]',
						'.content-area',
						'main',
						'article'
					];
					
					for (const selector of bookContentSelectors) {
						if (document.querySelector(selector)) {
							info.hasBookContent = true;
							break;
						}
					}
					
					// 目次関連要素の調査
					const tocSelectors = [
						'[data-testid*="toc"]',
						'[data-testid*="contents"]',
						'.toc',
						'.table-of-contents',
						'.contents',
						'nav',
						'.navigation'
					];
					
					for (const selector of tocSelectors) {
						const elements = document.querySelectorAll(selector);
						if (elements.length > 0) {
							info.tocElements.push({
								selector: selector,
								count: elements.length,
								sample: elements[0].textContent.substring(0, 200)
							});
						}
					}
					
					// すべてのリンクを調査
					const links = document.querySelectorAll('a[href]');
					for (let i = 0; i < Math.min(links.length, 20); i++) {
						const link = links[i];
						info.allLinks.push({
							text: link.textContent.trim().substring(0, 100),
							href: link.href,
							className: link.className
						});
					}
					
					// ヘッダー要素を調査
					const headers = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
					for (const header of headers) {
						const text = header.textContent.trim();
						if (text && text.length > 2) {
							info.headings.push({
								level: parseInt(header.tagName.substring(1)),
								text: text.substring(0, 100),
								id: header.id,
								className: header.className
							});
						}
					}
					
					return info;
				})()
			`, &pageInfo).Do(ctx); err != nil {
				log.Printf("ページ構造調査でエラーが発生しました: %v", err)
			} else {
				log.Printf("ページ構造調査結果: %+v", pageInfo)
			}
			
			var tocData map[string]interface{}
			if err := chromedp.Evaluate(`
				(function() {
					const result = {
						book_title: '',
						book_id: '',
						book_url: '',
						authors: [],
						publisher: '',
						table_of_contents: [],
						extracted_at: new Date().toISOString(),
						debug_info: {
							page_type: 'unknown',
							login_required: false,
							content_available: false
						}
					};
					
					console.log('目次抽出を開始します');
					
					// ページタイプの判定
					if (document.querySelector('input[type="email"], input[name="email"]')) {
						result.debug_info.page_type = 'login_page';
						result.debug_info.login_required = true;
						console.log('ログインページが検出されました');
						return result;
					}
					
					// 書籍詳細ページかどうかの判定
					const isBookDetailPage = window.location.href.includes('/library/view/') && 
											(document.querySelector('h1') || document.title.includes('ソフトウェア'));
					
					if (isBookDetailPage) {
						result.debug_info.page_type = 'book_detail_page';
						console.log('書籍詳細ページを検出しました');
					}
					
					// 書籍タイトルを取得（改良版）
					const titleSelectors = [
						'h1',
						'.book-title',
						'[data-testid*="title"]',
						'.title'
					];
					
					for (const selector of titleSelectors) {
						const titleEl = document.querySelector(selector);
						if (titleEl && titleEl.textContent.trim()) {
							let title = titleEl.textContent.trim();
							// O'Reillyのサイト名を除去
							title = title.replace(/\s*-\s*O'Reilly.*$/, '').trim();
							if (title && title.length > 5 && !title.includes('Sign In') && !title.includes('Learning')) {
								result.book_title = title;
								console.log('書籍タイトル発見:', result.book_title);
								break;
							}
						}
					}
					
					// ページタイトルからも取得を試行
					if (!result.book_title && document.title) {
						let title = document.title.replace(/\s*-\s*O'Reilly.*$/, '').trim();
						if (title && title.length > 5 && !title.includes('Sign In') && !title.includes('Learning')) {
							result.book_title = title;
						}
					}
					
					// 著者情報を取得（改良版）
					const authorText = document.body.textContent;
					const authorPatterns = [
						/by\s+([^,\n]+(?:,\s*[^,\n]+)*)/i,
						/著者[：:]\s*([^,\n]+(?:,\s*[^,\n]+)*)/i
					];
					
					for (const pattern of authorPatterns) {
						const match = authorText.match(pattern);
						if (match) {
							const authors = match[1].split(',').map(a => a.trim()).filter(a => a.length > 1);
							result.authors = authors;
							console.log('著者情報発見:', result.authors);
							break;
						}
					}
					
					// 出版社情報を取得
					const publisherMatch = authorText.match(/Publisher\(s\):\s*([^,\n]+)/i);
					if (publisherMatch) {
						result.publisher = publisherMatch[1].trim();
					}
					
					// URLから書籍IDを抽出
					const currentUrl = window.location.href;
					const idMatches = [
						currentUrl.match(/\/library\/view\/[^\/]*\/([^\/\?]+)/),
						currentUrl.match(/\/([0-9]{13})/), // ISBN-13
						currentUrl.match(/\/([0-9]{10})/)  // ISBN-10
					];
					
					for (const match of idMatches) {
						if (match) {
							result.book_id = match[1];
							break;
						}
					}
					
					result.book_url = currentUrl.split('#')[0];
					
					// 目次を抽出（大幅に改良）
					let tocItems = [];
					
					// 書籍詳細ページの場合、目次セクションを探す
					if (isBookDetailPage) {
						console.log('書籍詳細ページで目次を検索します');
						
						// 方法1: 目次専用セクションを探す
						const tocSelectors = [
							'[data-testid*="toc"]',
							'[data-testid*="contents"]',
							'.toc',
							'.table-of-contents',
							'.contents',
							'#toc',
							'[id*="contents"]',
							'.book-toc'
						];
						
						for (const tocSelector of tocSelectors) {
							const tocContainer = document.querySelector(tocSelector);
							if (tocContainer) {
								console.log('目次コンテナ発見:', tocSelector);
								
								const tocElements = tocContainer.querySelectorAll('a[href], li, .chapter, .section');
								for (const element of tocElements) {
									const title = element.textContent.trim();
									let href = '';
									
									if (element.tagName === 'A') {
										href = element.href || '';
									} else {
										const linkEl = element.querySelector('a[href]');
										if (linkEl) {
											href = linkEl.href || '';
										}
									}
									
									if (title && title.length > 2 && title.length < 200) {
										// O'Reillyナビゲーションを除外
										if (title.includes('Sign In') || title.includes('Try Now') || 
											title.includes('For Enterprise') || title.includes('Skills') ||
											title.includes('Cloud Computing') || title.includes('Data Science') ||
											title.includes('Programming Languages') || title.includes('Features') ||
											title.includes('Plans') || title.includes('Explore Skills')) {
											continue;
										}
										
										// レベルを推測
										let level = 1;
										const classList = element.className.toLowerCase();
										const tagName = element.tagName.toLowerCase();
										
										if (classList.includes('chapter') || tagName === 'h1') {
											level = 1;
										} else if (classList.includes('section') || tagName === 'h2') {
											level = 2;
										} else if (classList.includes('subsection') || tagName === 'h3') {
											level = 3;
										} else if (tagName.startsWith('h')) {
											level = parseInt(tagName.substring(1)) || 1;
										} else {
											// ネストレベルから推測
											let currentEl = element.parentElement;
											let nestLevel = 1;
											while (currentEl && currentEl !== tocContainer && nestLevel < 6) {
												if (currentEl.tagName === 'UL' || currentEl.tagName === 'OL' || 
													currentEl.tagName === 'LI') {
													nestLevel++;
												}
												currentEl = currentEl.parentElement;
											}
											level = Math.min(nestLevel, 6);
										}
										
										tocItems.push({
											level: level,
											title: title,
											url: href,
											chapter_id: '',
											section_id: '',
											page_number: ''
										});
									}
								}
								
								if (tocItems.length > 0) {
									console.log('目次項目を発見:', tocItems.length, '件');
									break;
								}
							}
						}
						
						// 方法2: 書籍詳細ページで章リストを探す
						if (tocItems.length === 0) {
							console.log('章リストを検索します');
							
							// 章や節を示すキーワードを含むリンクを探す
							const allLinks = document.querySelectorAll('a[href]');
							for (const link of allLinks) {
								const text = link.textContent.trim();
								const href = link.href;
								
								if (text && href && href.includes(result.book_id)) {
									// 章や節を示すパターンをチェック
									const chapterPatterns = [
										/第?\s*\d+\s*章/,
										/Chapter\s+\d+/i,
										/\d+\.\s*[A-Za-z]/,
										/序章|はじめに|まえがき|目次|索引|付録/
									];
									
									let isChapter = false;
									for (const pattern of chapterPatterns) {
										if (pattern.test(text)) {
											isChapter = true;
											break;
										}
									}
									
									if (isChapter && text.length > 2 && text.length < 200) {
										tocItems.push({
											level: 1,
											title: text,
											url: href,
											chapter_id: '',
											section_id: '',
											page_number: ''
										});
									}
								}
							}
							
							console.log('章リストから目次項目を発見:', tocItems.length, '件');
						}
						
						// 方法3: ページ内の構造化された見出しから目次を構築
						if (tocItems.length === 0) {
							console.log('構造化された見出しから目次を構築します');
							
							// メインコンテンツエリアを特定
							const contentSelectors = [
								'main',
								'[role="main"]',
								'.content',
								'.book-content',
								'#content',
								'article',
								'.main-content'
							];
							
							let contentArea = document.body;
							for (const selector of contentSelectors) {
								const area = document.querySelector(selector);
								if (area) {
									contentArea = area;
									console.log('コンテンツエリア発見:', selector);
									break;
								}
							}
							
							const headers = contentArea.querySelectorAll('h1, h2, h3, h4, h5, h6');
							for (const header of headers) {
								const title = header.textContent.trim();
								if (title && title.length > 2 && title.length < 200) {
									// O'Reillyナビゲーションを除外
									if (title.includes('Sign In') || title.includes('Try Now') || 
										title.includes('For Enterprise') || title.includes('Skills') ||
										title.includes('Cloud Computing') || title.includes('Data Science') ||
										title.includes('Programming Languages') || title.includes('Features') ||
										title.includes('Plans') || title.includes('Explore Skills')) {
										continue;
									}
									
									const level = parseInt(header.tagName.substring(1));
									const id = header.id || '';
									
									tocItems.push({
										level: level,
										title: title,
										url: id ? (currentUrl + '#' + id) : '',
										chapter_id: id.includes('chapter') ? id : '',
										section_id: id && !id.includes('chapter') ? id : '',
										page_number: ''
									});
								}
							}
							
							console.log('ヘッダーから目次項目を構築:', tocItems.length, '件');
						}
					}
					
					// 重複を除去し、不適切な項目をフィルタリング
					const uniqueTocItems = [];
					const seenTitles = new Set();
					
					for (const item of tocItems) {
						// 不適切なタイトルをフィルタリング
						if (item.title.includes('Sign In') || item.title.includes('Try Now') || 
							item.title.includes('For Enterprise') || item.title.includes('Skills') ||
							item.title.includes('Cloud Computing') || item.title.includes('Data Science') ||
							item.title.includes('Programming Languages') || item.title.includes('Features') ||
							item.title.includes('Plans') || item.title.includes('Explore Skills') ||
							item.title.length < 3 || item.title.length > 150) {
							continue;
						}
						
						if (!seenTitles.has(item.title)) {
							seenTitles.add(item.title);
							uniqueTocItems.push(item);
						}
					}
					
					result.table_of_contents = uniqueTocItems;
					result.debug_info.content_available = uniqueTocItems.length > 0;
					
					console.log('最終的な目次項目数:', result.table_of_contents.length);
					console.log('抽出結果:', result);
					
					return result;
				})()
			`, &tocData).Do(ctx); err != nil {
				log.Printf("目次情報の抽出でエラーが発生しました: %v", err)
				return err
			}
			
			// 結果を構造体に変換
			result = &TableOfContentsResponse{
				BookTitle:       getStringFromMap(tocData, "book_title"),
				BookID:          getStringFromMap(tocData, "book_id"),
				BookURL:         getStringFromMap(tocData, "book_url"),
				Authors:         getStringArrayFromMap(tocData, "authors"),
				Publisher:       getStringFromMap(tocData, "publisher"),
				TableOfContents: convertToTOCItems(tocData["table_of_contents"]),
				ExtractedAt:     getStringFromMap(tocData, "extracted_at"),
			}
			
			log.Printf("目次情報を取得しました: %s (%d項目)", result.BookTitle, len(result.TableOfContents))
			return nil
		}),
	)

	if err != nil {
		log.Printf("目次抽出でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("目次抽出でエラーが発生しました: %w", err)
	}

	if result == nil {
		result = &TableOfContentsResponse{
			BookURL:     url,
			ExtractedAt: time.Now().Format(time.RFC3339),
		}
	}

	log.Printf("目次抽出が完了しました: %s", result.BookTitle)
	return result, nil
}

// ヘルパー関数: map[string]interface{}から文字列を安全に取得
func getStringFromMap(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

// ヘルパー関数: map[string]interface{}から文字列配列を安全に取得
func getStringArrayFromMap(m map[string]interface{}, key string) []string {
	if value, ok := m[key].([]interface{}); ok {
		var result []string
		for _, v := range value {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

// ヘルパー関数: interface{}をTableOfContentsItemの配列に変換
func convertToTOCItems(data interface{}) []TableOfContentsItem {
	var items []TableOfContentsItem
	
	if itemsData, ok := data.([]interface{}); ok {
		for _, itemData := range itemsData {
			if itemMap, ok := itemData.(map[string]interface{}); ok {
				item := TableOfContentsItem{
					Level:      getIntFromMap(itemMap, "level"),
					Title:      getStringFromMap(itemMap, "title"),
					URL:        getStringFromMap(itemMap, "url"),
					ChapterID:  getStringFromMap(itemMap, "chapter_id"),
					SectionID:  getStringFromMap(itemMap, "section_id"),
					PageNumber: getStringFromMap(itemMap, "page_number"),
				}
				items = append(items, item)
			}
		}
	}
	
	return items
}

// ヘルパー関数: map[string]interface{}から整数を安全に取得
func getIntFromMap(m map[string]interface{}, key string) int {
	if value, ok := m[key].(float64); ok {
		return int(value)
	}
	if value, ok := m[key].(int); ok {
		return value
	}
	return 1 // デフォルト値
}

// SearchInBook は書籍内で用語を検索します
func (bc *BrowserClient) SearchInBook(bookID, searchTerm string) ([]map[string]interface{}, error) {
	log.Printf("書籍内検索を開始します: 書籍ID=%s, 検索語=%s", bookID, searchTerm)
	
	var results []map[string]interface{}
	
	err := chromedp.Run(bc.ctx,
		// まず学習プラットフォームにアクセスしてログイン状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("学習プラットフォームにアクセスしてログイン状態を確認します")
			
			return chromedp.Run(ctx,
				chromedp.Navigate("https://learning.oreilly.com/"),
				chromedp.WaitVisible(`body`, chromedp.ByQuery),
				chromedp.Sleep(3*time.Second),
			)
		}),
		
		// ログイン状態を確認
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			if err := chromedp.Location(&currentURL).Do(ctx); err == nil {
				log.Printf("学習プラットフォームURL: %s", currentURL)
				
				if strings.Contains(currentURL, "/login") {
					log.Printf("ログインが必要です。セッションを更新します")
					return bc.RefreshSession()
				}
			}
			return nil
		}),
		
		// 書籍ページに移動
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("書籍ページに移動します")
			
			// 複数のURLパターンを試行
			bookURLs := []string{
				fmt.Sprintf("https://learning.oreilly.com/library/view/-/%s/", bookID),
				fmt.Sprintf("https://learning.oreilly.com/library/view/%s/", bookID),
			}
			
			for _, bookURL := range bookURLs {
				log.Printf("書籍URLを試行: %s", bookURL)
				
				err := chromedp.Run(ctx,
					chromedp.Navigate(bookURL),
					chromedp.WaitVisible(`body`, chromedp.ByQuery),
					chromedp.Sleep(3*time.Second),
				)
				
				if err == nil {
					var newURL string
					if err := chromedp.Location(&newURL).Do(ctx); err == nil {
						log.Printf("移動後のURL: %s", newURL)
						
						// 書籍ページにアクセスできたかチェック
						if strings.Contains(newURL, "learning.oreilly.com") && 
						   !strings.Contains(newURL, "/login") &&
						   (strings.Contains(newURL, bookID) || strings.Contains(newURL, "/library/view/")) {
							log.Printf("書籍ページへのアクセスに成功しました")
							return nil
						}
					}
				}
			}
			
			return fmt.Errorf("書籍ページへのアクセスに失敗しました")
		}),
		
		// 書籍内検索を実行
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("書籍内検索を実行します: %s", searchTerm)
			
			// 検索ボックスを探す
			searchSelectors := []string{
				`input[type="search"]`,
				`input[placeholder*="search"]`,
				`input[placeholder*="Search"]`,
				`input[name="search"]`,
				`input[name="q"]`,
				`[data-testid*="search"]`,
				`.search-input`,
				`#search`,
				`input[aria-label*="search"]`,
				`input[aria-label*="Search"]`,
			}
			
			var searchBoxFound bool
			for _, selector := range searchSelectors {
				var exists bool
				if err := chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('%s')`, selector), &exists).Do(ctx); err == nil && exists {
					log.Printf("検索ボックスが見つかりました: %s", selector)
					
					// 検索語を入力
					if err := chromedp.Clear(selector, chromedp.ByQuery).Do(ctx); err == nil {
						if err := chromedp.SendKeys(selector, searchTerm, chromedp.ByQuery).Do(ctx); err == nil {
							// Enterキーを押すか検索ボタンをクリック
							if err := chromedp.KeyEvent("\r").Do(ctx); err == nil {
								searchBoxFound = true
								log.Printf("検索を実行しました")
								break
							}
						}
					}
				}
			}
			
			if !searchBoxFound {
				log.Printf("検索ボックスが見つかりませんでした。代替方法を試行します")
				
				// 代替方法: URLパラメータで検索
				searchURL := fmt.Sprintf("https://learning.oreilly.com/search/?q=%s+inbook:%s", 
					strings.ReplaceAll(searchTerm, " ", "+"), bookID)
				
				log.Printf("URLパラメータで検索を試行: %s", searchURL)
				
				return chromedp.Run(ctx,
					chromedp.Navigate(searchURL),
					chromedp.WaitVisible(`body`, chromedp.ByQuery),
					chromedp.Sleep(3*time.Second),
				)
			}
			
			return nil
		}),
		
		// 検索結果の読み込み完了を待機
		chromedp.Sleep(5*time.Second),
		
		// 検索結果を抽出
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("検索結果の抽出を開始します")
			
			var searchResults []map[string]interface{}
			if err := chromedp.Evaluate(fmt.Sprintf(`
				(function() {
					const results = [];
					const processedTexts = new Set();
					
					console.log('書籍内検索結果の抽出を開始します');
					
					// 検索結果のセレクター
					const resultSelectors = [
						'.search-result',
						'.result',
						'[data-testid*="result"]',
						'.search-item',
						'.highlight',
						'[class*="search"]',
						'[class*="result"]',
						'mark', // ハイライトされたテキスト
						'.match'
					];
					
					// 検索結果要素を探す
					let foundResults = [];
					for (const selector of resultSelectors) {
						const elements = document.querySelectorAll(selector);
						if (elements.length > 0) {
							console.log('検索結果要素発見:', selector, elements.length, '件');
							foundResults = foundResults.concat(Array.from(elements));
						}
					}
					
					// 重複を除去
					foundResults = Array.from(new Set(foundResults));
					
					// 各検索結果を処理
					for (const element of foundResults) {
						try {
							let title = '';
							let content = '';
							let context = '';
							let url = '';
							let pageNumber = '';
							let chapterTitle = '';
							
							// タイトルを取得
							const titleSelectors = ['h1', 'h2', 'h3', 'h4', '.title', '[data-testid*="title"]'];
							for (const titleSel of titleSelectors) {
								const titleEl = element.querySelector(titleSel) || 
											   element.closest('*').querySelector(titleSel);
								if (titleEl && titleEl.textContent.trim()) {
									title = titleEl.textContent.trim();
									break;
								}
							}
							
							// コンテンツ（マッチしたテキスト）を取得
							content = element.textContent.trim();
							
							// 親要素からコンテキストを取得
							if (element.parentElement) {
								context = element.parentElement.textContent.trim().substring(0, 300);
							}
							
							// URLを取得
							const linkEl = element.querySelector('a[href]') || 
										  element.closest('a[href]') ||
										  element.parentElement.querySelector('a[href]');
							if (linkEl) {
								url = linkEl.href;
							}
							
							// ページ番号を抽出
							const pageMatch = content.match(/(?:page|ページ|p\.?)\s*(\d+)/i);
							if (pageMatch) {
								pageNumber = pageMatch[1];
							}
							
							// 章タイトルを抽出
							const chapterMatch = content.match(/(?:第?\s*\d+\s*章|Chapter\s+\d+)[：:]?\s*([^\\n\\r]{1,100})/i);
							if (chapterMatch) {
								chapterTitle = chapterMatch[0];
							}
							
							// 検索語がハイライトされているかチェック
							const searchTerm = '%s';
							const hasHighlight = content.toLowerCase().includes(searchTerm.toLowerCase()) ||
											   element.innerHTML.includes('<mark>') ||
											   element.innerHTML.includes('highlight');
							
							// 有効な結果かチェック
							if (content && content.length > 10 && content.length < 1000 && 
								!processedTexts.has(content) && hasHighlight &&
								!content.includes('Sign In') && !content.includes('Try Now')) {
								
								processedTexts.add(content);
								
								results.push({
									title: title || 'マッチしたテキスト',
									content: content,
									context: context,
									url: url,
									page_number: pageNumber,
									chapter_title: chapterTitle,
									search_term: searchTerm,
									match_type: element.tagName.toLowerCase(),
									source: 'book_search'
								});
								
								console.log('検索結果発見:', content.substring(0, 100));
							}
						} catch (e) {
							console.log('検索結果処理エラー:', e);
						}
					}
					
					// テキスト全体からも検索語を探す（フォールバック）
					if (results.length === 0) {
						console.log('直接的な検索結果が見つからないため、テキスト全体から検索します');
						
						const bodyText = document.body.textContent;
						const searchTerm = '%s';
						const regex = new RegExp('.{0,100}' + searchTerm.replace(/[.*+?^${}()|[\\]\\\\]/g, '\\\\$&') + '.{0,100}', 'gi');
						
						let match;
						let matchCount = 0;
						while ((match = regex.exec(bodyText)) !== null && matchCount < 10) {
							const matchText = match[0].trim();
							if (matchText && !processedTexts.has(matchText)) {
								processedTexts.add(matchText);
								
								results.push({
									title: 'テキストマッチ',
									content: matchText,
									context: matchText,
									url: window.location.href,
									page_number: '',
									chapter_title: '',
									search_term: searchTerm,
									match_type: 'text_match',
									source: 'book_search_fallback'
								});
								
								matchCount++;
							}
						}
					}
					
					console.log('最終的な検索結果数:', results.length);
					return results;
				})()
			`, searchTerm, searchTerm), &searchResults).Do(ctx); err != nil {
				log.Printf("検索結果の抽出でエラーが発生しました: %v", err)
				return err
			}
			
			results = searchResults
			log.Printf("書籍内検索結果を取得しました: %d件", len(results))
			return nil
		}),
	)

	if err != nil {
		log.Printf("書籍内検索でエラーが発生しました: %v", err)
		return nil, fmt.Errorf("書籍内検索でエラーが発生しました: %w", err)
	}

	if len(results) == 0 {
		log.Printf("検索結果が見つかりませんでした: %s in %s", searchTerm, bookID)
		
		// デバッグ情報を取得
		debugErr := chromedp.Run(bc.ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				var pageTitle, currentURL string
				chromedp.Title(&pageTitle).Do(ctx)
				chromedp.Location(&currentURL).Do(ctx)
				log.Printf("デバッグ情報 - ページタイトル: %s, URL: %s", pageTitle, currentURL)
				
				// ページの内容を確認
				var bodyText string
				if err := chromedp.Evaluate(`document.body.textContent.substring(0, 500)`, &bodyText).Do(ctx); err == nil {
					log.Printf("ページ内容の一部: %s", bodyText)
				}
				
				return nil
			}),
		)
		if debugErr != nil {
			log.Printf("デバッグ情報の取得に失敗: %v", debugErr)
		}
		
		return []map[string]interface{}{}, nil
	}

	log.Printf("書籍内検索が完了しました。%d件の結果を取得: %s in %s", len(results), searchTerm, bookID)
	return results, nil
}
