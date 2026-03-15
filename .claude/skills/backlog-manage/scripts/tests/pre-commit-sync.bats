#!/usr/bin/env bats

setup() {
  TEST_DIR="$(mktemp -d)"
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  SCRIPT="${SCRIPT_DIR}/pre-commit-sync.sh"

  # テスト用 git repo を作成
  git -C "$TEST_DIR" init --quiet
  git -C "$TEST_DIR" config user.email "test@example.com"
  git -C "$TEST_DIR" config user.name "Test"

  # backlog-cli モックバイナリを配置
  MOCK_CLI_DIR="$TEST_DIR/.claude/skills/backlog-manage/cli/bin"
  mkdir -p "$MOCK_CLI_DIR"
  cat > "$MOCK_CLI_DIR/backlog-cli" <<'MOCK'
#!/usr/bin/env bash
# Mock backlog-cli: regenerate-md は成功する
if [[ "$*" == *"regenerate-md"* ]]; then
  exit 0
fi
exit 1
MOCK
  chmod +x "$MOCK_CLI_DIR/backlog-cli"

  # .backlog ディレクトリと MD ファイルを用意
  mkdir -p "$TEST_DIR/.backlog"
  printf '' > "$TEST_DIR/.backlog/README.md"
  printf '' > "$TEST_DIR/.backlog/TASKS.md"
  printf '' > "$TEST_DIR/.backlog/IDEAS.md"
  printf '' > "$TEST_DIR/.backlog/ISSUES.md"

  # 初期コミット
  git -C "$TEST_DIR" add -A
  git -C "$TEST_DIR" commit --quiet -m "initial"
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "JSONL ステージ時に MD が再生成されステージングされる" {
  # JSONL を変更してステージ
  printf '{"id":"task-1"}\n' > "$TEST_DIR/.backlog/tasks.jsonl"
  git -C "$TEST_DIR" add .backlog/tasks.jsonl

  run bash -c "cd '$TEST_DIR' && '$SCRIPT'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"MD summaries synced"* ]]
}

@test "バイナリが存在せず task も失敗する場合はエラー終了" {
  # モックバイナリを削除
  rm "$TEST_DIR/.claude/skills/backlog-manage/cli/bin/backlog-cli"

  # task コマンドのモック (失敗する)
  MOCK_TASK="$TEST_DIR/mock-task"
  cat > "$MOCK_TASK" <<'MOCK'
#!/usr/bin/env bash
exit 1
MOCK
  chmod +x "$MOCK_TASK"

  run bash -c "cd '$TEST_DIR' && PATH='$(dirname "$MOCK_TASK"):$PATH' '$SCRIPT'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Failed to build backlog-cli"* ]]
}

@test "regenerate-md 失敗時はエラー終了" {
  # regenerate-md が失敗するモックに差し替え
  cat > "$TEST_DIR/.claude/skills/backlog-manage/cli/bin/backlog-cli" <<'MOCK'
#!/usr/bin/env bash
exit 1
MOCK
  chmod +x "$TEST_DIR/.claude/skills/backlog-manage/cli/bin/backlog-cli"

  run bash -c "cd '$TEST_DIR' && '$SCRIPT'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"regenerate-md failed"* ]]
}
