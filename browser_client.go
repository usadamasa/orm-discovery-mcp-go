package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
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
		
		// 学習プラットフォームに移動
		chromedp.Navigate("https://learning.oreilly.com/home/"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		
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
