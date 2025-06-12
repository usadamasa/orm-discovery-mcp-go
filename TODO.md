# OreillyClient ヘッドレスブラウザ移行 TODO

## 概要
現在のOreillyClientはAPIベースとヘッドレスブラウザの両方をサポートしていますが、完全にヘッドレスブラウザ方式に移行するための変更が必要です。

## 現在の状況分析

### 既存の実装
- `oreilly_client.go`: APIベースの実装（HTTP APIを直接呼び出し）
- `browser_client.go`: ヘッドレスブラウザの実装（chromedpを使用）
- `main.go`: 現在はブラウザクライアントを使用してOreillyClientを初期化
- `server.go`: MCPサーバーの実装
- `config.go`: 設定管理（API認証情報とブラウザ認証情報の両方をサポート）

### 現在の問題点
1. OreillyClientがAPIベースのメソッドを持ちながら、BrowserClientに依存している
2. 認証情報の管理が複雑（API用のトークンとブラウザ用の認証情報が混在）
3. エラーハンドリングがAPI前提で設計されている

## 変更が必要な箇所

### 1. oreilly_client.go の大幅な変更 🔴 HIGH PRIORITY

#### 1.1 構造体の変更
- [ ] `OreillyClient`構造体からAPI関連のフィールドを削除
  - `httpClient *http.Client` → 削除
  - `cookieStr string` → 削除  
  - `jwtToken string` → 削除
  - `sessionID string` → 削除
  - `refreshToken string` → 削除
- [ ] `browserClient *BrowserClient`のみを保持する構造に変更

#### 1.2 コンストラクタの変更
- [ ] `NewOreillyClient`関数を削除（API認証情報ベースのコンストラクタ）
- [ ] `NewOreillyClientWithBrowser`を`NewOreillyClient`にリネーム
- [ ] API認証情報を受け取るパラメータを削除

#### 1.3 メソッドの完全な書き換え
- [ ] `Search`メソッドをブラウザベースに変更
  - HTTP APIコールを削除
  - ブラウザでの検索ページ操作に変更
  - DOM操作による検索結果の取得
- [ ] `ListCollections`メソッドをブラウザベースに変更
  - HTTP APIコールを削除
  - ブラウザでのコレクションページ操作に変更
- [ ] `CreateCollection`メソッドをブラウザベースに変更
  - HTTP APIコールを削除
  - ブラウザでのコレクション作成フォーム操作に変更
- [ ] `AddToCollection`メソッドをブラウザベースに変更
- [ ] `RemoveFromCollection`メソッドをブラウザベースに変更
- [ ] `GetCollectionDetails`メソッドをブラウザベースに変更

#### 1.4 ヘルパーメソッドの削除
- [ ] `buildCookieString`メソッドを削除（不要になる）
- [ ] API関連のヘッダー設定ロジックを削除

### 2. browser_client.go の機能拡張 🟡 MEDIUM PRIORITY

#### 2.1 検索機能の実装
- [ ] `SearchContent`メソッドを追加
  - 検索ページへの遷移
  - 検索クエリの入力
  - 検索結果の取得とパース
  - フィルター操作（言語、コンテンツタイプなど）

#### 2.2 コレクション管理機能の実装
- [ ] `ListCollections`メソッドを追加
  - コレクションページへの遷移
  - コレクション一覧の取得
- [ ] `CreateCollection`メソッドを追加
  - コレクション作成ページへの遷移
  - フォーム入力と送信
- [ ] `AddContentToCollection`メソッドを追加
  - コンテンツページでのコレクション追加操作
- [ ] `RemoveContentFromCollection`メソッドを追加
- [ ] `GetCollectionDetails`メソッドを追加
  - 特定のコレクションページへの遷移
  - コレクション詳細情報の取得

#### 2.3 エラーハンドリングの改善
- [ ] ページ読み込みエラーの処理
- [ ] 要素が見つからない場合の処理
- [ ] セッション切れの検出と再ログイン

#### 2.4 パフォーマンス最適化
- [ ] ページ遷移の最適化
- [ ] 不要な待機時間の削減
- [ ] 並行処理の検討

### 3. config.go の変更 🟢 LOW PRIORITY

#### 3.1 設定項目の整理
- [ ] API関連の設定項目を削除
  - `OReillyCookie` → 削除
  - `OReillyJWT` → 削除
  - `SessionID` → 削除
  - `RefreshToken` → 削除
- [ ] ブラウザ関連の設定項目のみ保持
  - `OReillyUserID` → 保持
  - `OReillyPassword` → 保持

#### 3.2 新しい設定項目の追加
- [ ] ブラウザ設定のオプション追加
  - `BROWSER_HEADLESS` → ヘッドレスモードの制御
  - `BROWSER_TIMEOUT` → ブラウザ操作のタイムアウト
  - `BROWSER_WAIT_TIME` → ページ読み込み待機時間

### 4. main.go の変更 🟢 LOW PRIORITY

#### 4.1 初期化ロジックの簡素化
- [ ] API認証情報のチェックを削除
- [ ] ブラウザ認証情報のみのチェックに変更
- [ ] エラーメッセージの更新

### 5. server.go の変更 🟡 MEDIUM PRIORITY

#### 5.1 エラーハンドリングの更新
- [ ] API固有のエラーメッセージをブラウザ操作用に変更
- [ ] ブラウザ操作特有のエラー（要素が見つからない、タイムアウトなど）の処理追加

#### 5.2 レスポンス形式の調整
- [ ] ブラウザから取得したデータの形式に合わせてレスポンス構造を調整
- [ ] API レスポンスとの互換性を保つための変換処理

### 6. 新しいファイルの作成 🟡 MEDIUM PRIORITY

#### 6.1 DOM操作ヘルパー
- [ ] `dom_helpers.go`を作成
  - 共通のDOM操作関数
  - 要素の待機とクリック
  - テキスト入力とフォーム送信
  - データの抽出とパース

#### 6.2 ページオブジェクト
- [ ] `pages/`ディレクトリを作成
  - `search_page.go` → 検索ページの操作
  - `collection_page.go` → コレクションページの操作
  - `content_page.go` → コンテンツページの操作

### 7. テストの更新 🟢 LOW PRIORITY

#### 7.1 既存テストの更新
- [ ] API呼び出しのモックをブラウザ操作のモックに変更
- [ ] テストデータの更新

#### 7.2 新しいテストの追加
- [ ] ブラウザ操作のテスト
- [ ] DOM要素の存在確認テスト
- [ ] エラーケースのテスト

## 実装の優先順位

### Phase 1: 基盤整備
1. `oreilly_client.go`の構造体とコンストラクタの変更
2. `config.go`の設定項目整理
3. `main.go`の初期化ロジック更新

### Phase 2: 検索機能の移行
1. `browser_client.go`に検索機能を実装
2. `oreilly_client.go`の`Search`メソッドをブラウザベースに変更
3. `server.go`のエラーハンドリング更新

### Phase 3: コレクション機能の移行
1. `browser_client.go`にコレクション管理機能を実装
2. `oreilly_client.go`のコレクション関連メソッドをブラウザベースに変更

### Phase 4: 最適化とテスト
1. パフォーマンス最適化
2. エラーハンドリングの改善
3. テストの更新

## 注意事項

### セキュリティ
- [ ] ブラウザでの認証情報の安全な管理
- [ ] セッション管理の改善
- [ ] ログ出力での機密情報の除外

### パフォーマンス
- [ ] ブラウザ起動時間の最適化
- [ ] ページ遷移の効率化
- [ ] メモリ使用量の監視

### 互換性
- [ ] 既存のMCPツールインターフェースとの互換性維持
- [ ] レスポンス形式の一貫性
- [ ] エラーメッセージの統一

## 完了基準

- [ ] すべてのAPI呼び出しがブラウザ操作に置き換えられている
- [ ] 既存の機能がすべてブラウザベースで動作する
- [ ] エラーハンドリングが適切に実装されている
- [ ] パフォーマンスが許容範囲内である
- [ ] テストがすべて通る
- [ ] ドキュメントが更新されている

## 推定工数

- Phase 1: 2-3日
- Phase 2: 3-4日  
- Phase 3: 4-5日
- Phase 4: 2-3日

**総工数: 11-15日**
