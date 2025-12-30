package browser

import (
	"context"
	"net/http"
	"net/url"

	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// MockCookieManager は cookie.Manager インターフェースのモック実装
type MockCookieManager struct {
	cookies          []*http.Cookie
	fileExists       bool
	saveError        error
	loadError        error
	setCookiesError  error
	savedCookiesURLs []*url.URL
}

// NewMockCookieManager は新しいMockCookieManagerを作成します
func NewMockCookieManager() *MockCookieManager {
	return &MockCookieManager{
		cookies:          make([]*http.Cookie, 0),
		fileExists:       false,
		savedCookiesURLs: make([]*url.URL, 0),
	}
}

// Ensure MockCookieManager implements cookie.Manager
var _ cookie.Manager = (*MockCookieManager)(nil)

// SaveCookies はブラウザのCookieをファイルに保存する（モック）
func (m *MockCookieManager) SaveCookies(ctx *context.Context) error {
	if m.saveError != nil {
		return m.saveError
	}
	// 実際には何もしない（モック）
	return nil
}

// LoadCookies はファイルからCookieを読み込んでブラウザに設定する（モック）
func (m *MockCookieManager) LoadCookies(ctx *context.Context) error {
	if m.loadError != nil {
		return m.loadError
	}
	// 実際には何もしない（モック）
	return nil
}

// CookieFileExists はCookieファイルが存在するかどうかをチェックする（モック）
func (m *MockCookieManager) CookieFileExists() bool {
	return m.fileExists
}

// DeleteCookieFile はCookieファイルを削除する（モック）
func (m *MockCookieManager) DeleteCookieFile() error {
	// 実際には何もしない（モック）
	m.fileExists = false
	return nil
}

// GetCookiesForURL は指定されたURLに対して適切なCookieを返す（モック）
func (m *MockCookieManager) GetCookiesForURL(url *url.URL) []*http.Cookie {
	return m.cookies
}

// SetCookies は指定されたURLに対してCookieを設定する（モック）
func (m *MockCookieManager) SetCookies(url *url.URL, cookies []*http.Cookie) error {
	if m.setCookiesError != nil {
		return m.setCookiesError
	}
	m.savedCookiesURLs = append(m.savedCookiesURLs, url)
	m.cookies = cookies
	return nil
}

// WithCookies はテスト用にCookieを設定するヘルパーメソッド
func (m *MockCookieManager) WithCookies(cookies []*http.Cookie) *MockCookieManager {
	m.cookies = cookies
	return m
}

// WithFileExists はテスト用にファイル存在フラグを設定するヘルパーメソッド
func (m *MockCookieManager) WithFileExists(exists bool) *MockCookieManager {
	m.fileExists = exists
	return m
}

// WithSaveError はテスト用にSaveエラーを設定するヘルパーメソッド
func (m *MockCookieManager) WithSaveError(err error) *MockCookieManager {
	m.saveError = err
	return m
}

// WithLoadError はテスト用にLoadエラーを設定するヘルパーメソッド
func (m *MockCookieManager) WithLoadError(err error) *MockCookieManager {
	m.loadError = err
	return m
}

// MockHTTPClient は http.Client の Do メソッドのモック実装
type MockHTTPClient struct {
	response *http.Response
	err      error
	requests []*http.Request
}

// NewMockHTTPClient は新しいMockHTTPClientを作成します
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		requests: make([]*http.Request, 0),
	}
}

// Do は http.Request を実行します（モック）
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	return m.response, m.err
}

// WithResponse はテスト用にレスポンスを設定するヘルパーメソッド
func (m *MockHTTPClient) WithResponse(resp *http.Response) *MockHTTPClient {
	m.response = resp
	return m
}

// WithError はテスト用にエラーを設定するヘルパーメソッド
func (m *MockHTTPClient) WithError(err error) *MockHTTPClient {
	m.err = err
	return m
}

// MockRoundTripper は http.RoundTripper のモック実装
type MockRoundTripper struct {
	response *http.Response
	err      error
}

// NewMockRoundTripper は新しいMockRoundTripperを作成します
func NewMockRoundTripper() *MockRoundTripper {
	return &MockRoundTripper{}
}

// RoundTrip は http.Request を実行します（モック）
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// WithResponse はテスト用にレスポンスを設定するヘルパーメソッド
func (m *MockRoundTripper) WithResponse(resp *http.Response) *MockRoundTripper {
	m.response = resp
	return m
}

// WithError はテスト用にエラーを設定するヘルパーメソッド
func (m *MockRoundTripper) WithError(err error) *MockRoundTripper {
	m.err = err
	return m
}
