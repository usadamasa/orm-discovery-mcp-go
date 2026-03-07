#!/usr/bin/env bash
set -euo pipefail

# マーケットプレイスとプラグイン設定の統合検証を行う。

printf '=== JSON形式検証 ===\n'
python3 -m json.tool < .claude-plugin/marketplace.json > /dev/null && printf '✓ marketplace.json は有効なJSON形式です\n'
python3 -m json.tool < .claude-plugin/plugin.json > /dev/null && printf '✓ plugin.json は有効なJSON形式です\n'

printf '\n=== 必須フィールド検証 ===\n'
if grep -q '"name"' .claude-plugin/marketplace.json && \
   grep -q '"owner"' .claude-plugin/marketplace.json && \
   grep -q '"plugins"' .claude-plugin/marketplace.json; then
  printf '✓ marketplace.json: 必須フィールドが存在します\n'
else
  printf '✗ marketplace.json: 必須フィールドが不足しています (name, owner, plugins)\n'
  exit 1
fi

printf '\n=== バージョン同期検証 ===\n'
MARKETPLACE_VER=$(python3 -c "import json; print(json.load(open('.claude-plugin/marketplace.json'))['metadata']['version'])")
readonly MARKETPLACE_VER
PLUGINS_VER=$(python3 -c "import json; print(json.load(open('.claude-plugin/marketplace.json'))['plugins'][0]['version'])")
readonly PLUGINS_VER
if [ "$MARKETPLACE_VER" = "$PLUGINS_VER" ]; then
  printf '✓ バージョンが同期しています: %s\n' "$MARKETPLACE_VER"
else
  printf '✗ バージョン不一致: metadata=%s, plugins[]=%s\n' "$MARKETPLACE_VER" "$PLUGINS_VER"
  exit 1
fi

printf '\n=== エージェント定義検証 ===\n'
for agent in $(python3 -c "import json; agents=json.load(open('.claude-plugin/plugin.json')).get('agents',[]); [print(a) for a in agents]"); do
  if [ ! -f "$agent" ]; then
    printf '✗ エージェントファイルが見つかりません: %s\n' "$agent"
    exit 1
  fi
  if head -1 "$agent" | grep -q "^---"; then
    printf '✓ エージェント Frontmatter 存在: %s\n' "$agent"
  else
    printf '✗ エージェント Frontmatter が不足: %s\n' "$agent"
    exit 1
  fi
done

printf '\n=== MCP サーバー設定検証 ===\n'
if python3 -c "import json; servers=json.load(open('.claude-plugin/plugin.json')).get('mcpServers',{}); exit(0 if servers else 1)" 2>/dev/null; then
  printf '⚠ MCP サーバーが設定されています。バイナリが PATH 上にあることを確認してください\n'
  python3 -c "import json; servers=json.load(open('.claude-plugin/plugin.json')).get('mcpServers',{}); [print(f'  - {k}: {v.get(\"command\",\"\")}') for k,v in servers.items()]"
fi

printf '\n=== セキュリティ検証 ===\n'
if grep -rE "(password|secret|api_key)\s*[=:]\s*['\"][^\$]" .claude-plugin/ 2>/dev/null | grep -v '\${'; then
  printf '✗ ハードコードされた認証情報が検出されました\n'
  exit 1
else
  printf '✓ ハードコードされた認証情報はありません\n'
fi

printf '\n✓ マーケットプレイス検証が完了しました\n'
