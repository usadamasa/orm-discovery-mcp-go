# O'Reilly Learning Platform MCP Server

O'Reilly Learning PlatformのコンテンツをModel Context Protocol (MCP)経由で検索・アクセスできるGoサーバーです。

> [!WARNING]
> This project is an unofficial implementation of an MCP server for interacting with the O'Reilly Learning Platform.
> It is **not affiliated with or endorsed by O'Reilly Media, Inc.** in any way.
> This tool is provided **for educational and personal use only**, and may break at any time if the internal APIs or authentication flows used by O'Reilly change.
> Use at your own risk, and please refer to O'Reilly's [Terms of Service](https://www.oreilly.com/terms/) before using this tool.

## クイックスタート

### Claude Code Pluginとしてインストール (推奨)

Claude Codeのプラグインシステムを使って簡単にインストールできます。

```bash
# マーケットプレイスを追加
/plugin marketplace add usadamasa/orm-discovery-mcp-go

# プラグインをインストール
/plugin install orm-discovery-mcp-go

# 環境変数設定 (必須)
export OREILLY_USER_ID="your_email@acm.org"
export OREILLY_PASSWORD="your_password"
```

**注意**: MCP Serverバイナリは別途インストールが必要です。[Releases](https://github.com/usadamasa/orm-discovery-mcp-go/releases)からダウンロードするか、下記の手動ビルドを行ってください。

### 手動インストール

#### 1. ツールのインストール

```bash
# aquaでツールをインストール
aqua install

# ビルド
task build
```

### 2. 認証設定

プロジェクトディレクトリに`.env`ファイルを作成：

```bash
# .env
OREILLY_USER_ID=your_email@acm.org
OREILLY_PASSWORD=your_password
```

### 3. 起動

```bash
# 開発用
go run .

# Claude Code MCP設定
claude mcp add -s user orm-discovery-mcp-go \
  -e OREILLY_USER_ID="your_email@acm.org" \
  -e OREILLY_PASSWORD="your_password" \
  -- /your/path/to/orm-discovery-mcp-go
```

## 機能

### MCPツール
- **`search_content`**: O'Reillyコンテンツの検索（書籍、動画、記事の発見）
- **`ask_question`**: O'Reilly Answers AIへの自然言語での質問

### MCPリソース
- **`oreilly://book-details/{product_id}`**: 書籍詳細情報
- **`oreilly://book-toc/{product_id}`**: 書籍目次
- **`oreilly://book-chapter/{product_id}/{chapter_name}`**: チャプター内容
- **`oreilly://answer/{question_id}`**: AI生成回答の取得
- **`orm-mcp://history/recent`**: 直近20件の調査履歴
- **`orm-mcp://history/search?keyword=xxx`**: キーワードで履歴検索
- **`orm-mcp://history/{id}`**: 特定の調査履歴の詳細

### MCPプロンプト
- **`learn-technology`**: 特定技術の学習パスを生成（例: Kubernetes、React）
- **`research-topic`**: 技術トピックの多角的な調査（例: マイクロサービスアーキテクチャ）
- **`debug-error`**: エラーメッセージのデバッグガイドを生成
- **`review-history`**: 過去の調査履歴をレビューしてパターンや傾向を分析
- **`continue-research`**: 過去の調査を継続して深掘りする

### 利用フロー

#### コンテンツ検索・アクセス
1. `search_content`で検索 → `product_id`取得
2. `book-details`で書籍情報確認
3. `book-chapter`で必要な章を取得

#### AI質問応答
1. `ask_question`で技術的な質問を投稿 → `question_id`取得
2. `oreilly://answer/{question_id}`でAI生成回答を取得

#### プロンプト活用
1. **技術学習**: `learn-technology`で学習したい技術名を指定 → 体系的な学習パスを取得
2. **技術調査**: `research-topic`で調査トピックを指定 → 多角的な調査結果を取得
3. **エラー解決**: `debug-error`でエラーメッセージを指定 → デバッグガイドを取得

#### 調査履歴の活用
1. `orm-mcp://history/recent`で直近の調査履歴を確認
2. `orm-mcp://history/search?keyword=docker`でキーワード検索
3. `review-history`プロンプトで傾向分析
4. `continue-research`プロンプトで過去の調査を深掘り

## ファイル保存先

XDG Base Directory Specificationに準拠しています。

| 用途 | XDG環境変数 | デフォルトパス |
|------|-------------|----------------|
| ログ、Chrome一時データ、スクリーンショット | `$XDG_STATE_HOME` | `~/.local/state/orm-mcp-go/` |
| Cookie | `$XDG_CACHE_HOME` | `~/.cache/orm-mcp-go/` |
| 調査履歴 | `$XDG_DATA_HOME` | `~/.local/share/orm-mcp-go/research_history.json` |
| 将来の設定ファイル | `$XDG_CONFIG_HOME` | `~/.config/orm-mcp-go/` |

**デバッグ用**: `ORM_MCP_GO_DEBUG_DIR`を設定すると、全てのパスがその値で上書きされます。

詳細は[API_REFERENCE.md](API_REFERENCE.md)を参照してください。
