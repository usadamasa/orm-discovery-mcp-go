# API仕様書

## MCPツール

### search_content

O'Reilly Learning Platformでコンテンツを検索し、書籍、動画、記事の詳細情報を取得します。検索結果にはproduct_idが含まれ、これを使用してMCPリソース経由で詳細情報にアクセスできます。

#### パラメータ

| パラメータ | 型 | 必須 | デフォルト値 | 説明 |
|-----------|---|------|-------------|------|
| `query` | string | ✅ | - | 検索クエリ（技術、フレームワーク、概念、技術的課題など） |
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
      "query": "Docker containers",
      "rows": 50,
      "languages": ["en", "ja"]
    }
  },
  "id": 1
}'
```

#### レスポンス例

検索結果にはproduct_idが含まれ、これを使用してMCPリソース経由で詳細情報にアクセスできます：

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"count\": 10, \"total\": 100, \"results\": [{\"id\": \"9781098131814\", \"title\": \"Docker Deep Dive\", \"description\": \"...\", \"product_id\": \"9781098131814\", \"authors\": [\"Nigel Poulton\"], ...}]}"
      }
    ]
  }
}
```

## MCPリソース

MCPリソースを使用して書籍の詳細情報にアクセスします。リソースURIは`search_content`の結果から取得したproduct_idを使用して構築します。

### 1. oreilly://book-details/{product_id}

書籍の包括的な情報（タイトル、著者、出版日、説明、トピック、完全な目次）を取得します。

#### 使用例

```bash
# MCPクライアント経由でリソースにアクセス
# URI: oreilly://book-details/9781098131814
```

#### レスポンス内容

- 書籍メタデータ（タイトル、著者、出版社、出版日）
- 書籍の説明
- トピックとカテゴリ
- 完全な目次（章の構造とチャプター識別子を含む）

### 2. oreilly://book-toc/{product_id}

書籍の目次のみを詳細に取得します。チャプター名、セクション、ナビゲーション構造を含みます。

#### 使用例

```bash
# MCPクライアント経由でリソースにアクセス
# URI: oreilly://book-toc/9781098131814
```

#### レスポンス内容

- 章とセクションの階層構造
- チャプター識別子（get_book_chapter_contentで使用）
- ナビゲーション情報

### 3. oreilly://book-chapter/{product_id}/{chapter_name}

特定の書籍チャプターの完全なテキストコンテンツを抽出します。

#### 使用例

```bash
# MCPクライアント経由でリソースにアクセス
# URI: oreilly://book-chapter/9781098131814/ch01
```

#### レスポンス内容

- チャプターの見出しとサブ見出し
- 段落とテキストコンテンツ
- コード例とサンプル
- 図表のキャプション
- 構造化された要素

### 利用ワークフロー

1. `search_content`ツールで関心のある技術や概念を検索
2. 検索結果から`product_id`を取得
3. `oreilly://book-details/{product_id}`リソースで書籍詳細と目次を確認
4. `oreilly://book-chapter/{product_id}/{chapter_name}`リソースで必要なチャプターの詳細を取得

### 引用要件

**重要**: リソースから取得したコンテンツを参照する際は、必ず適切に引用してください：

- 書籍タイトルと著者名
- チャプタータイトル（該当する場合）
- 出版社：O'Reilly Media
- O'Reillyの利用規約に従った適切な帰属表示

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
