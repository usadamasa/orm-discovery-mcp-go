# O'Reilly Learning Platform API Analysis

このドキュメントは、O'Reilly Learning Platform MCP サーバーの実装を通じて解析されたAPIとエンドポイントの詳細な分析結果です。

## 概要

O'Reilly Learning Platform は公開APIを提供していないため、本実装ではブラウザ自動化（ChromeDP）を使用してWebインターフェースを操作し、DOM要素から情報を抽出しています。

## 認証フロー

### ログインプロセス
1. **初期アクセス**: `https://www.oreilly.com/member/login/`
2. **メールアドレス入力**: DOM要素 `input[type="email"]` への入力
3. **Continue ボタンクリック**: JavaScript実行による認証フロー継続
4. **学習プラットフォームアクセス**: `https://learning.oreilly.com/` への自動リダイレクト
5. **セッション確立**: 認証クッキーの取得と保存

### 取得される認証情報
- `_gd_session`: セッション管理クッキー
- `orm-jwt`: JWT認証トークン
- `groot_sessionid`: セッションID
- `orm-rt`: リフレッシュトークン

## 主要エンドポイントとAPI操作

### 1. コンテンツ検索 (`search_content`)

#### エンドポイント
```
https://www.oreilly.com/search/?q={query}&rows={rows}
```

#### パラメータ
- `query` (必須): 検索クエリ文字列
- `rows` (オプション): 結果件数 (デフォルト: 100)
- `languages` (オプション): 対象言語 (デフォルト: ["en", "ja"])
- `tzOffset` (オプション): タイムゾーンオフセット (デフォルト: -9)
- `aia_only` (オプション): AI支援コンテンツのみ (デフォルト: false)
- `feature_flags` (オプション): 機能フラグ (デフォルト: "improveSearchFilters")
- `report` (オプション): レポートデータ含む (デフォルト: true)
- `isTopics` (オプション): トピック検索のみ (デフォルト: false)

#### DOM抽出ロジック
```javascript
// 検索結果リンクの抽出
const linkSelectors = [
    'a[href*="learning.oreilly.com"]',
    'a[href*="/library/view/"]',
    'a[href*="/library/book/"]',
    'a[href*="/videos/"]',
    'a[href*="/book/"]',
    'a[href*="/video"]'
];
```

#### レスポンス構造
```json
{
    "results": [
        {
            "id": "string",
            "title": "string",
            "description": "string",
            "url": "string",
            "content_type": "book|video",
            "authors": ["string"],
            "publisher": "string",
            "source": "browser_search_oreilly_new"
        }
    ],
    "count": "integer",
    "total": "integer"
}
```

### 2. ホームページコレクション取得 (`list_collections`)

#### エンドポイント
```
https://learning.oreilly.com/home/
```

#### DOM抽出ロジック
```javascript
// コレクション要素の検索
const collectionSelectors = [
    '[data-testid*="collection"]',
    '.collection-card',
    '.playlist-card'
];

// タイトル抽出
const titleSelectors = [
    'h2', 'h3', '.title', 
    '[data-testid*="title"]'
];
```

#### レスポンス構造
```json
{
    "collections": [
        {
            "id": "string",
            "title": "string",
            "type": "collection|playlist",
            "source": "homepage"
        }
    ]
}
```

### 3. プレイリスト管理

#### 3.1 プレイリスト一覧取得 (`list_playlists`)
**エンドポイント**: `https://learning.oreilly.com/playlists/`

#### 3.2 プレイリスト作成 (`create_playlist`)
**操作**: DOM要素への直接入力とボタンクリック

#### 3.3 プレイリストへのコンテンツ追加 (`add_to_playlist`)
**操作**: JavaScript実行による動的要素操作

#### 3.4 プレイリスト詳細取得 (`get_playlist_details`)
**エンドポイント**: `https://learning.oreilly.com/playlists/{playlist_id}/`

### 4. 書籍関連操作

#### 4.1 目次抽出 (`extract_table_of_contents`)
**エンドポイント**: `https://learning.oreilly.com/library/view/{book_id}/`

**DOM抽出ロジック**:
```javascript
// 目次要素の検索
const tocSelectors = [
    '.toc', '.table-of-contents',
    '[data-testid*="toc"]',
    '.chapter-list', '.contents'
];
```

#### 4.2 書籍内検索 (`search_in_book`)
**エンドポイント**: `https://learning.oreilly.com/search/?q={query}+inbook:{book_id}`

#### 4.3 複数書籍要約 (`summarize_books`)
**プロセス**: 
1. コンテンツ検索実行
2. 各書籍の詳細情報取得
3. 日本語による要約生成

## JavaScript実行パターン

### 1. DOM要素の動的検索
```javascript
// 要素の存在確認と属性取得
const element = document.querySelector(selector);
if (element) {
    return element.textContent || element.innerText || '';
}
```

### 2. リンク正規化
```javascript
// 相対URLの絶対URL変換
if (url && !url.startsWith('http')) {
    if (url.startsWith('/')) {
        url = 'https://learning.oreilly.com' + url;
    }
}
```

### 3. 重複除去とフィルタリング
```javascript
// タイトルベースの重複除去
const processedTitles = new Set();
const uniqueResults = results.filter(item => {
    if (processedTitles.has(item.title)) {
        return false;
    }
    processedTitles.add(item.title);
    return true;
});
```

## エラーハンドリングとフォールバック

### 1. ログイン失敗時の再試行
- 複数URLでのアクセス試行
- セッション状態の確認
- 認証クッキーの検証

### 2. DOM要素が見つからない場合
- 複数セレクターでの検索
- 待機時間の調整
- デバッグ情報の出力

### 3. ページ読み込み失敗時
- リトライ機能
- タイムアウト設定
- エラーログの詳細記録

## 実行環境要件

### ブラウザ要件
- Chrome または Chromium のインストール
- ヘッドレスモード対応
- JavaScript実行環境

### 環境変数
```bash
OREILLY_USER_ID=your_email@example.com
OREILLY_PASSWORD=your_password
PORT=8080
TRANSPORT=stdio
```

## セキュリティ考慮事項

### 1. 認証情報の管理
- 環境変数での秘匿情報管理
- .envファイルの暗号化推奨
- セッション有効期限の管理

### 2. レート制限対応
- リクエスト間隔の調整
- 同時実行数の制限
- プラットフォーム側制限の遵守

### 3. データプライバシー
- ログ出力での機密情報マスキング
- 一時ファイルの適切な削除
- ネットワーク通信の暗号化

## パフォーマンス最適化

### 1. ブラウザリソース管理
- コンテキストの適切なクリーンアップ
- メモリリークの防止
- CPU負荷の最小化

### 2. キャッシュ戦略
- セッション情報のキャッシュ
- 検索結果の一時保存
- 重複リクエストの回避

### 3. 並行処理
- 複数検索の並行実行
- 非同期処理の活用
- デッドロックの回避

## 将来の拡張可能性

### 1. 新機能対応
- 追加のコンテンツタイプ対応
- 高度な検索フィルター
- ユーザー固有のコンテンツ

### 2. プラットフォーム変更対応
- DOM構造変更への対応
- 新しい認証フローへの対応
- APIエンドポイント変更への対応

### 3. 統合可能性
- 外部システムとの連携
- データエクスポート機能
- レポート生成機能

## 実装上の注意点

1. **DOM構造の変化**: O'Reilly Learning Platform のフロントエンド変更により、CSS セレクターやJavaScriptロジックの調整が必要になる場合があります。

2. **認証フローの複雑性**: ACM (Association for Computing Machinery) 機関ログインなど、複数の認証パスが存在し、それぞれに対応したフローが必要です。

3. **JavaScript実行タイミング**: SPA (Single Page Application) の動的コンテンツ読み込みに対応するため、適切な待機時間の設定が重要です。

4. **コンテンツ抽出の精度**: ブラウザ自動化による情報抽出のため、データの完全性と正確性の確保に継続的な調整が必要です。

このAPIサーバーは、O'Reilly Learning Platform の豊富な技術コンテンツへのプログラマティックなアクセスを可能にし、学習プラットフォームとの効率的な統合を実現します。