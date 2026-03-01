---
name: backlog-manage
description: バックログ管理 - タスク/アイデア/イシューの追加・完了・一覧・MDサマリ再生成・監査・振り返り
user_invocable: true
---

# Backlog Management Skill

`.backlog/` ディレクトリのJSONLファイルを操作してバックログを管理する。

## Context

!(grep -c . .backlog/tasks.jsonl 2>/dev/null || printf "0") | xargs printf "Active tasks: %s"
!(grep -c . .backlog/ideas.jsonl 2>/dev/null || printf "0") | xargs printf ", Active ideas: %s"
!(grep -c . .backlog/issues.jsonl 2>/dev/null || printf "0") | xargs printf ", Active issues: %s"
!(ls ~/.claude/projects/*/memory/SESSION_HANDOFF_*.md 2>/dev/null | wc -l | tr -d ' ' || printf "0") | xargs printf ", Handoff files: %s"
!(gh issue list -R usadamasa/orm-discovery-mcp-go --label voc --state open --json number 2>/dev/null | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "N/A") | xargs printf ", VOC issues: %s"
!(ls .backlog/*.bak 2>/dev/null | wc -l | tr -d ' ' || printf "0") | xargs printf ", Backup files: %s"
!(grep -c . .backlog/audit-log.jsonl 2>/dev/null || printf "0") | xargs printf ", Audit runs: %s"

## Usage

ユーザーの指示に応じて以下のコマンドを実行する。引数が不足している場合はユーザーに確認する。

## Commands

### `add-task`

タスクを追加する。

**必須**: title, description
**任意**: priority (default: p2), tags (default: [])

```bash
echo '{"id":"task-{YYYYMMDD}-{4hex}","type":"task","title":"{title}","description":"{description}","status":"active","priority":"{priority}","tags":{tags},"source":"manual","source_ref":null,"github_issue":null,"created_at":"{ISO8601}","created_by":"manual","updated_at":"{ISO8601}","done_at":null,"notes":""}' >> .backlog/tasks.jsonl
```

### `add-idea`

アイデアを追加する。

**必須**: title, description
**任意**: tags (default: [])

```bash
echo '{"id":"idea-{YYYYMMDD}-{4hex}","type":"idea","title":"{title}","description":"{description}","status":"active","tags":{tags},"source":"manual","source_ref":null,"promoted_to":null,"created_at":"{ISO8601}","created_by":"manual","done_at":null}' >> .backlog/ideas.jsonl
```

### `add-issue`

イシューを追加する。

**必須**: title, description
**任意**: severity (default: medium), tags (default: [])

```bash
echo '{"id":"issue-{YYYYMMDD}-{4hex}","type":"issue","title":"{title}","description":"{description}","severity":"{severity}","status":"active","tags":{tags},"source":"manual","source_ref":null,"github_issue":null,"created_at":"{ISO8601}","created_by":"manual","resolved_at":null}' >> .backlog/issues.jsonl
```

### `complete`

指定アイテムをdoneファイルに移動する。

**必須**: id (例: task-20260301-a3f2)

1. IDのプレフィックスからファイルを判定 (`task-` → tasks, `idea-` → ideas, `issue-` → issues)
2. アクティブファイルから該当行を抽出
3. `done_at` / `resolved_at` を現在時刻に設定、`status` を `done` / `resolved` に更新
4. doneファイルに追記
5. アクティブファイルから該当行を削除

```bash
# 例: タスクの完了
LINE=$(grep '"id":"task-20260301-a3f2"' .backlog/tasks.jsonl)
UPDATED=$(echo "$LINE" | python3 -c "
import sys, json
from datetime import datetime, timezone
entry = json.loads(sys.stdin.read())
entry['status'] = 'done'
entry['done_at'] = datetime.now(timezone.utc).isoformat()
print(json.dumps(entry, ensure_ascii=False))
")
echo "$UPDATED" >> .backlog/tasks.done.jsonl
grep -v '"id":"task-20260301-a3f2"' .backlog/tasks.jsonl > .backlog/tasks.jsonl.tmp && mv .backlog/tasks.jsonl.tmp .backlog/tasks.jsonl
```

### `list`

全アクティブアイテムの一覧を表示する。

```bash
echo "=== Tasks ===" && cat .backlog/tasks.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    if line.strip():
        e = json.loads(line)
        print(f\"  [{e.get('priority','p2')}] {e['id']}: {e['title']} ({e['status']})\")
" 2>/dev/null || echo "  (none)"

echo "=== Ideas ===" && cat .backlog/ideas.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    if line.strip():
        e = json.loads(line)
        print(f\"  {e['id']}: {e['title']} ({e['status']})\")
" 2>/dev/null || echo "  (none)"

echo "=== Issues ===" && cat .backlog/issues.jsonl | python3 -c "
import sys, json
for line in sys.stdin:
    if line.strip():
        e = json.loads(line)
        print(f\"  [{e['severity']}] {e['id']}: {e['title']} ({e['status']})\")
" 2>/dev/null || echo "  (none)"
```

### `promote-idea`

アイデアをタスクまたはイシューに昇格する。

**必須**: idea_id, target_type (`task` or `issue`)
**任意**: priority (default: p2、target_type が `task` の場合のみ有効)

1. ideas.jsonl から該当アイデアを取得
2. 新しいタスク/イシューエントリを作成 (`source: "idea"`, `source_ref: idea_id`、priority 指定があれば反映)
3. タスク/イシューファイルに追記
4. アイデアの `status` を `promoted`、`promoted_to` を新IDに設定
5. ideas.done.jsonl に移動、ideas.jsonl から削除

### `regenerate-md`

MDサマリファイルを全再生成する。全コマンド実行後に自動的に呼ばれる。

以下のファイルを再生成する:
- `.backlog/README.md` - 統計サマリ
- `.backlog/TASKS.md` - タスクボード (優先度別テーブル)
- `.backlog/IDEAS.md` - アイデアボード
- `.backlog/ISSUES.md` - イシューボード (重要度別テーブル)

### `audit`

バックログの健全性を自己診断し、ギャップを検出・修正する。

**引数**: なし (全チェックを実行)

#### Phase 1: Analyze (情報源スキャン)

以下のコマンドで現在の状態を収集する。

```bash
# 1. JSONL 整合性チェック
for f in .backlog/tasks.jsonl .backlog/ideas.jsonl .backlog/issues.jsonl; do
  if [ -f "$f" ]; then
    TOTAL=$(wc -l < "$f" | tr -d ' ')
    VALID=$(python3 -c "
import json, sys
count = 0
for line in open('$f'):
    if line.strip():
        try:
            json.loads(line)
            count += 1
        except: pass
print(count)
")
    printf "%s: %s lines, %s valid JSON\n" "$f" "$TOTAL" "$VALID"
  fi
done

# 2. 未追跡ハンドオフ
ls ~/.claude/projects/*/memory/SESSION_HANDOFF_*.md 2>/dev/null || echo "(none)"

# 3. 未連携 GH Issues (VOC ラベル)
gh issue list -R usadamasa/orm-discovery-mcp-go --label voc --state open --json number,title 2>/dev/null || echo "N/A"

# 4. MEMORY.md 重複チェック
python3 -c "
import re
with open('$(echo ~/.claude/projects/*/memory/MEMORY.md | head -1)') as f:
    content = f.read()
blocks = re.split(r'\n## ', content)
seen = {}
dupes = []
for b in blocks:
    key = b.strip()
    if key in seen:
        dupes.append(key[:60])
    seen[key] = True
if dupes:
    print(f'Duplicate blocks: {len(dupes)}')
    for d in dupes:
        print(f'  - {d}...')
else:
    print('No duplicates')
" 2>/dev/null || echo "N/A"

# 5. 残留バックアップファイル
ls .backlog/*.bak 2>/dev/null || echo "(none)"

# 6. アイデア滞留チェック (30日超)
python3 -c "
import json, sys
from datetime import datetime, timezone, timedelta
threshold = datetime.now(timezone.utc) - timedelta(days=30)
stale = []
for line in open('.backlog/ideas.jsonl'):
    if line.strip():
        e = json.loads(line)
        if e.get('status') == 'active':
            created = datetime.fromisoformat(e['created_at'].replace('Z', '+00:00'))
            if created < threshold:
                stale.append(e['id'] + ': ' + e['title'])
if stale:
    print(f'Stale ideas ({len(stale)}):')
    for s in stale: print(f'  - {s}')
else:
    print('No stale ideas')
" 2>/dev/null || echo "(none)"
```

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

```bash
# GH Issue 連携チェック
python3 -c "
import json, subprocess, sys
try:
    result = subprocess.run(
        ['gh', 'issue', 'list', '-R', 'usadamasa/orm-discovery-mcp-go',
         '--label', 'voc', '--state', 'open', '--json', 'number,title'],
        capture_output=True, text=True, timeout=10)
    gh_issues = {i['number'] for i in json.loads(result.stdout)} if result.returncode == 0 else set()
except: gh_issues = set()

tracked = set()
try:
    for line in open('.backlog/issues.jsonl'):
        if line.strip():
            e = json.loads(line)
            gh = e.get('github_issue')
            if gh: tracked.add(int(gh) if isinstance(gh, str) and gh.startswith('#') else gh)
except FileNotFoundError: pass

unlinked = gh_issues - tracked
if unlinked:
    print(f'Unlinked GH Issues: {sorted(unlinked)}')
else:
    print('All GH Issues linked')
"
```

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
# {PASSED_COUNT}, {TOTAL_COUNT} は Phase 3 の実際の値で置換する
# {FINDINGS} は Phase 1-2 の各チェック結果から構築する
# {PATCH_ACTIONS} は Phase 4 で実行した修正アクションの説明文で構築する
python3 -c "
import json, random
from datetime import datetime, timezone

now = datetime.now(timezone.utc)
entry = {
    'id': f'audit-{now.strftime(\"%Y%m%d\")}-{random.randint(0,65535):04x}',
    'run_at': now.isoformat(),
    'score': {'passed': {PASSED_COUNT}, 'total': {TOTAL_COUNT}},
    'findings': {FINDINGS},
    'patch_actions': {PATCH_ACTIONS}
}
print(json.dumps(entry, ensure_ascii=False))
" >> .backlog/audit-log.jsonl
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

#### Step 1: ログ読み込み

```bash
# 直近 N 件の audit-log を読み込む (N はユーザー指定、デフォルト 10)
tail -n 10 .backlog/audit-log.jsonl 2>/dev/null
```

ログが存在しない、または空の場合は「audit-log.jsonl が見つかりません。先に `/backlog-manage audit` を実行してください」と報告して終了する。

#### Step 2: パターン分析

読み込んだログに対して以下の 4 つの指標を分析する。`check` の値は Phase 2 テーブルの `check_key` カラムを参照。

```bash
# パターン分析テンプレート
python3 -c "
import json, sys
from collections import Counter

entries = []
for line in sys.stdin:
    if line.strip():
        entries.append(json.loads(line))

if not entries:
    print('No audit logs found')
    sys.exit(0)

# 再発チェック: 同一 check が 3 回以上 fail
fail_counts = Counter()
unpatched_counts = Counter()
for e in entries:
    for f in e.get('findings', []):
        if f['status'] == 'fail':
            fail_counts[f['check']] += 1
            if not f.get('patched', False):
                unpatched_counts[f['check']] += 1

recurring = {k: v for k, v in fail_counts.items() if v >= 3}

# スコア推移
scores = [e['score'] for e in entries if 'score' in e]

# 全パス判定
all_pass_streak = 0
for e in reversed(entries):
    s = e.get('score', {})
    if s.get('passed') == s.get('total') and s.get('total', 0) > 0:
        all_pass_streak += 1
    else:
        break

print(json.dumps({
    'recurring': recurring,
    'unpatched': dict(unpatched_counts),
    'scores': scores,
    'all_pass_streak': all_pass_streak,
    'total_runs': len(entries)
}, ensure_ascii=False, indent=2))
"
```

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

4桁hexはランダム生成:
```bash
python3 -c "import random; print(f'{random.randint(0,65535):04x}')"
```

## Important Rules

- JSONLファイルを `Write` ツールで上書きしない(既存行が消える)
- `echo >> file.jsonl` で追記する
- done ファイルは append-only(エントリを削除しない)
- MDサマリは自動生成のため手動編集しない
- 全操作後に `regenerate-md` を自動実行する
- JSON文字列内の日本語は `ensure_ascii=False` で出力する
