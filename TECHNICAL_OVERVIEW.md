# O'Reilly Learning Platform MCP Server - 技術要素概要

## 概要
O'Reilly Learning PlatformのAPIをModel Context Protocol (MCP)経由で利用できるようにしたサーバー実装です。ClineのMCPサーバーとして導入し、O'Reillyのコンテンツ検索とコレクション管理機能を提供します。

## 技術スタック

### 1. プログラミング言語・フレームワーク
- **Go言語**: サーバー実装の主要言語
- **mcp-go**: Model Context Protocolの Go実装ライブラリ
- **net/http**: HTTP クライアント機能
- **encoding/json**: JSON データの処理

### 2. アーキテクチャ

#### MCPサーバー構成
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│     Cline       │◄──►│  MCP Server      │◄──►│ O'Reilly API        │
│   (Claude)      │    │  (Go実装)        │    │ (REST API)          │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
```

#### コンポーネント構成
- **main.go**: エントリーポイント、設定読み込み
- **server.go**: MCPサーバー実装、ツール登録
- **oreilly_client.go**: O'Reilly API クライアント
- **config.go**: 環境変数設定管理

### 3. 認証システム

#### Cookie ベース認証
O'Reilly Learning Platformの認証には以下のCookieキーを使用：

| Cookie Key | 説明 | 重要度 |
|------------|------|--------|
| `orm-jwt` | JWTトークン | 最重要 |
| `groot_sessionid` | セッションID | 重要 |
| `orm-rt` | リフレッシュトークン | 重要 |

#### 認証方式の柔軟性
- **完全Cookie文字列**: ブラウザから取得した全Cookie
- **個別キー設定**: 重要なキーのみを環境変数で設定

### 4. API エンドポイント

#### 検索API
- **URL**: `https://learning.oreilly.com/search/api/search/`
- **メソッド**: GET
- **パラメータ**:
  - `q`: 検索クエリ (必須)
  - `rows`: 結果件数 (デフォルト: 100)
  - `language`: 検索言語 (デフォルト: ["en", "ja"])
  - `tzOffset`: タイムゾーンオフセット (デフォルト: -9)
  - `aia_only`: AI支援コンテンツのみ
  - `feature_flags`: 機能フラグ
  - `report`: レポートデータ含む
  - `isTopics`: トピック検索のみ

#### コレクションAPI
- **URL**: `https://learning.oreilly.com/v3/collections/`
- **メソッド**: GET

### 5. MCPツール実装

#### search_content ツール
```json
{
  "name": "search_content",
  "description": "Search content on O'Reilly Learning Platform",
  "inputSchema": {
    "properties": {
      "query": {"type": "string", "required": true},
      "rows": {"type": "number"},
      "languages": {"type": "array"},
      "tzOffset": {"type": "number"},
      "aia_only": {"type": "boolean"},
      "feature_flags": {"type": "string"},
      "report": {"type": "boolean"},
      "isTopics": {"type": "boolean"}
    }
  }
}
```

#### list_collections ツール
```json
{
  "name": "list_collections",
  "description": "List my collections on O'Reilly Learning Platform",
  "inputSchema": {
    "properties": {}
  }
}
```

### 6. データ構造

#### 検索結果
```go
type SearchResult struct {
    ID          string                 `json:"id"`
    Title       string                 `json:"title"`
    Description string                 `json:"description"`
    URL         string                 `json:"url"`
    WebURL      string                 `json:"web_url"`
    Type        string                 `json:"content_type"`
    Authors     []string               `json:"authors"`
    Publishers  []string               `json:"publishers"`
    Topics      []string               `json:"topics"`
    Language    string                 `json:"language"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

#### コレクション
```go
type Collection struct {
    ID                    string           `json:"id"`
    Name                  string           `json:"name"`
    Description           string           `json:"description"`
    WebURL                string           `json:"web_url"`
    Content               []CollectionItem `json:"content"`
    // その他のメタデータ
}
```

### 7. 設定管理

#### 環境変数
```bash
# 認証情報
OREILLY_JWT="jwt_token_here"
OREILLY_SESSION_ID="session_id_here"
OREILLY_REFRESH_TOKEN="refresh_token_here"

# または完全Cookie
OREILLY_COOKIE="complete_cookie_string"

# サーバー設定
PORT=8080
TRANSPORT=stdio  # または http
DEBUG=false
```

#### Cline MCP設定
```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "disabled": false,
      "timeout": 60,
      "command": "/path/to/orm-discovery-mcp-go",
      "args": [],
      "env": {
        "OREILLY_JWT": "jwt_token",
        "OREILLY_SESSION_ID": "session_id",
        "OREILLY_REFRESH_TOKEN": "refresh_token"
      },
      "transportType": "stdio"
    }
  }
}
```

### 8. エラーハンドリング

#### 認証エラー
- **401 Unauthorized**: JWTトークン期限切れ
- **403 Forbidden**: アクセス権限なし

#### APIエラー
- **404 Not Found**: エンドポイント不存在
- **429 Too Many Requests**: レート制限
- **500 Internal Server Error**: サーバーエラー

### 9. セキュリティ考慮事項

#### 認証情報の管理
- 環境変数での機密情報管理
- JWTトークンの期限管理
- Cookie情報の適切な取り扱い

#### HTTPSリクエスト
- TLS/SSL暗号化通信
- User-Agentヘッダーの設定
- タイムアウト設定

### 10. パフォーマンス最適化

#### HTTPクライアント
- 接続プール利用
- タイムアウト設定 (30秒)
- Keep-Alive接続

#### レスポンス処理
- ストリーミング読み取り
- JSON パース最適化
- メモリ効率的な処理

### 11. 運用・監視

#### ログ出力
- リクエスト/レスポンスログ
- エラーログ
- パフォーマンスメトリクス

#### ヘルスチェック
- API接続確認
- 認証状態確認
- レスポンス時間監視

## 導入効果

### 1. 開発効率向上
- ClineからO'Reillyコンテンツへの直接アクセス
- 技術情報の迅速な検索・参照
- 学習リソースの効率的な管理

### 2. 知識管理
- コレクション機能による体系的な学習管理
- 検索履歴の活用
- 関連コンテンツの発見

### 3. 拡張性
- 新しいO'Reilly API エンドポイントの追加容易性
- 他の学習プラットフォームとの統合可能性
- カスタム機能の実装柔軟性

## 実装・テスト結果

### 1. MCPサーバー導入状況
- ✅ Go言語でのMCPサーバー実装完了
- ✅ ClineのMCP設定ファイルに正常に登録
- ✅ 2つのツール（search_content、list_collections）が利用可能
- ✅ 実行可能ファイルのビルド成功

### 2. 認証システムテスト
- ✅ Cookie認証の重要キー特定完了
  - `orm-jwt`: JWTトークン（最重要）
  - `groot_sessionid`: セッションID
  - `orm-rt`: リフレッシュトークン
- ✅ 柔軟な認証方式実装（完全Cookie vs 個別キー）
- ⚠️ API認証で課題発生

### 3. API接続テスト結果
- **検索API**: 401 Unauthorized エラー
- **コレクションAPI**: 404 Not Found エラー
- **原因分析**: 
  - JWTトークンは有効期限内
  - Cookie情報は最新
  - APIエンドポイントまたは認証方法に追加要件の可能性

### 4. 技術的成果
- ✅ MCP プロトコルの実装
- ✅ Cookie ベース認証システム
- ✅ エラーハンドリング機能
- ✅ 設定管理システム
- ✅ ログ出力機能

## 今後の改善点

### 1. 機能拡張
- ブックマーク機能
- 学習進捗追跡
- レコメンデーション機能

### 2. 認証改善
- 自動トークンリフレッシュ
- OAuth2.0対応
- セッション管理強化

### 3. パフォーマンス
- キャッシュ機能
- 並列処理最適化
- レスポンス圧縮

### 4. 監視・運用
- メトリクス収集
- アラート機能
- ログ分析ツール連携
