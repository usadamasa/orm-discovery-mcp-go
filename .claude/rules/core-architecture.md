---
# paths を指定しない - 常にコンテキストに含める
---

# O'Reilly Learning Platform MCP Server - コアアーキテクチャ

## プロジェクト概要

O'Reilly Learning Platform へのプログラマティックアクセスを提供する MCP (Model Context Protocol) サーバー。モダンなブラウザ自動化と API 統合を活用。

## Core Components

### 1. Browser Package (`browser/`)

モジュラーなブラウザ自動化と API クライアント:

| ファイル | 役割 |
|---------|------|
| `auth.go` | ChromeDP ベースの認証 (ACM IdP 対応) |
| `search.go` | O'Reilly 内部 API への HTTP 呼び出し |
| `book.go` | 書籍メタデータとコンテンツ取得 |
| `types.go` | API レスポンスの型定義 |
| `cookie/cookie.go` | Cookie 管理と永続化 |
| `generated/api/` | OpenAPI 生成クライアント |

### 2. MCP Server (`server.go`)

- JSON-RPC リクエスト/レスポンス処理
- stdio / HTTP トランスポートモード対応

### 3. Config (`config.go`)

- `.env` ファイルと環境変数からの設定読み込み
- 実行可能ファイル相対パスでの .env 検出

## Key Design Patterns

### API-First Architecture

- O'Reilly 内部 API を直接利用 (DOM スクレイピングではなく)
- OpenAPI 生成クライアントで型安全性を確保
- ブラウザ自動化は認証のみに限定

### Cookie-Based Session Management

- JWT トークン (`orm-jwt`)、セッション ID (`groot_sessionid`)、リフレッシュトークン (`orm-rt`)
- ローカル Cookie キャッシングと有効期限管理
- Cookie 期限切れ時のパスワードログインへの自動フォールバック

### Structured Content Processing

- `golang.org/x/net/html` による HTML パース
- API レスポンスのフィールド正規化
- 章・TOC 用の構造化コンテンツモデリング

## MCP Tools

| Tool | Description | Mode |
|------|-------------|------|
| `oreilly_search_content` | コンテンツ検索 - 書籍/動画/記事のリスト取得 | BFS/DFS |
| `oreilly_ask_question` | O'Reilly Answers AI での技術 Q&A | - |

### oreilly_search_content 探索モード

| Mode | 説明 | レスポンスサイズ | 用途 |
|------|------|----------------|------|
| `bfs` (default) | 軽量結果 (id, title, authors のみ) | ~2-5KB | 高速発見、コンテキスト節約 |
| `dfs` | 完全な詳細結果 | ~50-100KB | 詳細分析 |

## MCP Resources

| URI Pattern | Description |
|-------------|-------------|
| `oreilly://book-details/{product_id}` | 書籍情報 (タイトル、著者、目次) |
| `oreilly://book-toc/{product_id}` | 詳細目次 |
| `oreilly://book-chapter/{product_id}/{chapter_name}` | 章コンテンツ全文 |
| `oreilly://answer/{question_id}` | Q&A 回答 |
| `orm-mcp://history/recent` | 最近の調査履歴 20件 |
| `orm-mcp://history/search{?keyword,type}` | 履歴検索 |
| `orm-mcp://history/{id}` | 特定履歴エントリ |
| `orm-mcp://history/{id}/full` | 完全 API レスポンス |

## MCP Resource Templates

- `oreilly://book-details/{product_id}`
- `oreilly://book-toc/{product_id}`
- `oreilly://book-chapter/{product_id}/{chapter_name}`
- `oreilly://answer/{question_id}`

## MCP Prompts

| Prompt | Title | Arguments |
|--------|-------|-----------|
| `learn-technology` | Learn a Technology | `technology` (required) |
| `research-topic` | Research a Topic | `topic` (required) |
| `debug-error` | Debug an Error | `error_message` (required) |
| `review-history` | Review Research History | None |
| `continue-research` | Continue Research | `topic` (required) |
| `summarize-history` | Summarize Research History | `history_id` (required), `focus` (optional) |

## Usage Workflows

### Content Discovery and Access

1. `oreilly_search_content` ツールでコンテンツを発見
2. 検索結果から `product_id` を抽出
3. `oreilly://book-details/{product_id}` で書籍詳細にアクセス
4. `oreilly://book-chapter/{product_id}/{chapter_name}` で章コンテンツにアクセス

### Natural Language Q&A

1. `oreilly_ask_question` ツールで技術的質問を送信
2. AI 生成回答、引用、関連リソースを受信
3. `oreilly://answer/{question_id}` で保存済み回答にアクセス可能

### Research History

1. すべての `oreilly_search_content` と `oreilly_ask_question` 呼び出しは自動記録
2. `orm-mcp://history/recent` で最近の履歴にアクセス
3. `orm-mcp://history/search?keyword=xxx` でキーワード検索

## Citation Requirements

**重要**: これらのリソースを通じてアクセスしたコンテンツは適切に引用すること:

- 書籍タイトルと著者
- 章タイトル (該当する場合)
- 出版社: O'Reilly Media
- O'Reilly 利用規約に従った適切な帰属表示
