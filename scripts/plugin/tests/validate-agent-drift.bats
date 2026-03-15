#!/usr/bin/env bats

setup() {
  TEST_DIR="$(mktemp -d)"
  mkdir -p "$TEST_DIR/internal/server"

  # internal/server/server.go フィクスチャ
  cat > "$TEST_DIR/internal/server/server.go" <<'GO'
	searchTool := &mcp.Tool{
		Name:  "oreilly_search_content",
	}
	askQuestionTool := &mcp.Tool{
		Name:  "oreilly_ask_question",
	}
	reauthTool := &mcp.Tool{
		Name:  "oreilly_reauthenticate",
	}
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-details/{product_id}",
		},
	)
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-toc/{product_id}",
		},
	)
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-chapter/{product_id}/{chapter_name}",
		},
	)
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://answer/{question_id}",
		},
	)
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-details/{product_id}",
		},
	)
GO

  # internal/server/history_resources.go フィクスチャ
  cat > "$TEST_DIR/internal/server/history_resources.go" <<'GO'
	s.server.AddResource(
		&mcp.Resource{
			URI:         "orm-mcp://history/recent",
		},
	)
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/search{?keyword,type}",
		},
	)
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/{id}",
		},
	)
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/{id}/full",
		},
	)
GO

  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
}

teardown() {
  rm -rf "$TEST_DIR"
}

create_agent_md() {
  local file="$TEST_DIR/agent.md"
  cat > "$file" <<'MD'
## Available Tools

- **oreilly_search_content**: Search
- **oreilly_ask_question**: Ask

## Available Resources

- `oreilly://book-details/{product_id}`
- `oreilly://book-toc/{product_id}`
- `oreilly://book-chapter/{product_id}/{chapter_name}`
- `oreilly://answer/{question_id}`
- `orm-mcp://history/recent`
- `orm-mcp://history/search{?keyword,type}`
- `orm-mcp://history/{id}`
- `orm-mcp://history/{id}/full`
MD
  printf '%s' "$file"
}

@test "全ツール・リソースが一致すると成功する" {
  agent_md="$(create_agent_md)"
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-agent-drift.sh' '$agent_md' 'oreilly_reauthenticate'"
  [ "$status" -eq 0 ]
  [[ "$output" == *"in sync"* ]]
}

@test "ツールが欠落しているとドリフト検出する" {
  agent_md="$TEST_DIR/agent.md"
  cat > "$agent_md" <<'MD'
## Available Tools

- **oreilly_search_content**: Search

## Available Resources

- `oreilly://book-details/{product_id}`
- `oreilly://book-toc/{product_id}`
- `oreilly://book-chapter/{product_id}/{chapter_name}`
- `oreilly://answer/{question_id}`
- `orm-mcp://history/recent`
- `orm-mcp://history/search{?keyword,type}`
- `orm-mcp://history/{id}`
- `orm-mcp://history/{id}/full`
MD
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-agent-drift.sh' '$agent_md' 'oreilly_reauthenticate'"
  [ "$status" -eq 1 ]
  [[ "$output" == *'Tool "oreilly_ask_question"'* ]]
}

@test "リソースが欠落しているとドリフト検出する" {
  agent_md="$TEST_DIR/agent.md"
  cat > "$agent_md" <<'MD'
## Available Tools

- **oreilly_search_content**: Search
- **oreilly_ask_question**: Ask

## Available Resources

- `oreilly://book-details/{product_id}`
- `oreilly://book-toc/{product_id}`
- `oreilly://answer/{question_id}`
- `orm-mcp://history/recent`
- `orm-mcp://history/search{?keyword,type}`
- `orm-mcp://history/{id}`
- `orm-mcp://history/{id}/full`
MD
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-agent-drift.sh' '$agent_md' 'oreilly_reauthenticate'"
  [ "$status" -eq 1 ]
  [[ "$output" == *'Resource "oreilly://book-chapter/{product_id}/{chapter_name}"'* ]]
}

@test "除外リストのツールは検出しない" {
  agent_md="$(create_agent_md)"
  # oreilly_reauthenticate は agent.md にないが除外されているので成功する
  run bash -c "cd '$TEST_DIR' && '$SCRIPT_DIR/validate-agent-drift.sh' '$agent_md' 'oreilly_reauthenticate'"
  [ "$status" -eq 0 ]
  [[ "$output" != *"oreilly_reauthenticate"* ]]
}
