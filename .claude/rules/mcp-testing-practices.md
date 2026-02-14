---
paths:
  - "**_test.go"
  - "server.go"
  - "main.go"
---

# MCP テスト手法

このルールは、MCP サーバー関連ファイルを編集する際に適用される。

## MCP Standard I/O Mode Testing

**CRITICAL**: すべての機能テストは MCP 標準入出力モードで実行する。スタンドアロン CLI コマンドは使用しない。

### Starting the MCP Server

```bash
# MCP サーバーを stdio モードで起動 (デフォルト)
go run .

# 出力例:
# 2025/06/28 13:10:51 設定を読み込みました
# 2025/06/28 13:10:53 ブラウザクライアントの初期化が完了しました
# 2025/06/28 13:10:54 MCPサーバーを標準入出力で起動します
```

## MCP Protocol Testing

MCP 互換クライアントを使用してテスト:

### 1. Search Content Testing

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "oreilly_search_content",
    "arguments": {
      "query": "Docker containers"
    }
  }
}
```

### 2. Ask Question Testing

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "oreilly_ask_question",
    "arguments": {
      "question": "What career paths are available for software engineers in their late 30s?",
      "max_wait_minutes": 5
    }
  }
}
```

### 3. Resource Access Testing

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "resources/read",
  "params": {
    "uri": "oreilly://book-details/9781098166298"
  }
}
```

## Testing with Claude Code

Claude Code を MCP クライアントとして使用するのが最も簡単:

1. MCP サーバー起動: `go run .`
2. Claude Code でツールとリソースを操作
3. 各種シナリオをテスト:
   - コンテンツ検索: "Search for books about machine learning"
   - 技術 Q&A: "Ask about Python best practices for beginners"
   - リソースアクセス: 書籍詳細と章コンテンツにアクセス

## Header Verification Testing

401 認証エラー解決の検証方法:

### 1. デバッグモード有効化

```bash
ORM_MCP_GO_DEBUG=true go run .
```

### 2. デバッグログの確認

```
API呼び出し先URL: https://learning.oreilly.com/api/v1/miso-answers-relay-service/questions/
送信予定Cookie数: 20
送信Cookie: groot_sessionid=... (Domain: .oreilly.com, Path: /)
```

### 3. 必須ヘッダーの確認

- `Accept: */*`
- `Referer: https://learning.oreilly.com/answers2/`
- `Origin: https://learning.oreilly.com`
- `Sec-Fetch-*` セキュリティヘッダー

## Important Testing Notes

- **スタンドアロン CLI コマンドは実装しない** - すべてのテストは MCP プロトコル経由
- **Cookie 認証** はブラウザクライアントで自動処理
- **デバッグモード** は認証問題のトラブルシューティングに詳細ログを提供
- **すべての API 呼び出し** は実ブラウザリクエストと同じ包括的ヘッダーセットを使用

## テスト実行コマンド

```bash
# 単体テスト実行
task test

# カバレッジ付きテスト
task test:coverage

# 完全 CI ワークフロー (テスト含む)
task ci
```
