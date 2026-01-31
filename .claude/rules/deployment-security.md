---
paths:
  - "config.go"
  - ".env*"
  - ".github/workflows/*.yml"
---

# デプロイとセキュリティ

このルールは、設定・環境・CI 関連ファイルを編集する際に適用される。

## Environment Setup

### Required Environment Variables

```bash
# コア認証情報
OREILLY_USER_ID=your_email@example.com    # O'Reilly アカウントメール
OREILLY_PASSWORD=your_password             # O'Reilly パスワード

# サーバー設定
PORT=8080                                  # HTTP サーバーポート (オプション)
TRANSPORT=stdio                            # トランスポートモード: stdio or http

# Search Mode 設定
ORM_MCP_GO_DEFAULT_MODE=bfs               # デフォルト検索モード: bfs | dfs

# Sampling 設定
ORM_MCP_GO_ENABLE_SAMPLING=true           # MCP Sampling による要約 (default: true)
ORM_MCP_GO_SAMPLING_MAX_TOKENS=500        # Sampling レスポンスの最大トークン (default: 500)

# Development and debugging
ORM_MCP_GO_DEBUG=true                     # デバッグログとスクリーンショット有効化
ORM_MCP_GO_TMP_DIR=/path/to/tmp           # Cookie 用カスタム temp ディレクトリ
```

### .env File Support

`.env` ファイルを実行可能ファイルと同じディレクトリに配置。システムが自動検出・読み込み、`.env` の値が環境変数より優先される。

## Dependencies

| Category | Dependency | Description |
|----------|------------|-------------|
| Core Framework | `github.com/mark3labs/mcp-go v0.43.2` | MCP プロトコル実装 |
| Browser Automation | `github.com/chromedp/chromedp` | Chrome DevTools Protocol |

## Runtime Requirements

- 認証ブラウザ自動化に Chrome または Chromium
- モダン言語機能に Go 1.24.3+

## Security Considerations

### Authentication Security

- 環境変数での認証情報ストレージ (ハードコード厳禁)
- キャッシュ認証の Cookie ファイルパーミッション (0600)
- セッションタイムアウトと Cookie 有効期限処理
- O'Reilly プラットフォームのレート制限準拠

### Development Security

- 機密スクリーンショットキャプチャのデバッグモード制御
- セキュアな temp ディレクトリ設定
- 認証情報漏洩のない適切なエラー処理

### 認証情報の取り扱いルール

1. **環境変数のみ使用** - 認証情報は `${VAR}` 参照のみ
2. **ハードコード厳禁** - ソースコードに秘密情報を含めない
3. **設定ファイルに秘密情報を含めない** - `.env` はリポジトリ外で管理
4. **適切なファイルパーミッション** - Cookie ファイルは 0600

## CI Integration

### GitHub Actions

- すべての `task ci` タスクが PR マージの条件
- ローカル検証で CI 失敗を防止

### Marketplace Validation

```bash
# マーケットプレイスとプラグイン設定の検証
task plugin:validate:marketplace

# 完全 CI 検証 (マーケットプレイス検証含む)
task ci
```

## Troubleshooting

### MCP Server not starting

1. バイナリのインストール確認: `which orm-discovery-mcp-go`
2. 環境変数が設定されているか確認
3. 手動実行でエラーを確認: `orm-discovery-mcp-go`

### Authentication errors

1. O'Reilly 認証情報が正しいか確認
2. ACM IdP ログインが必要か確認
3. Cookie キャッシュをクリア: `rm -rf /tmp/orm-mcp-go-cookies/`
