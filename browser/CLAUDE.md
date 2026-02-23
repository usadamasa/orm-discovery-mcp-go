# Browser Package Implementation Guide

現在の実装パターンと詳細ガイダンス for the browser package modules.

## Package Structure

```
browser/
├── types.go           # 型定義とレスポンス構造体
├── auth.go            # 認証とセッション管理
├── login.go           # ビジブルブラウザ起動と手動ログイン
├── search.go          # 検索API実装
├── book.go            # 書籍操作とコンテンツ取得
├── answers.go         # O'Reilly Answers AI質問応答実装
├── cookie/cookie.go   # クッキー管理とキャッシング
└── generated/api/     # OpenAPI生成クライアント
```

## Core Components

### 1. Authentication (auth.go)

**Primary Functions**:
- `NewBrowserClient()` - クッキー復元ロジック付きブラウザクライアント作成
- `Reauthenticate()` - ビジブルブラウザによる再認証 (引数なし)
- `validateAuthenticationViaHTTP()` - HTTPリクエストによるCookie有効性検証
- `Close()` - No-op (ブラウザプロセスは login.go で管理)
- `CheckAndResetAuth()` - Cookie有効性検証 + 無効時の削除
- `ReloadCookies()` - Cookieファイル再読み込みと検証

**Login Flow**:
1. Cookie restoration attempt via `LoadCookies()`
2. Authentication validation via `validateAuthenticationViaHTTP()` (HTTP GET to learning.oreilly.com/home/)
3. Fallback to `RunVisibleLogin()` if validation fails
4. `RunVisibleLogin` saves cookies via `cookie.Manager`

**Key Implementation Patterns**:
- **Cookie-first authentication**: Cookie復元 → HTTP検証 → RunVisibleLoginフォールバック
- **No browser state in BrowserClient**: Chrome プロセスは `runVisibleLogin` 内で完結
- **Bilingual logging**: 開発チーム向け日英バイリンガルログ出力

### 2. Visible Login (login.go)

**Primary Functions**:
- `RunVisibleLogin()` - 公開API: Chrome起動 → ログイン待機 → Cookie保存
- `runVisibleLogin()` - 内部実装: Chrome プロセス管理とポーリング
- `FindSystemChrome()` - システム Chrome のパス検索 (macOS / Linux)
- `WaitForCDPWithTimeout()` - CDP WebSocket URL 待機

**Key Implementation Patterns**:
- **exec.Command + NewRemoteAllocator**: Akamai ボット検知を回避
- **processDone channel**: Chrome プロセスの終了を即座に検知
- **processExited flag**: defer での二重 Wait を回避
- **Temporary UserDataDir**: 一時プロファイルで起動、終了後に削除

### 3. Search Implementation (search.go)

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

### 4. O'Reilly Answers Implementation (answers.go)

**Core Functions**:
- `AskQuestion()` - O'Reilly Answers AI経由の質問送信とポーリングによる回答待機
- `GetQuestionByID()` - 質問IDによる保存済み回答の取得
- `createQuestionRequest()` - デフォルトパラメータ付き質問リクエスト作成
- `submitQuestion()` - 生成されたOpenAPIクライアント使用の質問送信
- `pollForAnswer()` - 回答生成完了まで定期的なステータス確認

**Key Implementation Patterns**:
- **OpenAPI client integration**: 生成されたクライアント使用の質問API呼び出し
- **Polling-based answer retrieval**: 回答生成完了まで定期的なステータス確認
- **Comprehensive response structure**: 回答、ソース、関連リソース、フォローアップ質問を含む
- **Timeout management**: 設定可能な最大待機時間と適切なエラーハンドリング

### 5. Book Operations (book.go)

**Core Functions**:
- `GetBookDetails()` - 包括的書籍メタデータのAPI経由取得
- `GetBookTOC()` - 目次構造の取得
- `GetBookChapterContent()` - フル章コンテンツの抽出と解析
- `parseHTMLContent()` - HTMLの構造化コンテンツへの解析

**Key Implementation Patterns**:
- **API-first approach**: メタデータ用生成OpenAPIクライアント使用
- **HTML parsing with golang.org/x/net/html**: ブラウザDOMではなくネイティブGoHTMLパース
- **Structured content representation**: HTMLの構造化セクション、見出し、コードブロックへの変換

### 6. Type Definitions (types.go)

**Core Structures**:
```go
type BrowserClient struct {
    httpClient    HTTPDoer       // HTTP通信を実行するインターフェース (*http.Clientが実装)
    userAgent     string
    cookieManager cookie.Manager
    debug         bool
    stateDir      string         // XDG StateHome (Chrome一時データ、スクリーンショット用)
}
```

**Timeout Constants**:
```go
const (
    ChromeDPExecAllocatorTimeout = 45 * time.Second
    AuthValidationTimeout        = 15 * time.Second
    CookieOperationTimeout       = 10 * time.Second
    WaitVisibleTimeout           = 10 * time.Second
    APIOperationTimeout          = 30 * time.Second
    VisibleLoginTimeout          = 5 * time.Minute  // 手動ログイン待機
)
```

### 7. Cookie Management (cookie/cookie.go)

**Core Functions**:
- `NewCookieManager()` - クッキーマネージャーファクトリー関数
- `SaveCookiesFromData()` - Cookie データのJSONへの永続化
- `LoadCookies()` - ファイルからのCookie復元 (引数なし)
- `CookieFileExists()` / `DeleteCookieFile()` - ファイル管理ユーティリティ
- `GetCookiesForURL()` - URL に対応する Cookie の取得
- `SetCookies()` - URL に対して Cookie を設定

**Key Implementation Patterns**:
- **Interface-based design**: 異なる実装を可能にする`Manager`インターフェース
- **Selective cookie filtering**: 認証関連クッキーのみ保存
- **Security-conscious permissions**: 0600権限でのクッキーファイル保存

## Implementation Patterns

### Cookie-First Authentication Pattern
```go
// 1. Attempt cookie restoration
if cookieManager.CookieFileExists() {
    if err := cookieManager.LoadCookies(); err == nil {
        client.cookieManager = cookieManager
        // 2. Validate restored cookies via HTTP
        if client.validateAuthenticationViaHTTP() == nil {
            return client, nil // Success - cookies are valid
        }
    }
}

// 3. Fallback to visible browser login
client.cookieManager = cookieManager
RunVisibleLogin(filepath.Join(stateDir, "chrome-setup"), cookieManager)
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

## Architecture Decisions

### 1. API-First Design
- **Generated OpenAPI clients** for consistent API interaction
- **Type safety** through generated structs and interfaces
- **Automatic serialization/deserialization** of API responses

### 2. Browser Automation Strategy
- **exec.Command + NewRemoteAllocator** for authentication (Akamai bypass)
- **HTTP APIs for content** - faster and more reliable than DOM scraping
- **Cookie-based session management** - maintain authentication across API calls

### 3. Structured Content Processing
- **Native Go HTML parsing** instead of JavaScript execution
- **Rich content modeling** with separate types for different elements
- **Comprehensive normalization** to handle API response variations

## ChromeDP Lifecycle Management

**For detailed ChromeDP lifecycle management patterns and best practices, see**: `.claude/skills/chromedp-lifecycle/SKILL.md`

### Overview
ChromeDP is only required for initial authentication. All subsequent API calls use HTTP client with cookies. The implementation follows a "close-after-authentication" pattern.

### Key Patterns

#### 1. exec.Command + NewRemoteAllocator
Chrome を `exec.Command` でネイティブ起動し、`chromedp.NewRemoteAllocator` で CDP 接続する。`chromedp.NewExecAllocator` は Akamai にボットとして検知されるため使用しない。

#### 2. Process Exit Detection
`processDone` チャネルで Chrome プロセスの終了を即座に検知する。ユーザーが Chrome を閉じた場合、タイムアウトまで待たずに即座にエラーを返す。

#### 3. Automatic Reauthentication
401/403 エラーを検知 → `Reauthenticate()` で `RunVisibleLogin` を呼び出し → 再ログイン。

### State Management
`BrowserClient.Close()` は no-op。Chrome プロセスは `runVisibleLogin` の defer で自動クリーンアップされるため、`BrowserClient` はブラウザプロセスの状態を保持しない。

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
    slog.Warn("operation failed", "error", err)
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

### Testing New Features
```bash
# Test individual functions
go test ./browser -v -run TestSpecificFunction

# Full CI workflow
task ci

# Debug mode testing
ORM_MCP_GO_DEBUG=true go run .
```

### Cookie Management Guidelines
- **Always validate cookies** before using for API calls
- **Handle cookie expiration** gracefully with fallback to login
- **Use interface-based design** for testability and flexibility
- **Secure file permissions** for cookie storage (0600)

## Configuration

### Environment Variables
- `ORM_MCP_GO_DEBUG=true` - Enable debug logging
- `TRANSPORT=stdio|http` - Transport mode

This implementation guide reflects the current state of the browser package with its exec.Command + NewRemoteAllocator pattern, cookie-first authentication, and processDone-based Chrome exit detection.
