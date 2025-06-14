# 技術概要

## アーキテクチャ

### システム構成

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│     Cline       │◄──►│  MCP Server      │◄──►│ O'Reilly Platform   │
│   (Claude)      │    │  (Go実装)        │    │ (ブラウザベース)    │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │ ヘッドレスブラウザ │
                        │   (Chrome)       │
                        └──────────────────┘
```

### 技術スタック

- **言語**: Go 1.21+
- **プロトコル**: Model Context Protocol (MCP)
- **ライブラリ**: mcp-go, chromedp, oapi-codegen
- **API仕様**: OpenAPI 3.0.3 (コード生成)
- **認証**: ブラウザベース認証（自動ログイン）
- **通信**: JSON-RPC 2.0, WebDriver Protocol

## コンポーネント

### ファイル構成

| ファイル | 役割 |
|---------|------|
| `main.go` | エントリーポイント、設定読み込み |
| `server.go` | MCPサーバー実装、ツール登録 |
| `oreilly_client.go` | O'Reillyクライアント（ブラウザベース） |
| `browser_client.go` | ヘッドレスブラウザクライアント |
| `browser/book.go` | 書籍詳細・目次取得（OpenAPI v3使用） |
| `browser/types.go` | 型定義とAPIエンドポイント設定 |
| `browser/openapi.yaml` | OpenAPI 3.0.3 仕様書 |
| `browser/generated/api/` | OpenAPI生成クライアントコード |
| `config.go` | 環境変数設定管理 |

### MCPツール

| ツール名 | 機能 | 実装状況 |
|---------|------|----------|
| `search_content` | コンテンツ検索（API v2使用） | ✅ 実装済み |
| `get_book_details` | 書籍詳細取得（OpenAPI v3使用） | ✅ 実装済み |
| `get_book_toc` | 書籍目次取得（Flat TOC API使用） | ✅ 実装済み |

## 認証システム

### ブラウザベース認証

O'Reilly Learning Platformへの認証はヘッドレスブラウザを使用：

```go
type BrowserClient struct {
    ctx        context.Context
    cancel     context.CancelFunc
    httpClient *http.Client
    cookies    []*http.Cookie
    userAgent  string
}
```

### 認証フロー

1. ヘッドレスブラウザでO'Reillyログインページにアクセス
2. 環境変数のユーザーID/パスワードで自動ログイン
3. ACM IDPリダイレクトを自動処理
4. ログイン後のCookieを自動取得・保存
5. 以降のリクエストでCookieを使用

## データ構造

### 書籍詳細レスポンス

```go
type BookDetailResponse struct {
    ID            string                 `json:"id"`
    URL           string                 `json:"url"`
    WebURL        string                 `json:"web_url"`
    Title         string                 `json:"title"`
    Description   string                 `json:"description"`
    Authors       []Author               `json:"authors"`
    Publishers    []Publisher            `json:"publishers"`
    ISBN          string                 `json:"isbn"`
    VirtualPages  int                    `json:"virtual_pages"`
    AverageRating float64                `json:"average_rating"`
    Cover         string                 `json:"cover"`
    Issued        string                 `json:"issued"`
    Topics        []Topics               `json:"topics"`
    Language      string                 `json:"language"`
    Metadata      map[string]interface{} `json:"metadata"`
}
```

### 目次レスポンス

```go
type TableOfContentsResponse struct {
    BookID          string                 `json:"book_id"`
    BookTitle       string                 `json:"book_title"`
    TableOfContents []TableOfContentsItem  `json:"table_of_contents"`
    TotalChapters   int                    `json:"total_chapters"`
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type TableOfContentsItem struct {
    ID       string                 `json:"id"`
    Title    string                 `json:"title"`
    Href     string                 `json:"href"`
    Level    int                    `json:"level"`
    Parent   string                 `json:"parent,omitempty"`
    Children []TableOfContentsItem  `json:"children,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

### 検索結果

```go
type SearchResult struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    URL         string   `json:"url"`
    Type        string   `json:"content_type"`
    Authors     []string `json:"authors"`
    Publishers  []string `json:"publishers"`
    Topics      []string `json:"topics"`
    Language    string   `json:"language"`
}
```

## API エンドポイント

### O'Reilly API (OpenAPI 3.0.3対応)

| エンドポイント | メソッド | 用途 |
|---------------|---------|------|
| `/api/v2/search/` | GET | コンテンツ検索（主要エンドポイント） |
| `/api/v1/book/{bookId}` | GET | 書籍詳細取得 |
| `/api/v1/book/{bookId}/flat-toc/` | GET | 書籍目次取得（フラット形式） |

### MCPサーバー

| エンドポイント | メソッド | 用途 |
|---------------|---------|------|
| `/mcp` | POST | MCP JSON-RPC通信 |

## 設定管理

### 環境変数

```bash
# 認証情報（ブラウザベース）
OREILLY_USER_ID=your_email@acm.org
OREILLY_PASSWORD=your_password

# サーバー設定
PORT=8080
TRANSPORT=stdio
DEBUG=false
```

### Cline設定

```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "command": "/path/to/orm-discovery-mcp-go",
      "args": [],
      "env": {
        "OREILLY_USER_ID": "your_email@acm.org",
        "OREILLY_PASSWORD": "your_password"
      }
    }
  }
}
```

## エラーハンドリング

### 認証エラー

- **401 Unauthorized**: JWTトークン期限切れ
- **403 Forbidden**: アクセス権限なし

### システムエラー

- **404 Not Found**: エンドポイント不存在
- **429 Too Many Requests**: レート制限
- **500 Internal Server Error**: サーバーエラー

## セキュリティ

### 通信セキュリティ

- HTTPS通信の強制
- TLS/SSL証明書検証
- User-Agentヘッダー設定

### 認証情報管理

- 環境変数での機密情報管理
- JWTトークンの期限管理
- Cookie情報の適切な取り扱い

## パフォーマンス

### OpenAPIクライアント実装

```go
// OpenAPI生成クライアントを使用
client, err := api.NewClientWithResponses(APIEndpointBase,
    api.WithHTTPClient(bc.httpClient),
    api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
        // 認証ヘッダーとCookieを設定
        req.Header.Set("Accept", "application/json")
        req.Header.Set("X-Requested-With", "XMLHttpRequest")
        for _, cookie := range bc.cookies {
            req.AddCookie(cookie)
        }
        return nil
    }))
```

### レスポンス処理

- OpenAPI生成型による型安全性
- 自動JSONパース処理
- エラーハンドリングの統一化
- 配列・オブジェクト両形式対応

## 運用

### ログ出力

- リクエスト/レスポンスログ
- エラーログ
- パフォーマンスメトリクス

### ヘルスチェック

- API接続確認
- 認証状態確認
- レスポンス時間監視

## 開発・デプロイ

### ビルド

```bash
# 開発環境
go run .

# Task使用（推奨）
task build

# OpenAPIコード生成
task generate:api:oreilly

# コードフォーマット
task format
```

## 今後の拡張

### 機能拡張

- 書籍内検索機能
- 書籍要約機能
- プレイリスト管理機能
- ブックマーク機能

### 技術改善

- APIエンドポイントの拡張
- より多くのOpenAPI仕様対応
- パフォーマンス最適化
- エラーハンドリングの改善

## OpenAPI実装詳細

### スキーマ管理

- **仕様書**: `browser/openapi.yaml`
- **生成コード**: `browser/generated/api/`
- **設定ファイル**: `browser/oapi-codegen.yaml`

### コード生成プロセス

1. OpenAPI仕様書の更新
2. `task generate:api:oreilly` でGoクライアント生成
3. 型安全なAPIクライアント利用

### APIレスポンス変換

```go
// OpenAPI生成型から内部型への変換
func convertAPIBookDetailToLocal(apiBook *api.BookDetailResponse) *BookDetailResponse {
    // ポインタ型からプリミティブ型への安全な変換
    if apiBook.Title != nil {
        bookDetail.Title = *apiBook.Title
    }
    // ...
}
```
