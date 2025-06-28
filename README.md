# O'Reilly Learning Platform MCP Server

O'Reilly Learning PlatformのコンテンツをModel Context Protocol (MCP)経由で検索・アクセスできるGoサーバーです。

> [!WARNING]
> This project is an unofficial implementation of an MCP server for interacting with the O'Reilly Learning Platform.
> It is **not affiliated with or endorsed by O'Reilly Media, Inc.** in any way.
> This tool is provided **for educational and personal use only**, and may break at any time if the internal APIs or authentication flows used by O'Reilly change.
> Use at your own risk, and please refer to O'Reilly's [Terms of Service](https://www.oreilly.com/terms/) before using this tool.

## クイックスタート

### 1. ツールのインストール

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

### 利用フロー

#### コンテンツ検索・アクセス
1. `search_content`で検索 → `product_id`取得
2. `book-details`で書籍情報確認
3. `book-chapter`で必要な章を取得

#### AI質問応答
1. `ask_question`で技術的な質問を投稿 → `question_id`取得
2. `oreilly://answer/{question_id}`でAI生成回答を取得

詳細は[API_REFERENCE.md](API_REFERENCE.md)を参照してください。
