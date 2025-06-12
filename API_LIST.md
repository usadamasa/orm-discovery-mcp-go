# O'Reilly Learning Platform REST API 一覧

## 調査概要
- 調査日時: 2025年6月12日
- 調査方法: curlコマンドを使用してAPIエンドポイントを直接呼び出し
- 認証方式: Bearer Token (JWT)
- ベースURL: https://learning.oreilly.com/api

## 確認されたAPIエンドポイント

### 1. 検索API (Search API)
- **エンドポイント**: `POST /api/v2/search/`
- **ステータス**: ✅ 動作確認済み
- **認証**: 必要 (Bearer Token)
- **概要**: O'Reilly Learning Platformのコンテンツを検索する
- **リクエスト例**:
  ```json
  {
    "query": "golang",
    "limit": 5
  }
  ```
- **レスポンス**: 検索結果のリスト（書籍、動画、記事など）
- **主要フィールド**:
  - `results[]`: 検索結果の配列
  - `facets`: ファセット情報（フォーマット、トピック、出版社など）
  - `total`: 総件数
- **用途**: ホームページの検索機能、コンテンツ発見

### 2. コレクションAPI (Collections API)
- **エンドポイント**: `GET /api/v3/collections/`
- **ステータス**: ❌ 認証エラー (401 Unauthorized)
- **認証**: 必要 (より高い権限が必要と推測)
- **概要**: ユーザーのコレクション一覧を取得
- **エラー**: "Authentication credentials were not provided."
- **用途**: ユーザーの保存したコンテンツ、プレイリスト管理

### 3. ユーザー情報API (User API)
- **エンドポイント**: `GET /api/v2/user/me/`
- **ステータス**: ❌ Not Found (404)
- **概要**: 現在のユーザー情報を取得
- **用途**: ユーザープロフィール、設定情報

### 4. レコメンデーションAPI (Recommendations API)
- **エンドポイント**: `GET /api/v2/recommendations/`
- **ステータス**: ❌ Not Found (404)
- **概要**: ユーザーに対するコンテンツ推薦
- **用途**: ホームページのおすすめコンテンツ表示

## APIの特徴

### 認証方式
- **Bearer Token**: JWTトークンを使用
- **ヘッダー**: `Authorization: Bearer <JWT_TOKEN>`
- **権限レベル**: APIによって異なる権限が必要

### レスポンス形式
- **Content-Type**: `application/json`
- **文字エンコーディング**: UTF-8
- **エラーレスポンス**: 標準的なHTTPステータスコードとJSONメッセージ

### セキュリティ機能
- **HTTPS**: 全通信が暗号化
- **CORS**: Cross-Origin Resource Sharing対応
- **レート制限**: 実装されている可能性が高い
- **Cookie設定**: セキュリティ関連のCookieが設定される

## ホームページで使用される可能性のあるAPI

### 確認済み
1. **検索API** - メイン検索機能
2. **コンテンツ取得API** - 個別コンテンツの詳細情報

### 推測されるAPI（未確認）
1. **ユーザーダッシュボードAPI** - 学習進捗、最近のアクティビティ
2. **トレンドコンテンツAPI** - 人気コンテンツ、新着情報
3. **学習パスAPI** - 学習コース、カリキュラム情報
4. **ブックマークAPI** - 保存したコンテンツ管理
5. **学習履歴API** - 閲覧履歴、学習記録
6. **通知API** - システム通知、アップデート情報

## 技術的詳細

### HTTPヘッダー
- **Server**: istio-envoy (Istio Service Mesh使用)
- **Security Headers**: 
  - `x-frame-options: DENY`
  - `x-content-type-options: nosniff`
  - `strict-transport-security: max-age=31536000; includeSubDomains`
- **Cache Control**: プライベートキャッシュ設定

### インフラストラクチャ
- **CDN**: Akamai使用
- **Load Balancer**: Google Cloud Platform (GCP) ALB
- **Service Mesh**: Istio
- **SSL/TLS**: DigiCert証明書使用

## 制限事項

1. **認証制限**: 一部のAPIは高い権限レベルが必要
2. **エンドポイント発見**: 公開されていないAPIエンドポイントが多数存在する可能性
3. **レート制限**: API呼び出し頻度に制限がある可能性
4. **地域制限**: 地域によってアクセス可能なコンテンツが異なる可能性

## Cookie調査結果

### 発見されたCookie
ブラウザから抽出されたcookieファイル（`.cookie.csv`）から以下の重要なcookieを発見：

1. **`orm-jwt`** - JWTトークン（認証用）
   - 値: `eyJhbGciOiJSUzI1NiIsImtpZCI6ImI1ZjliMGU1YzM1ZDRiY2NjYjY1YzZkOGQxYzQ2MWI5In0...`
   - 有効期限: 2025-07-12T01:12:15.320Z
   - 用途: API認証、ユーザーセッション管理

2. **`groot_sessionid`** - セッションID
   - 値: `3tglm6tb7w29fy3ui7md0u5rjj0ttjum`
   - 有効期限: 2025-07-12T01:12:16.321Z
   - 用途: セッション管理

3. **`orm-rt`** - リフレッシュトークン
   - 値: `e51e871487b64512a2b3caf22061b453`
   - 有効期限: 2025-07-12T01:12:15.321Z
   - 用途: トークンの更新

### ブラウザログイン試行結果
- **ステータス**: ❌ 失敗
- **問題**: 保存されたcookieが期限切れまたは無効
- **現象**: cookieを設定してもサインインページにリダイレクトされる
- **推測**: JWTトークンの有効期限が切れている、またはセッションが無効化されている

### 追加発見されたCookie
- `_abck`, `bm_sz` - ボット検出・セキュリティ関連
- `akaalb_LearningALB` - Akamai Load Balancer用
- `_ga`, `_ga_092EL089CH` - Google Analytics
- `_vwo_*` - VWO（A/Bテスト）関連
- `_vis_opt_*` - Visual Website Optimizer
- `_dd_s` - DataDog監視

## 注意事項

- このAPIリストは限定的な調査に基づいており、O'Reilly Learning Platformの全APIを網羅していません
- APIの仕様は予告なく変更される可能性があります
- 商用利用時は適切なライセンスと利用規約の確認が必要です
- レート制限やセキュリティポリシーを遵守してください
- cookieベースの認証は時間制限があり、定期的な更新が必要です
