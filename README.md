# O'Reilly Learning Platform MCP Server

O'Reilly Learning PlatformのコンテンツをModel Context Protocol (MCP)経由で検索・管理できるGoサーバーです。

## 概要

このプロジェクトは[odewahn/orm-discovery-mcp](https://github.com/odewahn/orm-discovery-mcp)
にインスパイアされ、mcp-goパッケージを使用してMCPサーバーを構築する例を提供します。

> [!WARNING]
> This project is an unofficial implementation of an MCP server for interacting with the O'Reilly Learning Platform.
> It is **not affiliated with or endorsed by O'Reilly Media, Inc.** in any way.
> This tool is provided **for educational and personal use only**, and may break at any time if the internal APIs or authentication flows used by O'Reilly change.
> Use at your own risk, and please refer to O'Reilly's [Terms of Service](https://www.oreilly.com/terms/) before using this tool.

## 主な機能

- **コンテンツ検索**: O'Reillyコンテンツの高度な検索
- **目次抽出**: O'Reilly書籍の目次を自動抽出

## 開発環境セットアップ

### 1. 必要なツールのインストール

#### aquaを使用したツール管理

```bash
# aquaのインストール(未インストールの場合)
curl -sSfL https://raw.githubusercontent.com/aquaproj/aqua/main/install.sh | bash

# パッケージのインストール
aqua install
```

#### Task（タスクランナー）の使用

```bash
# 利用可能なタスクを確認
task --list

# OpenAPIクライアントコードの生成
task generate:api:oreilly

# 生成されたコードのクリーンアップ
task clean:generated
```

### 2. 認証情報の設定

#### 方法1: .envファイルを使用（推奨）

プロジェクトディレクトリに`.env`ファイルを作成：

```bash
# .env
OREILLY_USER_ID=your_email@acm.org
OREILLY_PASSWORD=your_password
PORT=8080
TRANSPORT=stdio
```

#### 方法2: 環境変数で設定

```bash
export OREILLY_USER_ID="your_email@acm.org"
export OREILLY_PASSWORD="your_password"
```

**注意**:

- ACMメンバーの場合は`@acm.org`のメールアドレスを使用
- ACM IDPリダイレクトは自動で処理されます
- `.env`ファイルの設定が環境変数より優先されます

### 3. サーバーの起動

```bash
go run .
```

### 4. Cline（Claude Desktop）での設定

```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "command": "/your/path/to/orm-discovery-mcp-go",
      "args": [],
      "env": {
        "OREILLY_USER_ID": "your_email@acm.org",
        "OREILLY_PASSWORD": "your_password"
      }
    }
  }
}
```

## 利用可能なツール

### コンテンツ検索・要約

| ツール名             | 説明                        |
|------------------|---------------------------|
| `search_content` | O'Reillyコンテンツの検索（ブラウザベース） |

### 書籍機能

| ツール名               | 説明      |
|--------------------|---------|
| `get_book_details` | 書籍の詳細取得 |
| `get_book_toc`     | 書籍の目次取得 |

## 開発ツール

### aqua（パッケージマネージャー）

- **設定ファイル**: `aqua.yaml`
- **管理ツール**: Task（go-task）
- **用途**: 開発に必要なツールの統一管理

### Task（タスクランナー）

- **設定ファイル**: `Taskfile.yml`
- **利用可能なタスク**:
    - `generate:api:oreilly`: OpenAPIクライアントコード生成
    - `clean:generated`: 生成コードのクリーンアップ

### OpenAPI/oapi-codegen

- **仕様ファイル**: `browser/openapi.yaml`
- **設定ファイル**: `browser/oapi-codegen.yaml`
- **出力先**: `browser/generated/api/`
- **用途**: O'Reilly Learning Platform APIクライアント生成

## 認証システム

### ヘッドレスブラウザ認証

このサーバーはヘッドレスブラウザ（Chrome）を使用して自動的にO'Reillyにログインします：

1. **自動ログイン**: 環境変数のID/パスワードでログイン
2. **ACM対応**: ACM IDPリダイレクトを自動処理
3. **セッション管理**: ログイン後のCookieを自動取得・管理
4. **ホームページ取得**: ブラウザでホームページのコレクション情報も取得

### 必要な環境変数

- `OREILLY_USER_ID`: O'Reillyのメールアドレス
- `OREILLY_PASSWORD`: O'Reillyのパスワード

## 免責事項

このツールの使用により生じるいかなる損害、損失、または不利益についても、開発者および貢献者は一切の責任を負いません。ユーザーは自己責任でこのツールを使用してください。

## ライセンス

[LICENSE](LICENSE)ファイルを参照してください。
