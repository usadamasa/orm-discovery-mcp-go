---
name: backlog-manage
description: バックログ管理 - タスク/アイデア/イシューの追加・完了・一覧・MDサマリ再生成
user_invocable: true
---

# Backlog Management Skill

`.backlog/` ディレクトリのJSONLファイルを操作してバックログを管理する。

## Context

!grep -c . .backlog/tasks.jsonl 2>/dev/null || printf "0" | xargs printf "Active tasks: %s"
!grep -c . .backlog/ideas.jsonl 2>/dev/null || printf "0" | xargs printf ", Active ideas: %s"
!grep -c . .backlog/issues.jsonl 2>/dev/null || printf "0" | xargs printf ", Active issues: %s"

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
        print(f\"  [{e['priority']}] {e['id']}: {e['title']} ({e['status']})\")
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

1. ideas.jsonl から該当アイデアを取得
2. 新しいタスク/イシューエントリを作成 (`source: "idea"`, `source_ref: idea_id`)
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
