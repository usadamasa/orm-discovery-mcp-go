---
name: dogfood-improve
description: >
  dogfood-verify スキルの自己改善ループ。server.go の MCP 登録と
  スキルファイルのスモークテスト項目を比較し、差分を検出・自動修正する。
  「スキル改善」「improve dogfood」「dogfood 更新」等の手動トリガーに対応。
  dogfood-verify 実行後にプロアクティブに呼び出すことも推奨。
---

# dogfood-improve

dogfood-verify スキルの自己改善ループ。MCP サーバーの登録状態とスキルファイルの整合性を検証し、自動更新する。

## トリガー条件

- **手動**: 「スキル改善」「improve dogfood」「dogfood 更新」等
- **プロアクティブ**: dogfood-verify 実行後、または MCP ツール追加/削除のコミット後

## Context

- Current branch: !`git branch --show-current`
- Skill file: !`wc -l .claude/skills/dogfood-verify/SKILL.md 2>/dev/null || echo "Not found"`

## ワークフロー

### Step 1: MCP 登録状態の収集

全 Go ソースファイルから現在登録されている MCP ケイパビリティを抽出する。

**注意**: これらの grep コマンドはエージェントへのガイドラインであり、Grep ツールを使用して実行する。

```bash
# ツール登録 (server.go)
grep -n 'mcp.AddTool' server.go

# リソース登録 (server.go + history_resources.go)
grep -n 'AddResource[^T]' server.go history_resources.go

# リソーステンプレート登録 (server.go + history_resources.go)
grep -n 'AddResourceTemplate' server.go history_resources.go

# プロンプト登録 (prompts.go, server.go)
grep -n 'AddPrompt' prompts.go server.go
```

各登録のツール名/リソース URI/プロンプト名を抽出し、リストを作成する。

### Step 2: dogfood-verify スキルの解析

`.claude/skills/dogfood-verify/SKILL.md` を読み込み、以下を抽出する:

- Phase 4 のスモークテスト項目 (ツール名のリスト)
- 各テストの成功条件と失敗時の対応
- Context セクションの情報

### Step 3: 差分検出

Step 1 と Step 2 の結果を比較し、以下の差分を検出する:

| 差分タイプ | 説明 | アクション |
|-----------|------|---------- |
| **新規ツール** | server.go に登録されているがスキルに未記載 | Phase 4 にスモークテスト追加 |
| **削除ツール** | スキルに記載されているが server.go から削除 | Phase 4 からテスト削除 |
| **新規リソース** | server.go / history_resources.go に登録されているがスキルに未記載 | 必要に応じて Phase 4 に追加 |
| **新規プロンプト** | prompts.go / server.go に登録されているがスキルに未記載 | 情報を更新 |

### Step 4: スキルファイル自動更新

差分が検出された場合、dogfood-verify の SKILL.md を更新する。

#### 新規ツール追加時のテンプレート

Phase 4 セクションに以下の形式で追加:

```markdown
#### 4-N: {tool_name}

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> {tool_name} の基本的な動作を確認してください

| 成功条件 | 失敗時の対応 |
|---------|------------|
| {期待される結果} | {エラー時の案内} |
```

#### ツール削除時

該当するテスト項目を削除し、番号を振り直す。

### Step 5: パターン準拠チェック

dogfood-verify スキルが以下のパターンに準拠しているか検証する:

| チェック項目 | 基準 |
|-------------|------|
| Context セクション | `!` バッククォートでランタイム情報を埋め込み |
| Phase 構成 | CI → Install → 確認 → テスト → レポート の順序 |
| 結果報告 | 構造化レポート (✅/❌/⏭️) |
| 注意事項 | CI 優先ポリシー、エラー時停止 |
| finalize-pr 連携 | Phase 7 で推奨 |

### Step 6: 結果報告

```markdown
## Dogfood Improve Result

### MCP ケイパビリティ:
- Tools: {数} 件 (server.go)
- Resources: {数} 件
- Resource Templates: {数} 件
- Prompts: {数} 件

### 差分検出:
- 新規追加: {リスト}
- 削除済み: {リスト}
- 変更なし: {リスト}

### パターン準拠: ✅/❌
### 更新内容: {変更概要 or "変更なし"}
```

### Step 7: コミット提案 (変更あり時のみ)

スキルファイルに変更があった場合、コミットを提案する:

```
chore: update dogfood-verify skill to match current MCP capabilities
```

## 注意事項

- ソースファイルの解析は Grep ツールベース (AST 解析ではない)
- 検索対象: server.go, history_resources.go, prompts.go (MCP 登録を含む全ファイル)
- ツール名は `mcp.AddTool` の引数から推定する
- 自動更新後は差分を表示してユーザーに確認を取る
- このスキルは dogfood-verify のみを対象とする (他スキルは改善しない)
