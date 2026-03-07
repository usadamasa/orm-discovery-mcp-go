#!/usr/bin/env bats

setup() {
  TEST_DIR="$(mktemp -d)"
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"

  # 正常なフィクスチャを作成
  mkdir -p "$TEST_DIR/.claude-plugin"
  mkdir -p "$TEST_DIR/plugins/agents"

  cat > "$TEST_DIR/.claude-plugin/marketplace.json" <<'JSON'
{
  "name": "test-plugin",
  "owner": { "name": "tester", "email": "test@example.com" },
  "metadata": { "description": "test", "version": "1.0.0" },
  "plugins": [{ "name": "test-plugin", "source": "./", "version": "1.0.0" }]
}
JSON

  cat > "$TEST_DIR/.claude-plugin/plugin.json" <<'JSON'
{
  "name": "test-plugin",
  "agents": ["./plugins/agents/test.md"],
  "mcpServers": { "test": { "command": "test-cmd" } }
}
JSON

  cat > "$TEST_DIR/plugins/agents/test.md" <<'MD'
---
name: test
description: test agent
---
Test content
MD
}

teardown() {
  rm -rf "$TEST_DIR"
}

@test "正常な設定で検証が成功する" {
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-marketplace.sh'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"マーケットプレイス検証が完了しました"* ]]
}

@test "不正なJSONで検証が失敗する" {
  printf '{invalid' > "$TEST_DIR/.claude-plugin/marketplace.json"
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-marketplace.sh'"
  [ "$status" -ne 0 ]
}

@test "バージョン不一致で検証が失敗する" {
  cat > "$TEST_DIR/.claude-plugin/marketplace.json" <<'JSON'
{
  "name": "test-plugin",
  "owner": { "name": "tester" },
  "metadata": { "description": "test", "version": "1.0.0" },
  "plugins": [{ "name": "test-plugin", "source": "./", "version": "2.0.0" }]
}
JSON
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-marketplace.sh'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"バージョン不一致"* ]]
}

@test "エージェントファイルが存在しないと失敗する" {
  rm "$TEST_DIR/plugins/agents/test.md"
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-marketplace.sh'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"エージェントファイルが見つかりません"* ]]
}

@test "Frontmatterが欠落していると失敗する" {
  printf 'No frontmatter here\n' > "$TEST_DIR/plugins/agents/test.md"
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-marketplace.sh'"
  [ "$status" -eq 1 ]
  [[ "$output" == *"Frontmatter が不足"* ]]
}
