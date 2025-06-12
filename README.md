# O'Reilly Learning Platform MCP Server

O'Reilly Learning PlatformのコンテンツをModel Context Protocol (MCP)経由で検索・管理できるGoサーバーです。

## 概要

このプロジェクトは[odewahn/orm-discovery-mcp](https://github.com/odewahn/orm-discovery-mcp)にインスパイアされ、mcp-goパッケージを使用してMCPサーバーを構築する例を提供します。

## 主な機能

- O'Reillyコンテンツの検索
- マイコレクションの一覧表示
- 書籍の日本語要約生成

## クイックスタート

### 1. 認証情報の設定

重要なCookieキーを個別に設定（推奨）：

```bash
export OREILLY_JWT="your_orm_jwt_token_here"
export OREILLY_SESSION_ID="your_groot_sessionid_here"
export OREILLY_REFRESH_TOKEN="your_orm_rt_token_here"
```

### 2. サーバーの起動

```bash
go run .
```

### 3. Cline（Claude Desktop）での設定

```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "command": "/your/path/to/orm-discovery-mcp-go",
      "args": [],
      "env": {
        "OREILLY_JWT": "your_orm_jwt_token_here",
        "OREILLY_SESSION_ID": "your_groot_sessionid_here",
        "OREILLY_REFRESH_TOKEN": "your_orm_rt_token_here"
      }
    }
  }
}
```

## 利用可能なツール

| ツール名 | 説明 |
|---------|------|
| `search_content` | O'Reillyコンテンツの検索 |
| `list_collections` | マイコレクションの一覧表示 |
| `summarize_books` | 書籍の日本語要約生成 |

## ドキュメント

- [技術概要](TECHNICAL_OVERVIEW.md) - アーキテクチャと実装詳細
- [API仕様](API_REFERENCE.md) - 利用可能なAPIとパラメータ

## 認証情報の取得方法

1. ブラウザでO'Reilly Learning Platformにログイン
2. 開発者ツール（F12）→ Application → Cookies → learning.oreilly.com
3. 以下のキーの値をコピー：
   - `orm-jwt` (最重要)
   - `groot_sessionid`
   - `orm-rt`

## 免責事項

このツールの使用により生じるいかなる損害、損失、または不利益についても、開発者および貢献者は一切の責任を負いません。ユーザーは自己責任でこのツールを使用してください。

## ライセンス

[LICENSE](LICENSE)ファイルを参照してください。
