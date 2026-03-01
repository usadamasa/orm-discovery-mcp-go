---
paths:
  - ".backlog/**"
---

# Backlog Data Practices

`.backlog/` ディレクトリ内のデータファイルを操作する際のルール。

## JSONL ファイル操作

- **追記のみ**: `echo '{json}' >> file.jsonl` で追記する。`Write` ツールでファイル全体を上書きしない(既存行が消える)
- **削除操作**: `grep -v` + tmpファイル + `mv` パターンで行を削除する
- **JSON整合性**: 1行 = 1 JSON オブジェクト。改行を含めない

## done ファイル

- `*.done.jsonl` は **append-only**。エントリを削除・編集しない
- 完了/解決時にアクティブファイルから done ファイルへ移動する

## MD サマリファイル

- `README.md`, `TASKS.md`, `IDEAS.md`, `ISSUES.md` は **自動生成**
- 手動編集しない。`/backlog-manage regenerate-md` で再生成する

## ID 規則

- 形式: `{type}-{YYYYMMDD}-{4桁hex}` (例: `task-20260301-a3f2`)
- type: `task`, `idea`, `issue`
