package browser

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === Close Tests ===

func TestBrowserClient_Close(t *testing.T) {
	client := &BrowserClient{}
	assert.NotPanics(t, func() { client.Close() })
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

			req, err := http.NewRequest("GET", "https://learning.oreilly.com/api/v1/test", nil)
			require.NoError(t, err)

			err = editor(t.Context(), req)
			require.NoError(t, err)

			for key, expectedValue := range tt.expectedHeaders {
				assert.Equal(t, expectedValue, req.Header.Get(key), "ヘッダー %s", key)
			}

			if len(tt.cookies) > 0 {
				cookieHeader := req.Header.Get("Cookie")
				for _, cookie := range tt.cookies {
					assert.Contains(t, cookieHeader, cookie.Name+"="+cookie.Value)
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

			req, err := http.NewRequest("POST", "https://learning.oreilly.com/api/v1/test", nil)
			require.NoError(t, err)

			err = editor(t.Context(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedReferer, req.Header.Get("Referer"))
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
				return NewMockHTTPClient().WithError(io.EOF)
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
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, content)
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
			expectedContentEncoding: "",
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
			require.NoError(t, err)

			resp, err := gzipTransport.RoundTrip(req)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.expectDecompression {
				assert.Equal(t, tt.expectedContentEncoding, resp.Header.Get("Content-Encoding"))
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, tt.expectedBody, string(bodyBytes))
		})
	}
}

// === validateAuthenticationViaHTTP Tests ===

func TestBrowserClient_ValidateAuthenticationViaHTTP(t *testing.T) {
	tests := []struct {
		name                string
		setupHTTPClient     func() *MockHTTPClient
		cookies             []*http.Cookie
		wantErr             bool
		wantUnauthenticated bool
	}{
		{
			name: "正常系: 200レスポンスで認証成功",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(200, "<html>home</html>", nil),
				)
			},
			cookies: []*http.Cookie{
				{Name: "orm-jwt", Value: "valid-token", Domain: ".oreilly.com"},
				{Name: "groot_sessionid", Value: "session-123", Domain: ".oreilly.com"},
			},
			wantErr: false,
		},
		{
			name: "異常系: 401レスポンスで認証失敗 (errUnauthenticated)",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(401, "Unauthorized", nil),
				)
			},
			cookies: []*http.Cookie{
				{Name: "orm-jwt", Value: "expired-token", Domain: ".oreilly.com"},
			},
			wantErr:             true,
			wantUnauthenticated: true,
		},
		{
			name: "異常系: 403レスポンスで認証失敗 (errUnauthenticated)",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(403, "Forbidden", nil),
				)
			},
			cookies: []*http.Cookie{
				{Name: "orm-jwt", Value: "invalid-token", Domain: ".oreilly.com"},
			},
			wantErr:             true,
			wantUnauthenticated: true,
		},
		{
			name: "異常系: HTTPリクエストエラーで認証失敗 (ネットワークエラー)",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithError(io.EOF)
			},
			cookies:             []*http.Cookie{},
			wantErr:             true,
			wantUnauthenticated: false,
		},
		{
			name: "異常系: 500レスポンスで認証失敗 (予期しないステータス)",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(500, "Internal Server Error", nil),
				)
			},
			cookies:             []*http.Cookie{},
			wantErr:             true,
			wantUnauthenticated: false,
		},
		{
			name: "正常系: リダイレクト（302）は認証失敗として扱う",
			setupHTTPClient: func() *MockHTTPClient {
				return NewMockHTTPClient().WithResponse(
					createMockHTTPResponse(302, "", map[string]string{"Location": "https://www.oreilly.com/member/login/"}),
				)
			},
			cookies:             []*http.Cookie{},
			wantErr:             true,
			wantUnauthenticated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCookieManager := NewMockCookieManager().WithCookies(tt.cookies)
			mockHTTPClient := tt.setupHTTPClient()

			client := &BrowserClient{
				httpClient:    mockHTTPClient,
				cookieManager: mockCookieManager,
				userAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			}

			err := client.validateAuthenticationViaHTTP()

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantUnauthenticated {
					assert.ErrorIs(t, err, errUnauthenticated)
				} else {
					assert.NotErrorIs(t, err, errUnauthenticated)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
