#!/usr/bin/env bash
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
readonly ROOT

BACKLOG_DIR="${ROOT}/.backlog"
readonly BACKLOG_DIR

BINARY="${ROOT}/.claude/skills/backlog-manage/cli/bin/backlog-cli"
readonly BINARY

# backlog-cli バイナリの存在確認。なければビルド。
if [ ! -f "${BINARY}" ]; then
  printf 'backlog-cli binary not found. Building...\n'
  if ! task backlog:build; then
    printf 'ERROR: Failed to build backlog-cli. Run: task backlog:build\n' >&2
    exit 1
  fi
fi

# MD サマリを再生成
if ! "${BINARY}" --dir "${BACKLOG_DIR}" regenerate-md; then
  printf 'ERROR: regenerate-md failed.\n' >&2
  exit 1
fi

# .backlog/ 配下に変更があればステージングに追加
if ! git -C "${ROOT}" diff --quiet -- "${BACKLOG_DIR}"; then
  git -C "${ROOT}" add .backlog/
fi

printf 'backlog: MD summaries synced.\n'
