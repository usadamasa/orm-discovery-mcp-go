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

### 4. create_collection

O'Reilly Learning Platformで新しいコレクションを作成します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `name` | string | ✅ | - | コレクション名 |
| `description` | string | ❌ | - | コレクションの説明 |
| `privacy_setting` | string | ❌ | "private" | プライバシー設定（private, public, unlisted） |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "create_collection",
    "arguments": {
      "name": "Go言語学習リソース",
      "description": "Go言語を学習するための書籍とリソース集",
      "privacy_setting": "private"
    }
  },
  "id": 4
}'
```

### 5. add_to_collection

既存のコレクションにコンテンツを追加します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `collection_id` | string | ✅ | - | コレクションID |
| `content_id` | string | ✅ | - | 追加するコンテンツのIDまたはOURN |
| `content_type` | string | ❌ | - | コンテンツタイプ（book, video等） |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "add_to_collection",
    "arguments": {
      "collection_id": "12345",
      "content_id": "urn:orm:book:9781492077992",
      "content_type": "book"
    }
  },
  "id": 5
}'
```

### 6. remove_from_collection

コレクションからコンテンツを削除します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `collection_id` | string | ✅ | - | コレクションID |
| `content_id` | string | ✅ | - | 削除するコンテンツのIDまたはOURN |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "remove_from_collection",
    "arguments": {
      "collection_id": "12345",
      "content_id": "urn:orm:book:9781492077992"
    }
  },
  "id": 6
}'
```

### 7. get_collection_details

特定のコレクションの詳細情報を取得します。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `collection_id` | string | ✅ | - | コレクションID |
| `include_content` | boolean | ❌ | true | コンテンツ一覧を含めるかどうか |

#### 使用例

```bash
curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get_collection_details",
    "arguments": {
      "collection_id": "12345",
      "include_content": true
    }
  },
  "id": 7
}'
```

#### レスポンス例

```json
{
  "jsonrpc": "2.0",
  "id": 7,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"collection\":{\"id\":\"12345\",\"name\":\"Go言語学習リソース\",\"description\":\"Go言語を学習するための書籍とリソース集\",\"sharing\":\"private\",\"web_url\":\"https://learning.oreilly.com/collections/12345\",\"created_time\":\"2024-01-01T00:00:00Z\",\"last_modified_time\":\"2024-01-02T00:00:00Z\",\"is_default\":false,\"is_owned\":true,\"is_following\":false,\"owner_display_name\":\"Your Name\",\"follower_count\":0,\"can_be_assigned\":true,\"type\":\"collection\",\"topics\":[\"Programming\",\"Go\"],\"content_count\":5},\"content\":[{\"id\":\"content1\",\"ourn\":\"urn:orm:book:9781492077992\",\"content_type\":\"book\",\"date_added\":\"2024-01-01T00:00:00Z\",\"index\":1.0,\"title\":\"Learning Go\",\"description\":\"An Idiomatic Approach to Real-World Go Programming\"}]}"
      }
    ]
  }
}
```

## 認証

### 環境変数設定

#### 方法1: 個別キー設定（推奨）

```bash
export OREILLY_JWT="your_orm_jwt_token_here"
export OREILLY_SESSION_ID="your_groot_sessionid_here"
export OREILLY_REFRESH_TOKEN="your_orm_rt_token_here"
```

#### 方法2: 完全Cookie文字列

```bash
export OREILLY_COOKIE="your_complete_cookie_string_here"
```

### 必要なCookieキー

| キー | 説明 | 重要度 |
|-----|------|--------|
| `orm-jwt` | JWTトークン | 最重要 |
| `groot_sessionid` | セッションID | 重要 |
| `orm-rt` | リフレッシュトークン | 重要 |

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

- API呼び出し頻度に制限がある可能性があります
- JWTトークンには有効期限があり、定期的な更新が必要です
- 地域によってアクセス可能なコンテンツが異なる場合があります
