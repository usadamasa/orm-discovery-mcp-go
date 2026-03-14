---
name: context-efficiency-improve
description: |
  MCPサーバーのコンテキスト効率を計測・分析・改善するワークフロー。
  Tool/Resource/Prompt の説明文サイズやレスポンスサイズを定量的に計測し、
  TDD で改善を実装する。改善の前後比較を自動化する。

  Use when: 「コンテキスト効率を改善して」「トークン消費を減らしたい」
  「MCP記述を最適化」「レスポンスサイズを確認」「説明文が長すぎる」
  「tools/list が重い」等、MCP メタデータやレスポンスのサイズ最適化に
  関する依頼を受けたとき。レスポンスサイズの計測だけでも使う。
---

# Context Efficiency Improvement Workflow

MCP サーバーが LLM に送信するメタデータとレスポンスのサイズを最適化するワークフロー。
`tools/list` で毎回送信される Tool 説明文が最もインパクトが大きく、最優先で改善する。

## Phase 1: Measure (計測)

現在のメトリクスを取得してベースラインを記録する。

```bash
task measure:context-efficiency
```

これは `go test -v -run TestContextEfficiencyReport ./...` を実行し、以下を出力する:

| カテゴリ | 内容 | 影響範囲 |
|---------|------|---------|
| A. Tool 説明 | 各ツールの description 文字数 | **全リクエスト** (tools/list) |
| B. Resource 説明 | リソースの description 文字数 | resources/list |
| C. Prompt 説明 | プロンプトの description 文字数 | prompts/list |
| D. 合計ペイロード | A+B+C の合計 + 推定トークン数 | 全体 |
| E-G. レスポンス | search/history のレスポンスサイズ + キャッシュファイルサイズ | 使用時のみ |

ガードレールテストも同時に実行:

```bash
go test -v -run TestToolDescriptionSizes ./...
```

## Phase 2: Analyze (分析)

### 改善優先度

影響範囲に基づいて優先度を判定する:

1. **Tool 説明** (最優先) - 全リクエストで無条件に消費される
2. **Resource/Template 説明** - resources/list で消費
3. **Prompt 説明** - prompts/list で消費
4. **レスポンスサイズ** - 使用時のみ消費

### ターゲット値

`mcp-tool-progressive-disclosure` スキルのガイドラインに基づく:

- Tool 説明: 各 350 文字以内 (定数 `maxToolDescriptionLen`)
- 1 Good/1 Poor の例ペアのみ
- jsonschema パラメータ description と情報を重複させない

### 分析チェックリスト

- [ ] 各 Tool 説明がターゲット以内か
- [ ] 余分な例示がないか (1 Good/1 Poor ペアで十分)
- [ ] jsonschema パラメータ description との重複がないか
- [ ] rate limit/timeout 等の運用情報が description に含まれていないか
- [ ] `format="markdown"` 等のヒントが jsonschema 側に既出でないか

## Phase 3: Improve (改善)

### TDD で改善を実装する

1. `tool_descriptions.go` の定数を編集 (テストが先に存在する)
2. `go test -v -run TestToolDescriptionSizes ./...` で検証
3. `mcp-tool-progressive-disclosure` スキルの記述パターンに従う

### 記述パターン (progressive-disclosure)

```
[1行目: 何をするツールか]

Example: "[Good例]" (Good) / "[Poor例]" (Poor)

[結果の使い方: リソースへの導線]

IMPORTANT: [引用要件]
```

### 削除候補の典型パターン

- 3つ以上の Good 例 (1 Good/1 Poor で十分)
- ファイルベース遅延読み込みの冗長な説明 (jsonschema に既出)
- `Rate limit` ガイダンス (運用情報は agent 定義に移動)
- `Set format="markdown"` (jsonschema の format パラメータに既出)
- `Default timeout` (jsonschema に既出)

## Phase 4: Verify (検証)

```bash
# 1. CI 全体パス
task ci

# 2. コンテキスト効率レポート (before/after 比較)
task measure:context-efficiency

# 3. ガードレールテスト
go test -v -run TestToolDescriptionSizes ./...
```

改善効果をコミットメッセージに記載する:

```
optimize: reduce tool description size by X% (-Y tokens)

Before: Tool descriptions total AAAA chars (~BBB tokens)
After:  Tool descriptions total CCCC chars (~DDD tokens)
Savings: -EEE chars (~FFF tokens) per request
```

## Key Files

| ファイル | 役割 |
|---------|------|
| `tool_descriptions.go` | Tool 説明文の定数定義 |
| `context_efficiency_test.go` | ガードレール + レポートテスト |
| `Taskfile.yml` (`measure:context-efficiency`) | 計測タスク |
| `.claude/skills/mcp-tool-progressive-disclosure/SKILL.md` | 記述パターンガイド |

## Related Skills

- **mcp-tool-progressive-disclosure**: Phase 2-3 で参照する記述パターンガイド (WHAT)
- **mcp-go-sdk-practices**: Phase 3 で参照する SDK 実装パターン (HOW)
