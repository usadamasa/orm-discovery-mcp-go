# Browser Package Implementation Guide

現在の実装パターンと詳細ガイダンス for the browser package modules.

## Package Structure

```
browser/
├── types.go           # 型定義とレスポンス構造体
├── auth.go            # 認証とセッション管理
├── search.go          # 検索API実装
├── book.go            # 書籍操作とコンテンツ取得
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

### 3. Book Operations (book.go)

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

This implementation guide reflects the current state of the browser package with its modern API-first approach, sophisticated cookie management, and robust error handling patterns.