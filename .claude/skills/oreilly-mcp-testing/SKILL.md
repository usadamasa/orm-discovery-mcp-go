---
name: oreilly-mcp-testing
description: MCP (Model Context Protocol) サーバーの動作確認手法ガイド。stdio/HTTPモードでのテスト、JSON-RPCリクエスト送信、デバッグモード活用、認証エラーの検証方法を含みます。
---

# MCP動作確認ガイド

このスキルは、orm-discovery-mcp-goサーバーの動作確認とデバッグ手法を提供します。

## 概要

MCPサーバーのテストは、標準入出力(stdio)モードまたはHTTPモードで行います。CLIコマンドは実装されていないため、すべての機能テストはMCPプロトコル経由で実施します。

## クイックスタート

### サーバー起動

```bash
# ビルド
task build

# stdioモード(デフォルト)で起動
./bin/orm-discovery-mcp-go

# デバッグモードで起動(詳細ログ + スクリーンショット)
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go

# HTTPモードで起動
TRANSPORT=http PORT=8080 ./bin/orm-discovery-mcp-go
```

### 環境変数

| 変数 | 説明 | 必須 |
|------|------|------|
| `OREILLY_USER_ID` | O'Reillyアカウントのメールアドレス | ✅ |
| `OREILLY_PASSWORD` | O'Reillyパスワード | ✅ |
| `TRANSPORT` | トランスポートモード: `stdio` または `http` | ❌ (デフォルト: stdio) |
| `PORT` | HTTPサーバーポート | ❌ (デフォルト: 8080) |
| `ORM_MCP_GO_DEBUG` | デバッグモード有効化 | ❌ |
| `ORM_MCP_GO_TMP_DIR` | 一時ディレクトリ(Cookie保存先) | ❌ |

## テスト手法

### 1. Claude Codeからのテスト(推奨)

最も簡単な方法は、Claude CodeをMCPクライアントとして使用することです。

**手順**:
1. MCPサーバーを起動: `./bin/orm-discovery-mcp-go`
2. Claude Codeでツールとリソースにアクセス
3. 各シナリオをテスト:
   - コンテンツ検索: "Search for books about machine learning"
   - Q&A: "Ask about Python best practices for beginners"
   - リソースアクセス: 書籍詳細やチャプター内容を取得

### 2. JSON-RPCリクエストによるテスト

stdioモードのサーバーに対してJSON-RPCリクエストを送信します。

#### search_content ツールのテスト

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": {
      "query": "Docker containers"
    }
  }
}
```

#### ask_question ツールのテスト

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "ask_question",
    "arguments": {
      "question": "What career paths are available for software engineers in their late 30s?",
      "max_wait_minutes": 5
    }
  }
}
```

#### リソースアクセスのテスト

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

### 3. デバッグモードでの検証

認証問題やAPIエラーのトラブルシューティングに使用します。

```bash
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go
```

**出力されるデバッグ情報**:
- API呼び出し先URL
- 送信Cookie数と詳細
- HTTPヘッダー情報
- 認証フロー中のスクリーンショット

**ヘッダー検証例**:
```
API呼び出し先URL: https://learning.oreilly.com/api/v1/miso-answers-relay-service/questions/
送信予定Cookie数: 20
送信Cookie: groot_sessionid=... (Domain: .oreilly.com, Path: /)
```

### 4. ブラウザライフサイクルの検証

ChromeDPが正しく管理されているか確認します。

```bash
# 通常モード - 認証後にブラウザプロセスが終了するか確認
./bin/orm-discovery-mcp-go
ps aux | grep chrome  # chromeプロセスが存在しないこと

# デバッグモード - ブラウザが維持されているか確認
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go
ps aux | grep chrome  # chromeプロセスが存在すること
```

### 5. Cookie認証の検証

Cookie保存先とChrome分離を確認します。

```bash
# Cookie保存先の確認
ls -la /var/tmp/orm-discovery-mcp-go/cookies.json

# Chromeデータディレクトリの確認
ls -la /var/tmp/chrome-user-data

# ユーザーのChromeに影響がないことを確認
ls -la ~/.config/google-chrome/Default/  # 変更なし
```

## 利用可能なMCPエンドポイント

### ツール

| ツール | 説明 |
|--------|------|
| `search_content` | コンテンツ検索 - 書籍/ビデオ/記事を検索 |
| `ask_question` | O'Reilly Answers AIへの質問 |

### リソース

| リソースURI | 説明 |
|------------|------|
| `oreilly://book-details/{product_id}` | 書籍詳細情報 |
| `oreilly://book-toc/{product_id}` | 目次情報 |
| `oreilly://book-chapter/{product_id}/{chapter_name}` | チャプターコンテンツ |
| `oreilly://answer/{question_id}` | 回答の取得 |

## トラブルシューティング

### 401認証エラー

**症状**: APIリクエストが401/403エラーを返す

**確認手順**:
1. デバッグモードでサーバーを起動
2. ログでCookie送信状況を確認
3. 必要なヘッダーが送信されているか確認:
   - `Accept: */*`
   - `Referer: https://learning.oreilly.com/answers2/`
   - `Origin: https://learning.oreilly.com`
   - `Sec-Fetch-*` セキュリティヘッダー

**解決策**:
- Cookieファイルを削除して再認証
- 環境変数の認証情報を確認

### サーバーが応答しない

**症状**: stdioモードでリクエストに応答がない

**確認手順**:
1. 初期化ログが出力されているか確認
2. JSON-RPCリクエストの形式が正しいか確認
3. 改行でリクエストが終了しているか確認

### ブラウザプロセスが残留

**症状**: 終了後もchromeプロセスが残る

**確認手順**:
```bash
ps aux | grep chrome
pkill -f chromium  # 必要に応じて手動終了
```

**解決策**:
- デバッグモードが有効になっていないか確認
- main.goのdefer cleanup が正しく動作しているか確認

## 参考リンク

- [MCP公式ドキュメント](https://modelcontextprotocol.io/)
- [ChromeDPライフサイクル管理](chromedp-lifecycle.md)
- プロジェクトCLAUDE.md: テスト手法の詳細
