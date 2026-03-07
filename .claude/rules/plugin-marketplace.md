---
paths:
  - ".claude-plugin/**/*.json"
  - "plugins/agents/**/*.md"
---

# Claude Code Plugin 構成ルール

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

## plugin.json

公式スキーマ: `name` のみ必須。

```json
{
  "name": "plugin-name",
  "author": { "name": "author-name" },
  "repository": "https://github.com/owner/repo",
  "license": "MIT",
  "agents": ["./plugins/agents/agent-name.md"],
  "mcpServers": {
    "server-name": {
      "command": "binary-name",
      "args": [],
      "env": { "ENV_VAR": "${ENV_VAR}" }
    }
  }
}
```

### フィールド制限

npm の `package.json` とは異なるスキーマ。以下のフィールドは使用不可:
`homepage`, `bugs`, `main`, `scripts`, `transport` (mcpServers 内)

`version` と `description` は公式スキーマでは有効だが、本プロジェクトでは `marketplace.json` に一本化しているため `plugin.json` には書かない。

## marketplace.json

```json
{
  "name": "repository-name",
  "owner": { "name": "github-username", "email": "user@example.com" },
  "metadata": { "description": "説明", "version": "0.1.0" },
  "plugins": [{ "name": "plugin-name", "source": "./" }]
}
```

バージョンは `metadata.version` と `plugins[].version` の 2 箇所で管理。

## エージェント定義 (Markdown)

### Frontmatter (必須)

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `name` | o | エージェント識別子 (ハイフン区切り) |
| `description` | o | 説明と使用例 |
| `model` | - | `inherit` (親から継承) または特定モデル |
| `color` | - | UI表示色 |

### 本文セクション

Available Tools / Available Resources セクションは **server.go の登録内容と一致** させること。
ドリフト検出: `task plugin:validate:agent-drift`

## バージョン管理

### バージョンバンプのタイミング

| 変更種別 | バンプ | 例 |
|---------|--------|-----|
| ツール/リソースの追加・削除 | MINOR | 0.4.0 → 0.5.0 |
| ツール/リソースの引数・振る舞い変更 | MINOR | 0.4.0 → 0.5.0 |
| エージェント定義のみの修正 | PATCH | 0.4.0 → 0.4.1 |
| 破壊的変更 (URI 変更、ツール削除) | MAJOR | 0.4.0 → 1.0.0 |

### 更新箇所

以下の 2 箇所を同時に更新する:
1. `marketplace.json` の `metadata.version`
2. `marketplace.json` の `plugins[].version`

## 検証

- JSON 構文: `jq empty .claude-plugin/marketplace.json .claude-plugin/plugin.json`
- プラグイン全体: `CLAUDECODE= claude plugin validate .`
- エージェントドリフト: `task plugin:validate:agent-drift`
- 統合検証: `task plugin:validate:marketplace`
