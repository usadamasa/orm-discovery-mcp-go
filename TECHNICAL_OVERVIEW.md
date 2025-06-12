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
- **ライブラリ**: mcp-go, chromedp
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
| `config.go` | 環境変数設定管理 |

### MCPツール

| ツール名 | 機能 | 実装状況 |
|---------|------|----------|
| `search_content` | コンテンツ検索（ブラウザベース） | ✅ 実装済み |
| `list_collections` | コレクション一覧（ホームページ） | ✅ 実装済み |
| `list_playlists` | プレイリスト一覧 | ✅ 実装済み |
| `create_playlist` | プレイリスト作成 | ✅ 実装済み |
| `add_to_playlist` | プレイリストへのコンテンツ追加 | ✅ 実装済み |
| `get_playlist_details` | プレイリスト詳細取得 | ✅ 実装済み |
| `summarize_books` | 書籍要約生成 | ✅ 実装済み |
| `extract_table_of_contents` | 書籍目次抽出 | ✅ 実装済み |
| `search_in_book` | 書籍内検索 | ✅ 実装済み |

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

### コレクション

```go
type Collection struct {
    ID          string           `json:"id"`
    Name        string           `json:"name"`
    Description string           `json:"description"`
    WebURL      string           `json:"web_url"`
    Content     []CollectionItem `json:"content"`
}
```

## API エンドポイント

### O'Reilly API

| エンドポイント | メソッド | 用途 |
|---------------|---------|------|
| `/search/api/search/` | GET | コンテンツ検索 |
| `/v3/collections/` | GET | コレクション取得 |

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

### HTTPクライアント最適化

```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### レスポンス処理

- ストリーミング読み取り
- JSON パース最適化
- メモリ効率的な処理

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

# 本番ビルド
go build -o orm-discovery-mcp-go

# クロスコンパイル
GOOS=linux GOARCH=amd64 go build -o orm-discovery-mcp-go-linux
```

### テスト

```bash
# ユニットテスト
go test ./...

# 統合テスト
go test -tags=integration ./...
```

## 今後の拡張

### 機能拡張

- ブックマーク機能
- 学習進捗追跡
- レコメンデーション機能
- キャッシュ機能

### 技術改善

- 自動トークンリフレッシュ
- OAuth2.0対応
- 並列処理最適化
- メトリクス収集
