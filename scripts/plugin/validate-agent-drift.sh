#!/usr/bin/env bash
set -euo pipefail

# エージェント定義と server.go の登録内容のドリフトを検出する。
# 使用方法: ./scripts/validate-agent-drift.sh [AGENT_MD] [EXCLUDE_TOOLS]
#   AGENT_MD:       エージェント定義ファイルパス (default: plugins/agents/oreilly-researcher.md)
#   EXCLUDE_TOOLS:  除外するツール名 (スペース区切り, default: oreilly_reauthenticate)

AGENT_MD="${1:-plugins/agents/oreilly-researcher.md}"
EXCLUDE_TOOLS="${2:-oreilly_reauthenticate}"
DRIFT=0

# 1. server.go + history_resources.go からツール名を抽出
extract_server_tools() {
  grep -h 'Name:.*"oreilly_' server.go | sed 's/.*"\(oreilly_[^"]*\)".*/\1/' | sort
}

# 2. 除外リストを適用
filter_excluded_tools() {
  local tools="$1"
  local excludes="$2"
  for exclude in $excludes; do
    tools=$(printf '%s\n' "$tools" | grep -v "^${exclude}$" || true)
  done
  printf '%s\n' "$tools"
}

# 3. server.go + history_resources.go からリソース URI を抽出
extract_server_resources() {
  grep -hE '(URI:|URITemplate:).*"(oreilly://|orm-mcp://)' server.go history_resources.go \
    | sed 's/.*"\([^"]*\)".*/\1/' | sort -u
}

# 4. エージェント md からツール名を抽出
extract_agent_tools() {
  grep -oE 'oreilly_[a-z_]+' "$AGENT_MD" | sort -u
}

# 5. エージェント md からリソース URI を抽出
extract_agent_resources() {
  grep -oE '(oreilly|orm-mcp)://[^ )`]+' "$AGENT_MD" | sort -u
}

SERVER_TOOLS=$(filter_excluded_tools "$(extract_server_tools)" "$EXCLUDE_TOOLS")
SERVER_RESOURCES=$(extract_server_resources)
AGENT_TOOLS=$(extract_agent_tools)
AGENT_RESOURCES=$(extract_agent_resources)

# ドリフト検出: server にあるのに agent にないツール
for tool in $SERVER_TOOLS; do
  if ! printf '%s\n' "$AGENT_TOOLS" | grep -q "^${tool}$"; then
    printf 'WARNING: Tool "%s" is registered in server.go but missing from %s\n' "$tool" "$AGENT_MD"
    DRIFT=1
  fi
done

# ドリフト検出: server にあるのに agent にないリソース
for resource in $SERVER_RESOURCES; do
  if ! printf '%s\n' "$AGENT_RESOURCES" | grep -qF "$resource"; then
    printf 'WARNING: Resource "%s" is registered in server but missing from %s\n' "$resource" "$AGENT_MD"
    DRIFT=1
  fi
done

if [ "$DRIFT" -eq 1 ]; then
  printf '\nAgent definition drift detected. Update %s to match server registrations.\n' "$AGENT_MD"
  exit 1
fi

printf 'Agent definition is in sync with server registrations.\n'
