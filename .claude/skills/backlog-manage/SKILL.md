---
name: backlog-manage
description: バックログ管理 - タスク/アイデア/イシューの追加・完了・一覧・MDサマリ再生成・監査・振り返り
user_invocable: true
---

# Backlog Management Skill

`.backlog/` ディレクトリのJSONLファイルを操作してバックログを管理する。

コア操作は `backlog-cli` (Go CLI) で実行する。バイナリは `.claude/skills/backlog-manage/cli/bin/backlog-cli` にある。

## Context

!(grep -c . .backlog/tasks.jsonl 2>/dev/null || printf "0") | xargs printf "Active tasks: %s"
!(grep -c . .backlog/ideas.jsonl 2>/dev/null || printf "0") | xargs printf ", Active ideas: %s"
!(grep -c . .backlog/issues.jsonl 2>/dev/null || printf "0") | xargs printf ", Active issues: %s"
!(ls ~/.claude/projects/*/memory/SESSION_HANDOFF_*.md 2>/dev/null | wc -l | tr -d ' ' || printf "0") | xargs printf ", Handoff files: %s"
!(gh issue list -R usadamasa/orm-discovery-mcp-go --label voc --state open --json number 2>/dev/null | jq length 2>/dev/null || echo "N/A") | xargs printf ", VOC issues: %s"
!(ls .backlog/*.bak 2>/dev/null | wc -l | tr -d ' ' || printf "0") | xargs printf ", Backup files: %s"
!(grep -c . .backlog/audit-log.jsonl 2>/dev/null || printf "0") | xargs printf ", Audit runs: %s"

## Setup

バイナリが未ビルドの場合は自動的にビルドされる:

```bash
task backlog:build
```

以下の `backlog-cli` コマンドはプロジェクトルートから実行する (デフォルトで `--dir .backlog` を使用)。

## Usage

ユーザーの指示に応じて以下のコマンドを実行する。引数が不足している場合はユーザーに確認する。

## Commands

### `add-task`

タスクを追加する。

**必須**: title, description
**任意**: priority (default: p2), tags (default: [])

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli add task --title "{title}" --description "{description}" --priority "{priority}" --tags "{tag1},{tag2}"
```

### `add-idea`

アイデアを追加する。

**必須**: title, description
**任意**: tags (default: [])

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli add idea --title "{title}" --description "{description}" --tags "{tag1},{tag2}"
```

### `add-issue`

イシューを追加する。

**必須**: title, description
**任意**: severity (default: medium), tags (default: [])

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli add issue --title "{title}" --description "{description}" --severity "{severity}" --tags "{tag1},{tag2}"
```

### `complete`

指定アイテムをdoneファイルに移動する。

**必須**: id (例: task-20260301-a3f2)

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli complete {id}
```

IDプレフィックスから自動的にファイルを判定し、status/done_at/resolved_at を設定してdoneファイルに移動する。

### `list`

全アクティブアイテムの一覧を表示する。

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli list
# タイプでフィルタ
.claude/skills/backlog-manage/cli/bin/backlog-cli list --type task
```

### `promote-idea`

アイデアをタスクまたはイシューに昇格する。

**必須**: idea_id, target_type (`task` or `issue`)
**任意**: priority (default: p2), severity (default: medium)

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli promote --id {idea_id} --to task --priority p1
.claude/skills/backlog-manage/cli/bin/backlog-cli promote --id {idea_id} --to issue --severity high
```

### `regenerate-md`

MDサマリファイルを全再生成する。全コマンド実行後に自動的に呼ばれる。

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli regenerate-md
```

以下のファイルを再生成する:
- `.backlog/README.md` - 統計サマリ
- `.backlog/TASKS.md` - タスクボード (優先度別テーブル)
- `.backlog/IDEAS.md` - アイデアボード
- `.backlog/ISSUES.md` - イシューボード (重要度別テーブル)

### `audit`

バックログの健全性を自己診断し、ギャップを検出・修正する。

**引数**: なし (全チェックを実行)

#### Phase 1: Analyze (情報源スキャン)

`backlog-cli audit --run` を実行して自動チェックを行う。

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli audit --run
```

このコマンドは以下の 5 チェックを Go ネイティブで実行し、結果を audit-log.jsonl に自動記録する:
1. JSONL 整合性 (tasks/ideas/issues)
2. アイデア滞留 (30 日超)
3. 残留バックアップファイル
4. MD サマリ同期
5. 未追跡ハンドオフ

#### Phase 2: Diff (ギャップ検出)

Phase 1 の結果から以下のギャップを検出する。

| ギャップタイプ | check_key | 検出ロジック |
|--------------|-----------|-------------|
| JSONL 整合性 | `jsonl_integrity` | 行数 vs JSON パース成功数の不一致 |
| 未追跡ハンドオフ | `untracked_handoffs` | SESSION_HANDOFF_*.md が存在するが、対応する issue/task がない |
| 未連携 GH Issue | `unlinked_gh_issues` | `gh issue list --label voc` の number が issues.jsonl の `github_issue` にない |
| アイデア滞留 | `stale_ideas` | `created_at` から 30 日以上経過し `status=active` のまま |
| 残留バックアップ | `backup_files` | `.backlog/*.bak` ファイルが存在する |
| MEMORY 重複 | `memory_duplicates` | 同一テキストブロックが 2 回以上出現 |
| MD サマリ | `md_summaries` | MD サマリが JSONL と同期していない |

全 7 チェックは `backlog-cli audit --run` で自動実行される。`gh` コマンド未インストール時は `unlinked_gh_issues` が skip (pass) となる。

#### Phase 3: Report (構造化レポート)

Phase 1-2 の結果を `dogfood-verify` と同じ形式で報告する。

```
=== Backlog Health Check ===

✅ JSONL integrity: N ideas, N tasks, N issues (all valid JSON)
❌ Untracked handoffs: N files (names...)
❌ Unlinked GH Issues: N (#num, #num, ...)
⚠️ Stale ideas: N (none over 30 days)
❌ Backup files: N (filenames...)
❌ MEMORY duplicates: N blocks
✅ MD summaries: up to date

Score: N/M checks passed
```

#### Phase 4: Patch (自動修正)

検出したギャップに対して、ユーザー確認後に以下の修正を適用する。

| ギャップ | 自動修正アクション |
|---------|-------------------|
| 未追跡ハンドオフ | ハンドオフ内容を読み取り → `add-issue` で Issue 作成 → ハンドオフファイル削除 |
| 未連携 GH Issue | GH Issue の title/labels から severity を推定 → `add-issue` で作成 (`github_issue` フィールド設定) |
| 残留バックアップ | `.bak` ファイルを削除 |
| MEMORY 重複 | 重複ブロックを除去 (Edit ツール) |
| アイデア滞留 | 対話的にトリアージ提案 (promote or archive) |
| JSONL 整合性 | 壊れた行を報告 (自動修正はしない) |

**重要**: 各修正は既存コマンド (`add-issue`, `complete`) を再利用する。新しい修正ロジックを独自に書かない。

#### Phase 5: Validate (検証)

修正後に `regenerate-md` を実行し、再度 Phase 1-3 を実行して全チェックが ✅ になることを確認する。

#### Phase 6: Log (記録)

Phase 1-5 の実行結果を `.backlog/audit-log.jsonl` に追記する。チェック名は Phase 2 テーブルの `check_key` カラムを使用する。

```bash
# audit-log エントリの生成と追記
# {FINDINGS_JSON} は Phase 1-2 の各チェック結果の JSON 配列
# {PATCH_ACTIONS_JSON} は Phase 4 で実行した修正アクションの JSON 配列
backlog-cli audit log-entry \
  --findings '{FINDINGS_JSON}' \
  --patch-actions '{PATCH_ACTIONS_JSON}'
```

**findings の各要素**: `{'check': '<check_key>', 'status': 'pass'|'fail', 'detail': '...', 'patched': true|false}`

**スキーマ**:

| フィールド | 型 | 説明 |
|-----------|------|------|
| `id` | string | `audit-{YYYYMMDD}-{4hex}` |
| `run_at` | string | ISO 8601 タイムスタンプ |
| `score` | object | `{passed: N, total: M}` |
| `findings` | array | 各チェックの結果 (`check`, `status`, `detail`, `patched?`) |
| `patch_actions` | array | Phase 4 で実行した修正アクションの説明文 |

### `retrospective`

audit-log.jsonl を分析し、管理手法の改善提案を生成する。`audit` の Phase 6 で蓄積されたログを使用する。

**引数**: `--last N` (default: 10、対象とする直近の audit 回数)

#### Step 1-2: ログ読み込みとパターン分析

`backlog-cli retrospective` コマンドがログ読み込みとパターン分析を一括実行する。

```bash
# テキスト出力 (人間向け)
backlog-cli retrospective --last 10

# JSON 出力 (スキル内処理向け)
backlog-cli retrospective --last 10 --json
```

ログが存在しない場合は stderr に「no audit entries」と出力する。

| 指標 | 検出ロジック | 改善提案 |
|------|-------------|---------|
| 再発チェック | 同一 `check` が 3 回以上 `fail` | Phase 4 の Patch ロジック見直しを提案 |
| 未修正の繰り返し | `patched=false` の `fail` が連続 | `add-task` で対応タスクを作成提案 |
| スコア停滞 | 直近 N 回でスコアが改善しない | 構造的な問題を `issues.jsonl` に登録提案 |
| 全パス継続 | 直近 5 回以上すべて `pass` | 新しいチェック項目の追加を提案 |

#### Step 3: レポート出力

```
=== Backlog Retrospective ===
対象: 直近 N 回 (YYYY-MM-DD 〜 YYYY-MM-DD)

【再発パターン】
❌ stale_ideas: 5/10 回 fail (3 回 unpatched)
   → add-task 提案: アイデアの定期トリアージ
✅ jsonl_integrity: 問題なし

【スコア推移】
  3/7 → 6/7 → 7/7 (改善傾向 ✅)

【改善提案】
[ ] stale_ideas の閾値を 30 日 → 15 日に短縮
[ ] 新規チェック: pr_review_gate
```

#### Step 4: 改善適用 (オプション)

提案についてユーザーの承認を得た後に適用する。SKILL.md の局所的な変更 (数値閾値、新規チェック) のみ自動適用する。
`backlog-practices.md` やグローバル設定への変更はユーザーに委ねる。

## ID Generation

ID形式: `{type}-{YYYYMMDD}-{4桁hex}`

`backlog-cli` が自動生成する。手動での ID 生成は不要。

## Important Rules

- JSONLファイルを `Write` ツールで上書きしない(既存行が消える)
- コア操作 (add/complete/list/promote/regenerate-md) は `backlog-cli` を使う
- done ファイルは append-only(エントリを削除しない)
- MDサマリは自動生成のため手動編集しない
- 全操作後に `regenerate-md` を自動実行する
- audit/retrospective はスキル側のロジックで実行する (backlog-cli のスコープ外)
