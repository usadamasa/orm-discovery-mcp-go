---
paths:
  - "browser/**/*.go"
  - "browser/*.yaml"
---

# Browser Package ガイド

このルールは、browser パッケージ関連ファイルを編集する際に適用される。

## Modular Browser Package Architecture

`browser/` パッケージは4層のモジュラー設計:

### 1. Authentication Layer (`browser/auth.go`)

- Cookie-first 認証戦略と検証
- ログインフロー用 ChromeDP ベースのブラウザ自動化
  - ライフサイクル管理は `.claude/skills/chromedp-lifecycle.md` を参照
- ACM IdP の自動検出と処理

### 2. API Integration Layer (`browser/search.go`, `browser/book.go`)

- 型安全な API 呼び出し用 OpenAPI 生成クライアント
- O'Reilly 内部エンドポイントへの直接 HTTP 通信
- 包括的なレスポンス正規化とエラー処理

### 3. Data Management Layer (`browser/types.go`, `browser/cookie/`)

- すべての API レスポンス用の豊富な型定義
- JSON 永続化によるインターフェースベースの Cookie 管理
- 章と TOC 用の構造化コンテンツモデリング

### 4. Development Support (`browser/debug.go`, `browser/generated/`)

- 環境制御のデバッグ (スクリーンショットキャプチャ)
- API 一貫性のための自動 OpenAPI クライアント生成

## File Organization

| ファイル | 役割 |
|---------|------|
| `auth.go` | Cookie キャッシングと ACM IdP 対応の認証ロジック |
| `search.go` | OpenAPI クライアント使用の検索 API 実装 |
| `book.go` | 書籍操作 (詳細、TOC、章コンテンツ) |
| `types.go` | 型定義とレスポンス構造 |
| `debug.go` | デバッグユーティリティとスクリーンショットキャプチャ |
| `cookie/cookie.go` | Cookie 管理インターフェースと JSON 永続化 |
| `generated/api/` | OpenAPI 生成クライアントコード |

## Browser Requirements

ヘッドレスブラウザ操作に Chrome または Chromium のインストールが必要:

- O'Reilly Learning Platform での認証
- 複雑な認証フロー処理 (ACM IdP リダイレクト含む)

## Important Notes

### Modern Implementation Approach

- コンテンツ取得に O'Reilly 内部 API を直接使用 (高速で信頼性が高い)
- ブラウザ自動化は認証のみに限定 (コンテンツスクレイピングではない)
- OpenAPI 生成クライアントで型安全性と一貫性を提供

### Authentication Requirements

- 有効な O'Reilly Learning Platform 認証情報が必要
- ACM (Association for Computing Machinery) 機関ログインを自動検出・処理
- Cookie キャッシングで繰り返しログインを回避しパフォーマンス向上

### System Dependencies

- 認証ブラウザ自動化に Chrome/Chromium インストールが必要
- ツール依存管理に Aqua パッケージマネージャー
- 標準化されたビルド・開発ワークフローに Task ランナー

## Package Memory Reference

詳細な実装パターンとモジュール固有のガイダンスは `browser/CLAUDE.md` を参照:

- 認証フローパターンと Cookie 管理
- OpenAPI クライアント統合例
- HTML コンテンツパース戦略
- エラー処理とデバッグアプローチ

## Implemented Features

### Cookie Caching

✅ `browser/cookie/cookie.go` で実装済み:

- 設定可能な temp ディレクトリでの JSON 形式ストレージ
- パスワードログインへの自動フォールバック付き Cookie 検証
- セキュアなファイルパーミッション (0600) と有効期限処理
