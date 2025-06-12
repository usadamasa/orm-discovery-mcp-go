# API仕様書

## MCPツール一覧

### 1. search_content

O'Reilly Learning Platformでコンテンツを検索します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `query` | string | ✅ | - | 検索クエリ |
| `rows` | number | ❌ | 100 | 返す結果数 |
| `languages` | array | ❌ | ["en", "ja"] | 検索言語 |
| `tzOffset` | number | ❌ | -9 | タイムゾーンオフセット（JST） |
| `aia_only` | boolean | ❌ | false | AI支援コンテンツのみ検索 |
| `feature_flags` | string | ❌ | "improveSearchFilters" | 機能フラグ |
| `report` | boolean | ❌ | true | レポートデータを含める |
| `isTopics` | boolean | ❌ | false | トピックのみ検索 |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": {
      "query": "GraphQL",
      "rows": 50,
      "languages": ["en", "ja"]
    }
  },
  "id": 1
}'
```

#### レスポンス例

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "検索結果のテキスト形式データ"
      }
    ]
  }
}
```

### 2. list_collections

O'Reilly Learning Platformのマイコレクションを一覧表示します。

#### パラメータ

パラメータはありません。

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "list_collections",
    "arguments": {}
  },
  "id": 2
}'
```

### 3. summarize_books

検索結果から複数の書籍を取得し、日本語でまとめて表示します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `query` | string | ✅ | - | 書籍検索クエリ |
| `max_books` | number | ❌ | 5 | まとめる書籍の最大数 |
| `languages` | array | ❌ | ["en", "ja"] | 検索言語 |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "summarize_books",
    "arguments": {
      "query": "Go programming",
      "max_books": 5,
      "languages": ["en", "ja"]
    }
  },
  "id": 3
}'
```

#### 特徴

- 書籍のみをフィルタリング
- 著者、出版社、トピック、言語などの詳細情報を表示
- 統計情報と学習推奨事項を含む日本語のまとめを生成
- Markdownフォーマットで読みやすく整理

### 4. list_playlists

O'Reilly Learning Platformのプレイリストを一覧表示します。

#### パラメータ

パラメータはありません。

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "list_playlists",
    "arguments": {}
  },
  "id": 4
}'
```

### 5. create_playlist

O'Reilly Learning Platformで新しいプレイリストを作成します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `name` | string | ✅ | - | プレイリスト名 |
| `description` | string | ❌ | - | プレイリストの説明 |
| `is_public` | boolean | ❌ | false | 公開設定 |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "create_playlist",
    "arguments": {
      "name": "Go言語学習プレイリスト",
      "description": "Go言語を学習するための動画とリソース集",
      "is_public": false
    }
  },
  "id": 5
}'
```

### 6. add_to_playlist

既存のプレイリストにコンテンツを追加します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `playlist_id` | string | ✅ | - | プレイリストID |
| `content_id` | string | ✅ | - | 追加するコンテンツのIDまたはOURN |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "add_to_playlist",
    "arguments": {
      "playlist_id": "12345",
      "content_id": "urn:orm:video:9781492077992"
    }
  },
  "id": 6
}'
```

### 7. get_playlist_details

特定のプレイリストの詳細情報を取得します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `playlist_id` | string | ✅ | - | プレイリストID |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get_playlist_details",
    "arguments": {
      "playlist_id": "12345"
    }
  },
  "id": 7
}'
```

### 8. extract_table_of_contents

O'Reilly書籍の目次を抽出します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `url` | string | ✅ | - | O'Reilly書籍のURL |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "extract_table_of_contents",
    "arguments": {
      "url": "https://learning.oreilly.com/library/view/docker-deep-dive/9781806024032/chap04.xhtml"
    }
  },
  "id": 8
}'
```

### 9. search_in_book

特定のO'Reilly書籍内で用語を検索します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `book_id` | string | ✅ | - | 書籍IDまたはISBN |
| `search_term` | string | ✅ | - | 検索する用語またはフレーズ |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "search_in_book",
    "arguments": {
      "book_id": "9784814400607",
      "search_term": "アーキテクチャ"
    }
  },
  "id": 9
}'
```

## 認証

### 環境変数設定（ブラウザベース認証）

```bash
export OREILLY_USER_ID="your_email@acm.org"
export OREILLY_PASSWORD="your_password"
```

### 認証方式

このサーバーはヘッドレスブラウザを使用して自動的にO'Reillyにログインします：

1. **自動ログイン**: 環境変数のユーザーID/パスワードでログイン
2. **ACM対応**: ACM IDPリダイレクトを自動処理
3. **セッション管理**: ログイン後のCookieを自動取得・管理

### 必要な認証情報

| 変数名 | 説明 | 例 |
|-------|------|---|
| `OREILLY_USER_ID` | O'Reillyのメールアドレス | your_email@acm.org |
| `OREILLY_PASSWORD` | O'Reillyのパスワード | your_password |

## エラーレスポンス

### 認証エラー

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32000,
    "message": "Authentication failed"
  }
}
```

### パラメータエラー

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid params: query is required"
  }
}
```

## サーバー設定

### 環境変数

| 変数名 | デフォルト値 | 説明 |
|-------|-------------|------|
| `PORT` | 8080 | HTTPサーバーのポート番号 |
| `TRANSPORT` | stdio | 通信方式（stdio/http） |
| `DEBUG` | false | デバッグモード |

### 起動方法

```bash
# 開発環境
go run .

# 本番環境
go build -o orm-discovery-mcp-go
./orm-discovery-mcp-go
```

## 制限事項

- ブラウザ操作のため、通常のAPI呼び出しよりも処理時間が長くなる場合があります
- ヘッドレスブラウザ（Chrome）が必要です
- ログインセッションには有効期限があり、長時間使用しない場合は再ログインが必要です
- 地域によってアクセス可能なコンテンツが異なる場合があります
- O'Reillyのページ構造変更により、一部の機能が影響を受ける可能性があります
