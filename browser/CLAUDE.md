# Browser Package Implementation Guide

現在の実装パターンと詳細ガイダンス for the browser package modules.

## Package Structure

```
browser/
├── types.go           # 型定義とレスポンス構造体
├── auth.go            # 認証とセッション管理
├── search.go          # 検索API実装
├── book.go            # 書籍操作とコンテンツ取得
├── answers.go         # O'Reilly Answers AI質問応答実装
├── debug.go           # デバッグユーティリティ
├── cookie/cookie.go   # クッキー管理とキャッシング
└── generated/api/     # OpenAPI生成クライアント
```

## Core Components

### 1. Authentication (auth.go)

**Primary Functions**:
- `NewBrowserClient()` - クッキー復元ロジック付きブラウザクライアント作成
- `login()` - ACM IdP対応のO'Reillyログインフロー処理
- `validateAuthentication()` - 保存されたクッキーの有効性検証
- `Close()` - ブラウザリソースのクリーンアップ

**Login Flow**:
1. Cookie restoration attempt via `LoadCookies()`
2. Authentication validation at `https://learning.oreilly.com/home/`
3. Fallback to password login if validation fails
4. Navigate to `https://www.oreilly.com/member/login/`
5. Handle ACM IdP redirects automatically  
6. Extract and save authentication cookies via `SaveCookies()`

**Key Implementation Patterns**:
- **Cookie-first authentication**: クッキー復元 → 検証 → パスワードログインフォールバック
- **ChromeDP browser automation**: 安定性のための特定フラグ付きヘッドレスChrome
- **Bilingual logging**: 開発チーム向け日英バイリンガルログ出力
- **Timeout management**: 60秒ログインタイムアウト + 包括的エラーハンドリング

### 2. Search Implementation (search.go)

**Core Functions**:
- `SearchContent()` - O'Reilly内部API使用のプライマリ検索インターフェース
- `makeHTTPSearchRequest()` - 生成されたOpenAPIクライアント使用の低レベルHTTP API呼び出し
- `normalizeSearchResult()` - APIレスポンスバリエーションの複雑な正規化

**API Integration**:
```go
// Generated OpenAPI client usage
client := generated.NewClient(APIEndpointBase)
resp, err := client.SearchWithResponse(ctx, &generated.SearchParams{
    Q:           &query,
    Rows:        &rows,
    TzOffset:    &tzOffset,
    // ... other parameters
})
```

**Key Implementation Patterns**:
- **OpenAPI code generation**: `browser/generated/api/`からの生成クライアント使用
- **Robust field normalization**: 複数のAPIレスポンス形式対応
- **Native Go processing**: JavaScriptではなくGoでの結果処理
- **Cookie-based authentication**: APIリクエストに保存クッキー注入

**Result Normalization Strategy**:
```go
// Multiple field extraction attempts
title := raw.Title
if title == "" { title = raw.BookTitle }
if title == "" { title = raw.Name }

// URL normalization
itemURL := raw.WebURL
if itemURL == "" && raw.ProductID != "" {
    itemURL = APIEndpointBase + "/library/view/-/" + raw.ProductID + "/"
}
```

### 3. O'Reilly Answers Implementation (answers.go)

**Core Functions**:
- `AskQuestion()` - O'Reilly Answers AI経由の質問送信とポーリングによる回答待機
- `GetQuestionByID()` - 質問IDによる保存済み回答の取得
- `createQuestionRequest()` - デフォルトパラメータ付き質問リクエスト作成
- `submitQuestion()` - 生成されたOpenAPIクライアント使用の質問送信
- `pollForAnswer()` - 回答生成完了まで定期的なステータス確認

**Question Submission Flow**:
```go
// 3-step question processing
1. Question submission: createQuestionRequest() + submitQuestion()
2. Polling loop: pollForAnswer() with configurable timeout
3. Response parsing: comprehensive answer with sources and metadata
```

**Key Implementation Patterns**:
- **OpenAPI client integration**: 生成されたクライアント使用の質問API呼び出し
- **Polling-based answer retrieval**: 回答生成完了まで定期的なステータス確認
- **Comprehensive response structure**: 回答、ソース、関連リソース、フォローアップ質問を含む
- **Timeout management**: 設定可能な最大待機時間と適切なエラーハンドリング

**Answer Response Structure**:
```go
type AnswerResponse struct {
    QuestionID     string        `json:"question_id"`
    IsFinished     bool          `json:"is_finished"`
    MisoResponse   MisoResponse  `json:"miso_response"`
}

type MisoResponse struct {
    Data AnswerData `json:"data"`
}

type AnswerData struct {
    Answer              string                `json:"answer"`
    Sources             []AnswerSource        `json:"sources"`
    RelatedResources    []RelatedResource     `json:"related_resources"`
    AffiliationProducts []AffiliationProduct  `json:"affiliation_products"`
    FollowupQuestions   []string             `json:"followup_questions"`
}
```

### 4. Book Operations (book.go)

**Core Functions**:
- `GetBookDetails()` - 包括的書籍メタデータのAPI経由取得
- `GetBookTOC()` - 目次構造の取得
- `GetBookChapterContent()` - フル章コンテンツの抽出と解析
- `parseHTMLContent()` - HTMLの構造化コンテンツへの解析

**Content Extraction Flow**:
```go
// 3-step content extraction
1. TOC lookup: GetBookTOC(productID)
2. URL resolution: match chapter name to TOC entries
3. HTML parsing: GetChapterHTMLContent() + parseHTMLContent()
```

**Key Implementation Patterns**:
- **API-first approach**: メタデータ用生成OpenAPIクライアント使用
- **HTML parsing with golang.org/x/net/html**: ブラウザDOMではなくネイティブGoHTMLパース
- **Structured content representation**: HTMLの構造化セクション、見出し、コードブロックへの変換
- **Comprehensive element handling**: `h1-h6`, `p`, `pre`, `code`, `img`, `a`タグ処理

**HTML Content Structure**:
```go
type ParsedChapterContent struct {
    Sections []ContentSection `json:"sections"`
    Metadata map[string]interface{} `json:"metadata"`
}

type ContentSection struct {
    Type     string                 `json:"type"` // "heading", "paragraph", "code", etc.
    Content  string                 `json:"content"`
    Level    int                    `json:"level,omitempty"` // for headings
    Language string                 `json:"language,omitempty"` // for code blocks
    Children []ContentSection       `json:"children,omitempty"`
}
```

### 4. Type Definitions (types.go)

**Core Structures**:
```go
type BrowserClient struct {
    ctx           context.Context
    cancel        context.CancelFunc
    httpClient    *http.Client
    cookies       []*http.Cookie
    cookieManager cookie.Manager
    tmpDir        string
    debug         bool
}

type BookDetailResponse struct {
    ProductID          string            `json:"product_id"`
    Title              string            `json:"title"`
    Authors            []string          `json:"authors"`
    Publisher          string            `json:"publisher"`
    PublishedDate      string            `json:"published_date"`
    Description        string            `json:"description"`
    Topics             []string          `json:"topics"`
    WebURL             string            `json:"web_url"`
    TableOfContents    []TableOfContentsItem `json:"table_of_contents"`
    Metadata           map[string]interface{} `json:"metadata"`
}
```

**Key Implementation Patterns**:
- **Comprehensive JSON tagging**: 全エクスポートフィールドの完全なJSONタグ付け
- **Metadata maps**: 追加データストレージ用の柔軟なメタデータマップ
- **Rich content modeling**: 異なるコンテンツ要素用の分離型定義

### 5. Cookie Management (cookie/cookie.go)

**Core Functions**:
- `NewCookieManager()` - クッキーマネージャーファクトリー関数
- `SaveCookies()` - ブラウザクッキーのJSONへの抽出と永続化
- `LoadCookies()` - ファイルからブラウザコンテキストへのクッキー復元
- `CookieFileExists()` / `DeleteCookieFile()` - ファイル管理ユーティリティ

**Key Implementation Patterns**:
- **Interface-based design**: 異なる実装を可能にする`Manager`インターフェース
- **Selective cookie filtering**: 認証関連クッキーのみ保存
- **Security-conscious permissions**: 0600権限でのクッキーファイル保存
- **Robust expiration logic**: セッションクッキーと期限付きクッキー両方の処理

**Authentication Cookies**:
```go
var authCookieNames = map[string]bool{
    "orm-jwt":         true, // JWT authentication token
    "groot_sessionid": true, // Session ID
    "orm-rt":          true, // Refresh token
    "userid":          true, // User ID
    // ... other auth-related cookies
}
```

### 6. Debug Utilities (debug.go)

**Core Functions**:
- `debugScreenshot()` - デバッグ用条件付きスクリーンショット撮影

**Key Implementation Patterns**:
- **Environment-controlled debugging**: `bc.debug`がtrueの時のみ実行
- **Silent failure**: エラーがメインフローを中断せず、警告ログのみ
- **Descriptive naming**: デバッグコンテキスト用の意味のある名前でスクリーンショット保存

## Implementation Patterns

### Cookie-First Authentication Pattern
```go
// 1. Attempt cookie restoration
if bc.cookieManager.CookieFileExists() {
    if err := bc.cookieManager.LoadCookies(bc.ctx); err == nil {
        // 2. Validate restored cookies
        if err := bc.validateAuthentication(); err == nil {
            return nil // Success - cookies are valid
        }
    }
}

// 3. Fallback to password login
return bc.login()
```

### OpenAPI Client Integration Pattern
```go
// Generated client usage with cookie authentication
client := generated.NewClient(APIEndpointBase)

// Cookie injection for authentication
req.Header.Set("Cookie", cookieHeader)

// API call with parameters
resp, err := client.SearchWithResponse(ctx, &generated.SearchParams{
    Q:    &query,
    Rows: &rows,
})
```

### HTML Content Parsing Pattern
```go
func parseHTMLContent(htmlContent string) (*ParsedChapterContent, error) {
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        return nil, err
    }
    
    var sections []ContentSection
    parseNode(doc, &sections, 0)
    
    return &ParsedChapterContent{
        Sections: sections,
        Metadata: map[string]interface{}{
            "parsing_method": "golang_html_parser",
            "parsed_at":      time.Now().Format(time.RFC3339),
        },
    }, nil
}
```

## Architecture Decisions

### 1. API-First Design
- **Generated OpenAPI clients** for consistent API interaction
- **Type safety** through generated structs and interfaces
- **Automatic serialization/deserialization** of API responses

### 2. Browser Automation Strategy
- **ChromeDP for authentication only** - not for content scraping
- **HTTP APIs for content** - faster and more reliable than DOM scraping
- **Cookie-based session management** - maintain authentication across API calls

### 3. Structured Content Processing
- **Native Go HTML parsing** instead of JavaScript execution
- **Rich content modeling** with separate types for different elements
- **Comprehensive normalization** to handle API response variations

### 4. Performance Optimization
- **Cookie caching** to avoid repeated logins
- **OpenAPI client reuse** for multiple API calls
- **Structured error handling** with proper logging and debugging

## Development Guidelines

### Adding New API Endpoints
1. Update `browser/openapi.yaml` with new endpoint specifications
2. Run `task generate:api:oreilly` to regenerate client code
3. Add wrapper functions in appropriate module (search.go, book.go, etc.)
4. Implement response normalization for consistency

### Error Handling Best Practices
```go
// Always log context for debugging
if err != nil {
    log.Printf("Error in %s: %v", functionName, err)
    if bc.debug {
        bc.debugScreenshot("error-context")
    }
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

### Testing New Features
```bash
# Test individual functions
go test ./browser -v -run TestSpecificFunction

# Integration testing with real API
go run . test "search query"

# Debug mode testing
ORM_MCP_GO_DEBUG=true go run . test "search query"
```

### Cookie Management Guidelines
- **Always validate cookies** before using for API calls
- **Handle cookie expiration** gracefully with fallback to login
- **Use interface-based design** for testability and flexibility
- **Secure file permissions** for cookie storage (0600)

## Configuration

### Environment Variables
- `ORM_MCP_GO_DEBUG=true` - Enable debug logging and screenshots
- `ORM_MCP_GO_TMP_DIR=/path/to/tmp` - Custom temporary directory for cookies and screenshots

### Browser Configuration
```go
// ChromeDP options for stability
opts := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.Flag("headless", true),
    chromedp.Flag("disable-gpu", true),
    chromedp.Flag("no-sandbox", true),
    chromedp.Flag("disable-dev-shm-usage", true),
)
```

## ChromeDP Lifecycle Management

### Overview
ChromeDP is only required for initial authentication. All subsequent API calls use HTTP client with cookies. The implementation follows a "close-after-authentication" pattern to **avoid issues with URL operations in production environments**. As a secondary benefit, this also reduces memory usage.

### Key Patterns

#### 1. Authentication-Only Browser Usage
**Pattern**: Start ChromeDP → Authenticate → Close immediately (unless debug mode)

```go
// NewBrowserClient() - auth.go
func NewBrowserClient(userID, password string, cookieManager cookie.Manager, debug bool, tmpDir string) (*BrowserClient, error) {
    // 1. Start ChromeDP with explicit UserDataDir
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserDataDir(filepath.Join(tmpDir, "chrome-user-data")), // Explicit isolation
        chromedp.Flag("headless", true),
        // ... other flags
    )

    // 2. Authenticate (either via cookies or password login)
    // ... authentication logic ...

    // 3. Close immediately in non-debug mode
    if !debug {
        slog.Info("非デバッグモード: ブラウザコンテキストをクローズします")
        client.Close()
    }

    return client, nil
}
```

**Benefits**:
- **Avoids URL operation issues** in production environments (primary reason)
- Browser process only runs during authentication
- HTTP API calls work without browser
- 100-300MB memory reduction as secondary benefit

#### 2. Debug Mode Persistence
**Pattern**: Keep browser alive in debug mode for screenshot functionality

```go
// Debug mode check before closing
if !debug {
    client.Close()
}
// In debug mode, browser stays alive for debugScreenshot()
```

**Use Case**:
- Development and troubleshooting
- Screenshot capture during authentication flow
- Visual verification of browser state

#### 3. Automatic Reauthentication
**Pattern**: Detect 401/403 errors → Restart browser → Re-login → Close

```go
// ReauthenticateIfNeeded() - auth.go
func (bc *BrowserClient) ReauthenticateIfNeeded(userID, password string) error {
    slog.Info("Cookie有効期限切れ検出: 再認証を開始します")

    // 1. Temporarily restart browser
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserDataDir(filepath.Join(bc.tmpDir, "chrome-user-data")),
        // ... flags ...
    )
    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer allocCancel()

    ctx, ctxCancel := chromedp.NewContext(allocCtx)
    defer ctxCancel()

    // 2. Update browser context temporarily
    bc.ctx = ctx
    bc.ctxCancel = ctxCancel
    bc.allocCancel = allocCancel

    // 3. Re-login
    if err := bc.login(userID, password); err != nil {
        return fmt.Errorf("再認証に失敗しました: %w", err)
    }

    // 4. Save cookies
    bc.cookieManager.SaveCookies(&ctx)
    bc.syncCookiesFromBrowser()

    // 5. Close immediately (non-debug mode)
    if !bc.debug {
        bc.Close()
    }

    return nil
}
```

#### 4. HTTP API Error Handling
**Pattern**: Detect authentication errors → Trigger reauthentication → Retry

```go
// GetContentFromURL() - auth.go
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return "", fmt.Errorf("authentication error: status %d (cookies may have expired)", resp.StatusCode)
}

// server.go - SearchContentHandler
results, err := s.browserClient.SearchContent(requestParams.Query, options)
if err != nil && isAuthError(err) {
    // Automatic reauthentication
    if reauthErr := s.browserClient.ReauthenticateIfNeeded(s.config.OReillyUserID, s.config.OReillyPassword); reauthErr != nil {
        return mcp.NewToolResultError(fmt.Sprintf("再認証に失敗しました: %v", reauthErr)), nil
    }
    // Retry
    results, err = s.browserClient.SearchContent(requestParams.Query, options)
}
```

### State Management

#### Why No isClosed Flag?
**Answer**: Not needed - nil-safe checks are sufficient

```go
// Close() - No state flag required
func (bc *BrowserClient) Close() {
    if bc.ctxCancel != nil {
        bc.ctxCancel()
    }
    if bc.allocCancel != nil {
        bc.allocCancel()
    }
}
```

**Rationale**:
- Go's nil-safe checks prevent double-close issues
- Multiple Close() calls are harmless
- Simpler implementation without additional state

### Chrome Isolation (User Browser Protection)

#### Explicit UserDataDir Setting
**Pattern**: Always specify isolated UserDataDir to prevent interference

```go
chromedp.UserDataDir(filepath.Join(tmpDir, "chrome-user-data"))
```

**Benefits**:
- **Explicit isolation**: No accidental access to user's Chrome profile
- **Unified management**: tmpDir controls both cookies and Chrome data
- **Testability**: Easy to specify different directories for testing
- **Visibility**: Clear in logs where Chrome data is stored

**Default vs Explicit**:
- **Default** (implicit): ChromeDP creates `/tmp/chromedp-*` automatically
- **Explicit** (recommended): `filepath.Join(tmpDir, "chrome-user-data")` for clarity

### Implementation Checklist

When implementing ChromeDP-based features:

- [ ] Use explicit `UserDataDir` in ExecAllocatorOptions
- [ ] Close browser immediately after authentication (non-debug mode)
- [ ] Keep browser alive in debug mode for screenshots
- [ ] Implement 401/403 error detection
- [ ] Add automatic reauthentication with browser restart
- [ ] Use nil-safe checks instead of state flags
- [ ] Test both debug and non-debug modes
- [ ] Verify memory usage before/after browser close

### Memory Impact

Note: Memory reduction is a secondary benefit. The primary reason for closing the browser after authentication is to avoid URL operation issues in production environments.

| Mode | Browser State | Memory Usage | Use Case |
|------|---------------|--------------|----------|
| **Normal (Production)** | Closed after auth | ~10-30MB | Production deployment |
| **Debug** | Always running | ~100-300MB | Development & troubleshooting |
| **Reauthentication** | Temporary restart | Brief spike (~100-300MB) | Cookie expiration handling |

### Testing

#### Verify Browser Lifecycle
```bash
# 1. Non-debug mode - browser should close after auth
task build
./bin/orm-discovery-mcp-go
ps aux | grep chrome  # Should not find chrome process after startup

# 2. Debug mode - browser should stay alive
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go
ps aux | grep chrome  # Should find running chrome process

# 3. Memory comparison
ps aux | grep orm-discovery-mcp-go  # Compare RSS with/without debug mode
```

#### Verify Chrome Isolation
```bash
# Check UserDataDir location
ls -la /var/tmp/chrome-user-data  # ChromeDP data isolated here

# Verify no interference with user's Chrome
ls -la ~/.config/google-chrome/Default/  # Should be unchanged
```

This implementation guide reflects the current state of the browser package with its modern API-first approach, sophisticated cookie management, robust error handling patterns, and optimized ChromeDP lifecycle management.