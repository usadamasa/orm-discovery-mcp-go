---
paths:
  - ".claude-plugin/**/*.json"
  - "plugins/agents/**/*.md"
---

# Claude Code プラグインマーケットプレイス構成ルール

このルールは、Claude Code プラグインマーケットプレイスの構成方法を定義する。

## ディレクトリ構造

```
repository-root/
├── .claude-plugin/
│   ├── marketplace.json     # マーケットプレイス定義 (必須)
│   └── plugin.json          # プラグイン設定 (必須)
└── plugins/
    └── agents/
        └── {agent-name}.md  # エージェント定義 (任意)
```

## marketplace.json

マーケットプレイス全体の定義ファイル。

### 必須フィールド

```json
{
  "name": "repository-name",
  "owner": {
    "name": "github-username"
  },
  "metadata": {
    "description": "マーケットプレイスの説明",
    "version": "0.1.0"
  },
  "plugins": []
}
```

### plugins[] エントリ

```json
{
  "name": "plugin-name",
  "source": "./",
  "description": "プラグインの説明",
  "version": "0.1.0",
  "author": {
    "name": "author-name"
  },
  "repository": "https://github.com/owner/repo",
  "license": "MIT",
  "keywords": ["keyword1", "keyword2"],
  "category": "productivity",
  "tags": ["tag1", "tag2"]
}
```

### カテゴリ選択肢

- `productivity`: 生産性向上ツール
- `development`: 開発支援ツール
- `research`: リサーチ・調査ツール

## plugin.json

個別プラグインの設定ファイル。

### 必須フィールド

```json
{
  "name": "plugin-name",
  "version": "0.1.0",
  "description": "プラグインの説明"
}
```

### 任意フィールド

```json
{
  "author": {
    "name": "author-name"
  },
  "repository": "https://github.com/owner/repo",
  "license": "MIT",
  "agents": ["./plugins/agents/agent-name.md"],
  "mcpServers": {
    "server-name": {
      "command": "binary-name",
      "description": "サーバーの説明",
      "env": {
        "ENV_VAR": "${ENV_VAR}"
      }
    }
  }
}
```

### agents[] パス規則

- プラグインルートからの相対パス
- `./plugins/agents/` ディレクトリに配置
- ファイル名: `{agent-name}.md`

### mcpServers 設定

- `command`: 実行可能バイナリ名 (PATH上にあること)
- `env`: 環境変数マッピング (`${VAR}` 形式で参照)

## エージェント定義 (Markdown)

### Frontmatter (必須)

```yaml
---
name: agent-name
description: |
  エージェントの説明。

  Examples:
  <example>
  Context: 使用コンテキスト
  user: "ユーザーの発言"
  assistant: "アシスタントの応答"
  <commentary>
  コメンタリー。
  </commentary>
  </example>

model: inherit
color: blue
---
```

### Frontmatter フィールド

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `name` | ✓ | エージェント識別子 (ハイフン区切り) |
| `description` | ✓ | 説明と使用例 |
| `model` | - | `inherit` (親から継承) または特定モデル |
| `color` | - | UI表示色 (`blue`, `green`, `purple` など) |

### 本文セクション構成

1. **役割説明**: エージェントの専門性
2. **Available Tools**: 利用可能なツール一覧
3. **Available Resources**: 利用可能なリソース一覧
4. **Workflow**: 推奨ワークフロー
5. **Output Format**: 出力形式の規定
6. **Citation Requirements**: 引用要件 (該当する場合)

## バージョン管理

### バージョン同期規則

以下のバージョンは同期する:

1. `marketplace.json` の `metadata.version`
2. `marketplace.json` の `plugins[].version`
3. `plugin.json` の `version`

### バージョン形式

Semantic Versioning (semver) に従う:

```
MAJOR.MINOR.PATCH
例: 0.1.0, 1.0.0, 1.2.3
```

## 検証手順

### Taskfile 統合

```yaml
# Taskfile.yml
plugin:validate:marketplace:
  desc: Validate marketplace and plugin configuration
  cmds:
    - |
      # JSON形式検証
      jq empty .claude-plugin/marketplace.json
      jq empty .claude-plugin/plugin.json

      # バージョン同期検証
      MARKETPLACE_VER=$(jq -r '.metadata.version' .claude-plugin/marketplace.json)
      PLUGIN_VER=$(jq -r '.version' .claude-plugin/plugin.json)
      if [ "$MARKETPLACE_VER" != "$PLUGIN_VER" ]; then
        echo "Version mismatch: marketplace=$MARKETPLACE_VER, plugin=$PLUGIN_VER"
        exit 1
      fi

      # エージェントパス検証
      for agent in $(jq -r '.agents[]' .claude-plugin/plugin.json 2>/dev/null); do
        if [ ! -f "$agent" ]; then
          echo "Agent file not found: $agent"
          exit 1
        fi
      done

      echo "Marketplace validation passed"
```

### CI 統合

```yaml
# task ci に含める
ci:
  deps: [plugin:validate:marketplace]
  cmds:
    - task: generate:api:oreilly
    - task: format
    - task: lint
    - task: test:coverage
    - task: build
```

## インストール方法

### マーケットプレイス追加

```bash
/plugin marketplace add owner/repository-name
```

### プラグインインストール

```bash
/plugin install plugin-name
```

### 環境変数設定

MCPサーバーが必要とする環境変数を事前に設定:

```bash
export ENV_VAR="value"
```

## セキュリティ考慮事項

### 認証情報の取り扱い

- 認証情報 (password, API keys) は `${VAR}` 参照のみ使用
- ハードコード厳禁
- プラグイン設定ファイルには秘密情報を含めない
- README に環境変数設定の警告を記載

### MCP サーバー設定

- 環境変数は必ず `${VAR}` 形式で参照
- 認証エラー時の適切なエラーメッセージを実装
- `transport` フィールドで通信方式を明示 (stdio/http)

### 検証チェックリスト

- [ ] 認証情報がハードコードされていない
- [ ] 秘密情報が設定ファイルに含まれていない
- [ ] 環境変数参照が正しい形式 (`${VAR}`)

## リソース URI パターンの標準化

本プロジェクトでは以下の URI スキームを使用する:

| スキーム | 用途 | 例 |
|---------|------|-----|
| `oreilly://` | O'Reilly 標準コンテンツ | `oreilly://book-details/{product_id}` |
| `orm-mcp://` | MCP サーバー固有機能 | `orm-mcp://history/recent` |

### リソース URI 一覧

**O'Reilly コンテンツ:**
- `oreilly://book-details/{product_id}` - 書籍詳細
- `oreilly://book-toc/{product_id}` - 目次
- `oreilly://book-chapter/{product_id}/{chapter_name}` - 章コンテンツ
- `oreilly://answer/{question_id}` - Q&A 回答

**MCP サーバー固有:**
- `orm-mcp://history/recent` - 最近の調査履歴
- `orm-mcp://history/search{?keyword,type}` - 履歴検索
- `orm-mcp://history/{id}` - 特定の履歴エントリ
- `orm-mcp://history/{id}/full` - 完全なレスポンスデータ

## ベストプラクティス

1. **name の一貫性**: リポジトリ名、マーケットプレイス名、プラグイン名を統一
2. **description の明確さ**: 50-100文字で機能を簡潔に説明
3. **keywords/tags の適切さ**: 検索性を考慮した選択
4. **エージェント例の充実**: 3つ以上の使用例を含める
5. **バージョン同期の自動化**: リリース時にスクリプトで同期確認
6. **セキュリティ第一**: 認証情報は環境変数参照のみ
