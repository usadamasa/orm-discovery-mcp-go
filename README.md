# O'Reilly Learning Platform MCP Server

O'Reilly Learning PlatformのコンテンツをModel Context Protocol (MCP)経由で検索・管理できるGoサーバーです。

## 概要

このプロジェクトは[odewahn/orm-discovery-mcp](https://github.com/odewahn/orm-discovery-mcp)にインスパイアされ、mcp-goパッケージを使用してMCPサーバーを構築する例を提供します。

## 主な機能

- **コンテンツ検索**: O'Reillyコンテンツの高度な検索
- **コレクション管理**: コレクションの作成、編集、コンテンツの追加・削除
- **プレイリスト管理**: プレイリストの作成、一覧表示、コンテンツ追加、詳細取得
- **マイコレクション表示**: 既存コレクションの一覧表示と詳細取得
- **書籍要約生成**: 複数書籍の日本語要約とまとめ生成

## クイックスタート

### 1. 認証情報の設定

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

### 2. サーバーの起動

```bash
go run .
```

### 3. Cline（Claude Desktop）での設定

#### 方法1: .envファイルを使用（推奨）

プロジェクトディレクトリに`.env`ファイルを作成し、以下の設定をCline設定に追加：

```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "command": "/your/path/to/orm-discovery-mcp-go",
      "args": []
    }
  }
}
```

#### 方法2: 環境変数で直接設定

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
| ツール名 | 説明 |
|---------|------|
| `search_content` | O'Reillyコンテンツの検索 |
| `summarize_books` | 書籍の日本語要約生成 |

### コレクション管理
| ツール名 | 説明 |
|---------|------|
| `list_collections` | マイコレクションの一覧表示 |
| `create_collection` | 新しいコレクションの作成 |
| `add_to_collection` | コレクションへのコンテンツ追加 |
| `remove_from_collection` | コレクションからのコンテンツ削除 |
| `get_collection_details` | コレクションの詳細情報取得 |

### プレイリスト管理
| ツール名 | 説明 |
|---------|------|
| `list_playlists` | プレイリストの一覧表示 |
| `create_playlist` | 新しいプレイリストの作成 |
| `add_to_playlist` | プレイリストへのコンテンツ追加 |
| `get_playlist_details` | プレイリストの詳細情報取得 |

## 使用例

### Quarkusコレクションの作成

実装したコレクション管理機能を使用して、Quarkusに関する学習リソースを整理できます：

```bash
# 1. Quarkusコンテンツを検索
# 2. 専用コレクションを作成
# 3. 関連コンテンツを追加
# 4. コレクション内容を確認

# 詳細な手順は QUARKUS_COLLECTION_DEMO.md を参照
```

## ドキュメント

- [技術概要](TECHNICAL_OVERVIEW.md) - アーキテクチャと実装詳細
- [API仕様](API_REFERENCE.md) - 利用可能なAPIとパラメータ

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

### 従来の手動Cookie設定（オプション）

手動でCookieを設定したい場合：

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
