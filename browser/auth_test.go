package browser

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// === Close Tests ===

func TestBrowserClient_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func(t *testing.T) *BrowserClient
	}{
		{
			name: "正常系: 通常のクリーンアップ",
			setupClient: func(t *testing.T) *BrowserClient {
				ctx, ctxCancel := context.WithCancel(t.Context())
				_, allocCancel := context.WithCancel(t.Context())

				return &BrowserClient{
					ctx:         ctx,
					ctxCancel:   ctxCancel,
					allocCancel: allocCancel,
				}
			},
		},
		{
			name: "異常系: nilキャンセル関数でもパニックしない",
			setupClient: func(t *testing.T) *BrowserClient {
				return &BrowserClient{
					ctx:         t.Context(),
					ctxCancel:   nil,
					allocCancel: nil,
				}
			},
		},
		{
			name: "正常系: ctxCancelのみnil",
			setupClient: func(t *testing.T) *BrowserClient {
				allocCtx, allocCancel := context.WithCancel(t.Context())

				return &BrowserClient{
					ctx:         allocCtx,
					ctxCancel:   nil,
					allocCancel: allocCancel,
				}
			},
		},
		{
			name: "正常系: allocCancelのみnil",
			setupClient: func(t *testing.T) *BrowserClient {
				ctx, ctxCancel := context.WithCancel(t.Context())

				return &BrowserClient{
					ctx:         ctx,
					ctxCancel:   ctxCancel,
					allocCancel: nil,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient(t)

			// パニックが発生しないことを確認
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Close() でパニックが発生しました: %v", r)
				}
			}()

			client.Close()

			// Close()が正常に完了したことを確認（パニックしなければOK）
		})
	}
}

// === CreateRequestEditor Tests ===

func TestBrowserClient_CreateRequestEditor(t *testing.T) {
	tests := []struct {
		name            string
		cookies         []*http.Cookie
		expectedHeaders map[string]string
	}{
		{
			name: "正常系: 標準ヘッダー設定",
			cookies: []*http.Cookie{
				{Name: "orm-jwt", Value: "test-token"},
				{Name: "groot_sessionid", Value: "session-123"},
			},
			expectedHeaders: map[string]string{
				"Accept":          "*/*",
				"Accept-Language": "ja,en-US;q=0.7,en;q=0.3",
				"Content-Type":    "application/json",
				"Origin":          "https://learning.oreilly.com",
				"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			},
		},
		{
			name:    "正常系: Cookie未設定",
			cookies: []*http.Cookie{},
			expectedHeaders: map[string]string{
				"Accept":       "*/*",
				"Content-Type": "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCookieManager := NewMockCookieManager().WithCookies(tt.cookies)

			client := &BrowserClient{
				cookieManager: mockCookieManager,
				userAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			}

			editor := client.CreateRequestEditor()

			// テスト用のリクエストを作成
			req, err := http.NewRequest("GET", "https://learning.oreilly.com/api/v1/test", nil)
			if err != nil {
				t.Fatalf("リクエスト作成に失敗: %v", err)
			}

			// RequestEditorを実行
			if err := editor(t.Context(), req); err != nil {
				t.Fatalf("RequestEditor実行に失敗: %v", err)
			}

			// ヘッダーを確認
			for key, expectedValue := range tt.expectedHeaders {
				gotValue := req.Header.Get(key)
				if gotValue != expectedValue {
					t.Errorf("ヘッダー %s = %v, want %v", key, gotValue, expectedValue)
				}
			}

			// Cookieヘッダーを確認
			if len(tt.cookies) > 0 {
				cookieHeader := req.Header.Get("Cookie")
				for _, cookie := range tt.cookies {
					expectedCookieStr := cookie.Name + "=" + cookie.Value
					if !strings.Contains(cookieHeader, expectedCookieStr) {
						t.Errorf("Cookie %s が含まれていません。Cookie header: %s", expectedCookieStr, cookieHeader)
					}
				}
			}
		})
	}
}

func TestBrowserClient_CreateRequestEditorWithReferer(t *testing.T) {
	tests := []struct {
		name            string
		referer         string
		expectedReferer string
	}{
		{
			name:            "正常系: Refererが設定される",
			referer:         "https://learning.oreilly.com/answers2/",
			expectedReferer: "https://learning.oreilly.com/answers2/",
		},
		{
			name:            "正常系: カスタムReferer",
			referer:         "https://example.com/page",
			expectedReferer: "https://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCookieManager := NewMockCookieManager()

			client := &BrowserClient{
				cookieManager: mockCookieManager,
				userAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			}

			editor := client.CreateRequestEditorWithReferer(tt.referer)

			// テスト用のリクエストを作成
			req, err := http.NewRequest("POST", "https://learning.oreilly.com/api/v1/test", nil)
			if err != nil {
				t.Fatalf("リクエスト作成に失敗: %v", err)
			}

			// RequestEditorを実行
			if err := editor(t.Context(), req); err != nil {
				t.Fatalf("RequestEditor実行に失敗: %v", err)
			}

			// Refererヘッダーを確認
			gotReferer := req.Header.Get("Referer")
			if gotReferer != tt.expectedReferer {
				t.Errorf("Referer = %v, want %v", gotReferer, tt.expectedReferer)
			}
		})
	}
}

// === Test Helpers ===

// createMockHTTPResponse はテスト用のHTTPレスポンスを作成します
func createMockHTTPResponse(statusCode int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	for key, value := range headers {
		resp.Header.Set(key, value)
	}
	return resp
}

// createGzipResponse はgzip圧縮されたHTTPレスポンスを作成します
func createGzipResponse(statusCode int, body string) *http.Response {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(body)); err != nil {
		panic(err)
	}
	if err := gw.Close(); err != nil {
		panic(err)
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(&buf),
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
		},
	}
}

// === GetContentFromURL Tests ===

func TestBrowserClient_GetContentFromURL(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		setupHTTPClient func() *MockHTTPClient
		wantError       bool
		errorContains   string
		expectedBody    string
	}{
		{
			name: "正常系: HTMLコンテンツ取得",
			url:  "https://learning.oreilly.com/content.html",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(200, "<html>test</html>", nil),
				)
			},
			wantError:    false,
			expectedBody: "<html>test</html>",
		},
		{
			name: "正常系: XHTMLコンテンツ取得",
			url:  "https://learning.oreilly.com/content.xhtml",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(200, "<?xml version=\"1.0\"?><html>xhtml</html>", nil),
				)
			},
			wantError:    false,
			expectedBody: "<?xml version=\"1.0\"?><html>xhtml</html>",
		},
		{
			name: "正常系: gzip圧縮コンテンツ",
			url:  "https://learning.oreilly.com/content.html",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createGzipResponse(200, "<html>compressed</html>"),
				)
			},
			wantError:    false,
			expectedBody: "<html>compressed</html>",
		},
		{
			name: "異常系: 404エラー",
			url:  "https://learning.oreilly.com/notfound.html",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(404, "Not Found", nil),
				)
			},
			wantError:     true,
			errorContains: "status 404",
		},
		{
			name: "異常系: 500エラー",
			url:  "https://learning.oreilly.com/error.html",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(500, "Internal Server Error", nil),
				)
			},
			wantError:     true,
			errorContains: "status 500",
		},
		{
			name: "異常系: HTTP通信失敗",
			url:  "https://learning.oreilly.com/content.html",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithError(
					io.EOF,
				)
			},
			wantError:     true,
			errorContains: "HTTP request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCookieManager := NewMockCookieManager()
			mockHTTPClient := tt.setupHTTPClient()

			client := &BrowserClient{
				httpClient:    mockHTTPClient,
				cookieManager: mockCookieManager,
				userAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			}

			content, err := client.GetContentFromURL(tt.url)

			if tt.wantError {
				if err == nil {
					t.Errorf("エラーが期待されましたが、エラーが返されませんでした")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("エラーメッセージに %q が含まれていません。エラー: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
				if content != tt.expectedBody {
					t.Errorf("content = %q, want %q", content, tt.expectedBody)
				}
			}
		})
	}
}

// === GzipTransport Tests ===

func TestGzipTransport_RoundTrip(t *testing.T) {
	tests := []struct {
		name                    string
		setupMockTransport      func() *MockRoundTripper
		wantError               bool
		errorContains           string
		expectDecompression     bool
		expectedBody            string
		expectedContentEncoding string
	}{
		{
			name: "正常系: gzip圧縮レスポンスの自動解凍",
			setupMockTransport: func() *MockRoundTripper {
				return NewMockRoundTripper().WithResponse(
					createGzipResponse(200, "<html>compressed content</html>"),
				)
			},
			expectDecompression:     true,
			expectedBody:            "<html>compressed content</html>",
			expectedContentEncoding: "", // Content-Encodingヘッダーは削除される
		},
		{
			name: "正常系: 非圧縮レスポンスはそのまま",
			setupMockTransport: func() *MockRoundTripper {
				return NewMockRoundTripper().WithResponse(
					createMockHTTPResponse(200, "<html>uncompressed</html>", nil),
				)
			},
			expectDecompression: false,
			expectedBody:        "<html>uncompressed</html>",
		},
		{
			name: "正常系: 空のgzipレスポンス",
			setupMockTransport: func() *MockRoundTripper {
				return NewMockRoundTripper().WithResponse(
					createGzipResponse(200, ""),
				)
			},
			expectDecompression:     true,
			expectedBody:            "",
			expectedContentEncoding: "",
		},
		{
			name: "異常系: 基底Transportからのエラー",
			setupMockTransport: func() *MockRoundTripper {
				return NewMockRoundTripper().WithError(io.EOF)
			},
			wantError: true,
		},
		{
			name: "異常系: 無効なgzipデータ",
			setupMockTransport: func() *MockRoundTripper {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("invalid gzip data")),
					Header: http.Header{
						"Content-Encoding": []string{"gzip"},
					},
				}
				return NewMockRoundTripper().WithResponse(resp)
			},
			wantError:     true,
			errorContains: "failed to create gzip reader",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := tt.setupMockTransport()
			gzipTransport := &GzipTransport{
				Transport: mockTransport,
			}

			req, err := http.NewRequest("GET", "https://example.com/test", nil)
			if err != nil {
				t.Fatalf("リクエスト作成に失敗: %v", err)
			}

			resp, err := gzipTransport.RoundTrip(req)

			if tt.wantError {
				if err == nil {
					t.Errorf("エラーが期待されましたが、エラーが返されませんでした")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("エラーメッセージに %q が含まれていません。エラー: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if resp == nil {
				t.Fatal("レスポンスがnilです")
			}

			// Content-Encodingヘッダーの確認
			if tt.expectDecompression {
				gotEncoding := resp.Header.Get("Content-Encoding")
				if gotEncoding != tt.expectedContentEncoding {
					t.Errorf("Content-Encoding = %q, want %q", gotEncoding, tt.expectedContentEncoding)
				}
			}

			// レスポンスボディの読み込みと検証
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("レスポンスボディの読み込みに失敗: %v", err)
			}
			defer func() {
				if cerr := resp.Body.Close(); cerr != nil {
					t.Logf("レスポンスボディのクローズに失敗: %v", cerr)
				}
			}()

			gotBody := string(bodyBytes)
			if gotBody != tt.expectedBody {
				t.Errorf("body = %q, want %q", gotBody, tt.expectedBody)
			}
		})
	}
}

// === NewBrowserClient Tests ===

func TestNewBrowserClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		password      string
		wantError     bool
		errorContains string
	}{
		{
			name:          "異常系: userIDが空文字列",
			userID:        "",
			password:      "password",
			wantError:     true,
			errorContains: "OREILLY_USER_ID and OREILLY_PASSWORD are required",
		},
		{
			name:          "異常系: passwordが空文字列",
			userID:        "test@acm.org",
			password:      "",
			wantError:     true,
			errorContains: "OREILLY_USER_ID and OREILLY_PASSWORD are required",
		},
		{
			name:          "異常系: 両方とも空文字列",
			userID:        "",
			password:      "",
			wantError:     true,
			errorContains: "OREILLY_USER_ID and OREILLY_PASSWORD are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCookieManager := NewMockCookieManager()

			client, err := NewBrowserClient(tt.userID, tt.password, mockCookieManager, false, "/tmp")

			if tt.wantError {
				if err == nil {
					t.Errorf("エラーが期待されましたが、エラーが返されませんでした")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("エラーメッセージに %q が含まれていません。エラー: %v", tt.errorContains, err)
				}
				if client != nil {
					t.Errorf("エラー時はclientがnilであるべきですが、nilではありません")
				}
			} else {
				if err != nil {
					t.Errorf("予期しないエラー: %v", err)
				}
				if client == nil {
					t.Errorf("clientがnilです")
				}
			}
		})
	}
}
