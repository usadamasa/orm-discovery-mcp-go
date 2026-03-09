---
name: mcp-quality-eval
description: >
  MCP サーバーの統合品質評価ワークフロー。CI/機能テスト/コンテキスト効率/
  スキル同期/Agent品質/バックログ健全性の6ディメンションを一括評価し、
  統合スコアカードを出力する。失敗項目はバックログに自動登録する。
  トリガー: 「品質評価」「quality check」「MCP評価」「品質どう?」
  「全体チェック」「quality eval」
user_invocable: true
---

# mcp-quality-eval

既存の品質関連スキル (dogfood-verify, dogfood-improve, context-efficiency-improve, backlog-manage) をオーケストレーションし、Agent 品質評価を加えた統合品質評価ワークフロー。

## Context

- Branch: !`git branch --show-current`
- Git status: !`git status --short`
- MCP tools registered: Phase 0 で server.go を確認
- Backlog items: Phase 0 で .backlog/*.jsonl を確認

## Phase 0: Pre-flight

Context セクションの情報を確認し、評価に必要な前提条件を検証する。

- [ ] プロジェクトルートにいること (`server.go` が存在)
- [ ] `backlog-cli` バイナリが存在すること (なければ自動ビルド)

```bash
ls server.go
task backlog:build
```

## Phase 1: CI Gate [BLOCKING]

```bash
task ci
```

- **PASS** -> Phase 2 へ
- **FAIL** -> 即停止。CI 修正に注力する (CI 優先ポリシー)

Phase 1 の結果を記録: `ci_status = PASS | FAIL`

## Phase 2: Parallel Dimension Evaluation

4 つのディメンションを評価する。静的チェックは subagent で並列化、ライブテストはメインエージェントが直接実行する。

**並列化戦略**:
- 2A はメインエージェントが直接実行 (MCP ツール権限の都合)
- 2B + 2C + 2D(D1,D2,D4) は subagent で並列実行可 (全て静的チェック)
- 2A 完了後に 2D(D3 ライブシナリオ = agent-behavioral-eval) を開始

### 2A: Functional Correctness

dogfood-verify の Phase 3-4 を委譲する。

**重要**: メインエージェントが直接 MCP ツールを呼び出す。subagent 経由だと Claude の permission system で MCP ツール呼び出しが deny されるため。

以下を順に実行:

1. **認証確認**: `oreilly_reauthenticate` ツールを呼び出す
   - 失敗 -> 2A 全体を FAIL (認証エラー) とし、残りの 2A テストをスキップ
2. **oreilly_search_content**: 「Docker」で 5 件検索 (BFS モード)
   - 成功条件: 結果が返り、認証エラーなし
3. **oreilly_ask_question**: 「What is Docker?」(最大待機 60 秒)
   - 成功条件: answer フィールドが存在 (タイムアウトは WARNING)
4. **Resources チェーン**: 2 の結果から product_id を取得し `ReadMcpResourceTool` で book-details にアクセス
   - 成功条件: タイトル・著者情報が返る
5. **Prompts**: learn-technology (technology="Go") を JSON-RPC で確認
   - `bin/orm-discovery-mcp-go` に initialize + prompts/get を送信し、結果を検証
   - 成功条件: description と messages が返る

各テストの結果を記録: `func_results = {test_name: PASS|FAIL|WARN|SKIP}`

### 2B: Context Efficiency

コンテキスト効率を計測する (改善はしない)。

```bash
task measure:context-efficiency
```

続けてガードレールテスト:

```bash
go test -v -run TestToolDescriptionSizes ./...
```

結果を記録:
- `ctx_efficiency_report`: 計測レポート出力
- `ctx_guardrail`: PASS (全テスト通過) | FAIL (超過あり)

### 2C: Skill Sync

dogfood-improve の Step 1-3 を委譲する (差分検出のみ、修正はしない)。

1. **MCP 登録状態の収集**: server.go, history_resources.go, prompts.go から grep
2. **dogfood-verify スキル解析**: Phase 4 のスモークテスト項目を抽出
3. **差分検出**: 新規/削除されたツール・リソース・プロンプトを特定

結果を記録:
- `skill_sync = PASS` (差分なし) | `WARN` (差分あり、リスト添付)

### 2D: Agent Quality

4 つのサブ評価で構成する。

#### D1: Capability Coverage (静的)

```bash
task plugin:validate:agent-drift
```

- PASS: スクリプトが exit 0
- FAIL: ドリフト検出

#### D2: Agent Definition Completeness (静的)

`plugins/agents/oreilly-researcher.md` を静的に検査する。

**フロントマター検査**:

```bash
head -40 plugins/agents/oreilly-researcher.md
```

- `model: inherit` が設定されていること
- `memory: user` が設定されていること

**必須セクション検査** (9 セクション):

| # | セクション | grep パターン |
|---|----------|--------------|
| 1 | Available Tools | `## Available Tools` |
| 2 | Available Resources | `## Available Resources` |
| 3 | BFS/DFS Mode Selection Criteria | `## BFS/DFS Mode Selection Criteria` |
| 4 | Research Workflows | `## Research Workflows` |
| 5 | Output Format | `## Output Format` |
| 6 | Citation Requirements | `## Citation Requirements` |
| 7 | Memory Management | `## Memory Management` |
| 8 | VOC Collection | `## VOC Collection` |
| 9 | Session Finalization | `## Session Finalization` |

**追加チェック**:

- Citation フォーマット定義: `O'Reilly Media` と `Author` が含まれること
- VOC ラベル設定: `voc,bug` と `voc,enhancement` が含まれること
- Output テンプレート: `Summary Template` と `Quick Discovery Template` が含まれること

**判定**:

- フロントマター OK + 9 セクション全存在 + 追加チェック全 OK → PASS
- 1 項目以上欠損 → FAIL (欠損リスト添付)

#### D3: Behavioral Fidelity (ライブ) → agent-behavioral-eval に委譲

**前提**: 2A の認証確認が成功していること。2A が FAIL の場合、D3 全体を SKIP とする。

`agent-behavioral-eval` スキルを呼び出し、B1-B4 + VOC 横断観察を実行する。

結果を記録: `agent_behavioral = {B1: status, B2: status, B3: status, B4: status, VOC: status}`

#### D4: Memory Hygiene (静的)

auto memory の MEMORY.md 行数を確認する。Read tool で MEMORY.md を読み、行数をカウントする。

- PASS: 200 行未満
- WARN: 200 行以上

## Phase 3: Backlog Integration

Phase 2 で FAIL となった項目をバックログに登録する。

### Step 1: 失敗項目の収集

Phase 2 の全ディメンションから FAIL 項目を収集する。

### Step 2: 重複チェック

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli list --type issue
```

既存 issue の title と照合し、同一内容の issue が既にあればスキップする。

### Step 3: Issue 作成

重複のない FAIL 項目について `add-issue` で登録する。

severity 自動判定:
- CI / Functional Correctness の失敗 -> `high`
- Context Efficiency / Skill Sync の失敗 -> `medium`
- Agent Quality の失敗 -> `low`

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli add issue \
  --title "{失敗項目の説明}" \
  --description "mcp-quality-eval Phase 2 で検出: {詳細}" \
  --severity "{severity}" \
  --tags "quality-eval"
```

### Step 4: Backlog Health Check

`backlog-cli audit --run` を実行し、バックログの健全性を自動チェックする。

```bash
.claude/skills/backlog-manage/cli/bin/backlog-cli audit --run
```

このコマンドは以下の 7 チェックを実行し、結果を audit-log.jsonl に自動記録する:
- JSONL 整合性 (tasks/ideas/issues)
- アイデア滞留 (30 日超)
- 残留バックアップファイル
- MD サマリ同期
- 未追跡ハンドオフ
- 未連携 GH Issue (gh 未インストール時は skip)
- MEMORY 重複検出

結果を記録: `backlog_health = {passed: N, total: M}`

## Phase 4: Unified Scorecard

全ディメンションの結果を統合レポートとして出力する。

```markdown
## MCP Quality Evaluation Scorecard

| Dimension | Score | Status | Details |
|-----------|-------|--------|---------|
| CI Gate | 1/1 | PASS | task ci passed |
| Functional Correctness | N/M | PASS/WARN/FAIL | {failed items} |
| Context Efficiency | N/M | PASS/WARN/FAIL | {guardrail results} |
| Skill Sync | N/M | PASS/WARN | {drift items} |
| Agent Quality | N/M | PASS/WARN/FAIL | D1/D2/D3/D4 details |
| Backlog Health | N/M | PASS/WARN/FAIL | audit score |

**Agent Quality 集約ルール**:
- D1-D4 の各サブ評価を個別に判定
- FAIL が 1 件以上 → Agent Quality = FAIL
- FAIL なし + WARN あり → Agent Quality = WARN
- 全 PASS → Agent Quality = PASS
- Score: PASS 数 / 全サブ評価数 (D3 は B1-B4+VOC の 5 項目を展開)

### Overall: N/M checks passed

### New Issues Created: N items (or "None")

### Recommendation:
- [ALL PASS] `/finalize-pr` の実行を推奨
- [FAIL あり] 修正必要: {リスト}
```

## Phase 4.5: Eval Log Recording

Phase 4 のスコアカード出力後、`backlog-cli audit log-entry` で結果を `.backlog/audit-log.jsonl` に追記する。
`backlog-cli retrospective` が eval パターンを分析できるようにする。

### 記録コマンド

`backlog-cli audit log-entry` は `--findings` (JSON 配列) を受け取り、`id`, `run_at`, `score` を自動生成して audit-log.jsonl に追記する。stdout にエントリ ID を出力する。

```bash
backlog-cli audit log-entry \
  --findings '[
    {"check":"ci_gate","status":"pass","detail":"all CI checks passed"},
    {"check":"func_auth","status":"pass","detail":"cookie auth OK"},
    {"check":"func_search","status":"pass","detail":"BFS/DFS both returned results"},
    {"check":"func_ask","status":"warn","detail":"slow response 8.2s"},
    {"check":"func_resources","status":"pass","detail":"book-details, book-toc OK"},
    {"check":"func_prompts","status":"skip","detail":"MCP Prompts not available in subagent"},
    {"check":"ctx_guardrail","status":"pass","detail":"tools/list 2.1KB < 3KB"},
    {"check":"skill_sync","status":"pass","detail":"no drift"},
    {"check":"agent_drift","status":"pass","detail":"no drift"},
    {"check":"agent_definition","status":"pass","detail":"frontmatter valid"},
    {"check":"agent_behavioral_b1","status":"pass","detail":"BFS mode selected correctly"},
    {"check":"agent_behavioral_b2","status":"pass","detail":"DFS deep-dive executed"},
    {"check":"agent_behavioral_b3","status":"pass","detail":"ask_question used for Q&A"},
    {"check":"agent_behavioral_b4","status":"pass","detail":"VOC issue created"},
    {"check":"agent_voc","status":"pass","detail":"no new VOC"},
    {"check":"memory_hygiene","status":"pass","detail":"no duplicates"},
    {"check":"backlog_health","status":"pass","detail":"7/7 checks passed"}
  ]' \
  --patch-actions '[]'
# => audit-20260309-a1b2  (自動生成された ID が stdout に出力される)
```

### findings の check_key 一覧

| check_key | ディメンション | 値の意味 |
|-----------|--------------|---------|
| `ci_gate` | CI | task ci の通過 |
| `func_auth` | Functional | 認証成功 |
| `func_search` | Functional | search ツール動作 |
| `func_ask` | Functional | ask_question 動作 |
| `func_resources` | Functional | Resource アクセス |
| `func_prompts` | Functional | Prompts (通常 skip) |
| `ctx_guardrail` | Context Efficiency | サイズガードレール |
| `skill_sync` | Skill Sync | スキル↔実装の同期 |
| `agent_drift` | Agent Quality | エージェント定義ドリフト |
| `agent_definition` | Agent Quality | Frontmatter 検証 |
| `agent_behavioral_b1`-`b4` | Agent Quality | 行動評価シナリオ |
| `agent_voc` | Agent Quality | VOC 検出 |
| `memory_hygiene` | Backlog Health | MEMORY 重複チェック |
| `backlog_health` | Backlog Health | audit --run 結果 (7 チェック) |

### 自己進化の流れ

1. mcp-quality-eval 実行ごとに `backlog-cli audit log-entry` でエントリ追記
2. `backlog-cli retrospective --json` で eval パターンを分析
3. 再発チェック (同一 check が 3 回以上 fail) → 評価観点の改善提案
4. 全パス継続 (5 回以上) → 新しいチェック項目の追加提案

## Phase 5: Proactive Actions

スコアカードの結果に応じて推奨アクションを提示する。

| 条件 | 推奨 |
|------|------|
| 全 PASS + PR あり | `/finalize-pr` の実行を推奨 |
| Skill Sync WARN | `/dogfood-improve` の実行を推奨 |
| Context Efficiency FAIL | `/context-efficiency-improve` の実行を推奨 |
| Functional FAIL | 個別の失敗項目に応じた修正を案内 |
| Backlog Health FAIL | `/backlog-manage audit` の実行を推奨 |

## Orchestration Rules

1. **Phase 1 は BLOCKING**: CI 失敗で即停止
2. **Phase 2 は並列化**: 2B/2C/2D(D1,D2,D4) は同時実行 (全て静的)。2A 完了後に D3 (agent-behavioral-eval) 開始。2A が FAIL (認証失敗) の場合、D3 全体を SKIP とする
3. **Phase 3 は Phase 2 完了後**: 全ディメンション結果が揃ってから実行
4. **既存スキルを呼び出す、再実装しない**: 各スキルのロジックを複製せず、委譲する
5. **計測のみ、改善しない**: このスキルは評価と報告に徹する。修正は個別スキルに委ねる

## Related Skills

- **dogfood-verify**: Phase 2A のソース (E2E スモークテスト)
- **dogfood-improve**: Phase 2C のソース (スキル同期チェック)
- **context-efficiency-improve**: Phase 2B のソース (トークン計測)
- **agent-behavioral-eval**: Phase 2D-D3 のソース (エージェント行動評価)
- **backlog-manage**: Phase 3 + Phase 4.5 のソース (バックログ統合 + eval ログ分析)
- **mcp-tool-progressive-disclosure**: 参照専用 (記述パターン集)
- **mcp-go-sdk-practices**: 参照専用 (SDK 実装リファレンス)
