# オライリーコレクション MCP サーバー バックログ

## 🎯 現在の実装状況

### ✅ 実装済み機能
- [x] コンテンツ検索 (`search_content`) - ブラウザ認証対応
- [x] マイコレクション一覧表示 (`list_collections`) - ブラウザ認証対応 + ホームページ取得
- [x] 書籍要約生成 (`summarize_books`) - ブラウザ認証対応
- [x] ヘッドレスブラウザ認証システム (環境変数からID/パスワード取得)
- [x] ACM IDPリダイレクト対応
- [x] セッション管理とCookie自動取得
- [x] 基本的なMCPサーバー実装

## 📋 機能拡張バックログ

### 🔥 高優先度 (P0)

#### コレクション管理機能
- [x] **コレクション作成** (`create_collection`)
  - 新しいコレクションを作成する機能
  - パラメータ: name, description, privacy_setting
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [x] **コレクションにコンテンツ追加** (`add_to_collection`)
  - 既存のコレクションにコンテンツを追加
  - パラメータ: collection_id, content_id, content_type
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [x] **コレクションからコンテンツ削除** (`remove_from_collection`)
  - コレクションからコンテンツを削除
  - パラメータ: collection_id, content_id
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [x] **コレクション詳細取得** (`get_collection_details`)
  - 特定のコレクションの詳細情報とコンテンツ一覧を取得
  - パラメータ: collection_id, include_content
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### 認証・セキュリティ強化
- [x] **ヘッドレスブラウザ認証**
  - 環境変数からID/パスワードを取得してブラウザでログイン
  - ACM IDPリダイレクト対応
  - 実装ファイル: `browser_client.go`, `main.go`

- [x] **セッション管理**
  - ブラウザセッションの維持とCookie管理
  - 実装ファイル: `browser_client.go`, `oreilly_client.go`

- [ ] **自動トークンリフレッシュ**
  - JWTトークンの自動更新機能
  - 実装ファイル: `oreilly_client.go`, `browser_client.go`

- [ ] **認証状態確認** (`check_auth_status`)
  - 現在の認証状態を確認するツール
  - 実装ファイル: `server.go`, `oreilly_client.go`

### 🔶 中優先度 (P1)

#### 学習進捗管理
- [ ] **学習進捗取得** (`get_learning_progress`)
  - ユーザーの学習進捗を取得
  - パラメータ: content_id, time_range
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **学習進捗更新** (`update_learning_progress`)
  - 学習進捗を更新（ページ数、完了状況など）
  - パラメータ: content_id, progress_data
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **学習統計取得** (`get_learning_stats`)
  - 学習時間、完了した書籍数などの統計情報
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### ブックマーク・お気に入り機能
- [ ] **ブックマーク追加** (`add_bookmark`)
  - コンテンツをブックマークに追加
  - パラメータ: content_id, page_number, note
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **ブックマーク一覧取得** (`list_bookmarks`)
  - ユーザーのブックマーク一覧を取得
  - パラメータ: content_type, sort_by
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **ブックマーク削除** (`remove_bookmark`)
  - ブックマークを削除
  - パラメータ: bookmark_id
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### 高度な検索機能
- [ ] **フィルタ付き検索** (`advanced_search`)
  - より詳細なフィルタリング機能
  - パラメータ: filters (author, publisher, publication_date, difficulty_level)
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **類似コンテンツ検索** (`find_similar_content`)
  - 指定したコンテンツに類似したコンテンツを検索
  - パラメータ: content_id, similarity_threshold
  - 実装ファイル: `server.go`, `oreilly_client.go`

### 🔷 低優先度 (P2)

#### レコメンデーション機能
- [ ] **パーソナライズドレコメンデーション** (`get_recommendations`)
  - ユーザーの学習履歴に基づく推奨コンテンツ
  - パラメータ: recommendation_type, limit
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **トレンドコンテンツ取得** (`get_trending_content`)
  - 人気・トレンドのコンテンツを取得
  - パラメータ: time_period, content_type
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### ノート・メモ機能
- [ ] **ノート作成** (`create_note`)
  - コンテンツに対するノートを作成
  - パラメータ: content_id, page_number, note_text, tags
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **ノート一覧取得** (`list_notes`)
  - ユーザーのノート一覧を取得
  - パラメータ: content_id, tag_filter
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **ノート検索** (`search_notes`)
  - ノート内容を検索
  - パラメータ: search_query, content_filter
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### エクスポート・共有機能
- [ ] **コレクションエクスポート** (`export_collection`)
  - コレクションをCSV/JSON形式でエクスポート
  - パラメータ: collection_id, format, include_metadata
  - 実装ファイル: `server.go`, `oreilly_client.go`

- [ ] **学習レポート生成** (`generate_learning_report`)
  - 学習進捗レポートを生成
  - パラメータ: time_period, report_format
  - 実装ファイル: `server.go`, `oreilly_client.go`

### 🔧 技術的改善 (P1-P2)

#### パフォーマンス最適化
- [ ] **レスポンスキャッシュ機能**
  - 検索結果やコレクション情報のキャッシュ
  - 実装ファイル: 新規 `cache.go`

- [ ] **並列処理最適化**
  - 複数のAPI呼び出しの並列実行
  - 実装ファイル: `oreilly_client.go`

- [ ] **ページネーション対応**
  - 大量データの効率的な取得
  - 実装ファイル: `server.go`, `oreilly_client.go`

#### エラーハンドリング強化
- [ ] **リトライ機能**
  - API呼び出し失敗時の自動リトライ
  - 実装ファイル: `oreilly_client.go`

- [ ] **詳細エラーレスポンス**
  - より詳細なエラー情報の提供
  - 実装ファイル: `server.go`

#### ログ・監視機能
- [ ] **構造化ログ出力**
  - JSON形式での詳細ログ出力
  - 実装ファイル: 新規 `logger.go`

- [ ] **メトリクス収集**
  - API呼び出し回数、レスポンス時間などの収集
  - 実装ファイル: 新規 `metrics.go`

#### 設定管理改善
- [ ] **設定ファイル対応**
  - YAML/JSON設定ファイルのサポート
  - 実装ファイル: `config.go`

- [ ] **環境別設定**
  - 開発・本番環境の設定分離
  - 実装ファイル: `config.go`

## 🚀 実装ロードマップ

### ✅ フェーズ 0: ブラウザ認証システム (完了)
1. ✅ ヘッドレスブラウザを使用した認証システム
2. ✅ 環境変数からの認証情報取得
3. ✅ ACM IDPリダイレクト対応
4. ✅ セッション管理とCookie自動取得
5. ✅ ホームページからのコレクション情報取得

### フェーズ 1: コア機能拡張 (1-2週間)
1. ✅ コレクション管理機能の実装
2. 自動トークンリフレッシュ機能
3. 認証状態確認機能

### フェーズ 2: 学習支援機能 (2-3週間)
1. 学習進捗管理機能
2. ブックマーク機能
3. 高度な検索機能

### フェーズ 3: 高度な機能 (3-4週間)
1. レコメンデーション機能
2. ノート・メモ機能
3. エクスポート・共有機能

### フェーズ 4: 技術的改善 (継続的)
1. パフォーマンス最適化
2. エラーハンドリング強化
3. ログ・監視機能

## 📝 実装時の注意事項

### ブラウザ認証システム
- 環境変数 `OREILLY_USER_ID` と `OREILLY_PASSWORD` が必要
- ACM IDPリダイレクトの自動処理
- ヘッドレスブラウザのリソース管理（プロセス終了時のクリーンアップ）

### API制限への対応
- O'Reilly APIのレート制限を考慮した実装
- 適切な間隔でのAPI呼び出し
- エラー時のバックオフ戦略

### セキュリティ考慮事項
- 認証情報の安全な管理（環境変数使用）
- HTTPS通信の強制
- 入力値の適切なバリデーション
- ブラウザセッションの適切な管理

### ユーザビリティ
- 日本語での適切なエラーメッセージ
- 直感的なパラメータ名
- 豊富な使用例の提供

### テスト戦略
- ユニットテストの充実
- 統合テストの実装
- モックを使用したテスト環境構築

## 🔄 継続的改善

### ユーザーフィードバック収集
- 機能使用状況の分析
- ユーザーからの要望収集
- 定期的な機能見直し

### 技術的負債管理
- コードリファクタリング
- 依存関係の更新
- セキュリティパッチの適用
