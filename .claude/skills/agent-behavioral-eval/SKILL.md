---
name: agent-behavioral-eval
description: >
  oreilly-researcher エージェントの振る舞い評価。BFS/DFS モード選択、
  Q&A ワークフロー、出力フォーマット準拠、VOC 収集の 4 シナリオ + VOC 横断観察で
  エージェント行動品質を検証する。
  トリガー: 「エージェント行動評価」「agent behavioral eval」「振る舞い評価」
  「エージェント品質テスト」
user_invocable: true
---

# agent-behavioral-eval

oreilly-researcher エージェントの振る舞い規約 (BFS/DFS 選択、ワークフロー順序、出力形式、引用、VOC 収集) をライブシナリオで検証する。

## Phase 2A との境界

- **2A (Functional Correctness)**: ツールが動くか (search がデータを返す、resources が情報を返す)
- **このスキル (Behavioral Fidelity)**: エージェントが正しく振る舞うか (モード選択、ワークフロー順序、出力形式、引用)

## Context

- Branch: !`git branch --show-current`
- VOC issues before: Pre-flight で `gh issue list` を実行して取得
- Recent commits: !`git log --oneline -5`
- Backlog: Pre-flight で `backlog-cli list` を実行して取得
- Last eval: Pre-flight で audit-log.jsonl を確認
- Key dependencies: Pre-flight で go.mod を確認

## Pre-flight: 認証確認

`orm-discovery-mcp-go:oreilly-researcher` subagent で `oreilly_reauthenticate` を呼び出す。

- **成功** → シナリオ B1-B4 を開始
- **失敗** → 全シナリオを SKIP として終了

認証成功後、B2 用の事前状態を記録:
- MEMORY.md の現在の行数と最終行の内容を記録 (`wc -l` + `tail -1`)

## Prompt Generation

Context セクションの情報を元に、B1-B4 の検証プロンプトを動的に生成する。

### 生成ルール

| シナリオ | 文型制約 |
|---------|---------|
| B1 | 「〜に関する本を探してください」(単一トピック発見) |
| B2 | 「AとBを詳しく比較してください」(2技術比較) |
| B3 | 「〜のベストプラクティスとは?」(技術的質問) |
| B4 | 「AとBの書籍を N 件ずつリストアップ」(リスト要求) |

### 選択基準 (優先順)

1. Active tasks のタイトルに含まれる技術名
2. Recent commits で言及された技術
3. Key dependencies のライブラリ名 (フォールバック)
4. 前回 fail シナリオには前回と異なるトピック
5. B1-B4 でトピック分散、O'Reilly 検索可能な公開技術のみ

### フォールバック (バックログ空の場合)

- B1: 「MCP (Model Context Protocol) に関する本を探してください」
- B2: 「Go と Rust の書籍を詳しく比較してください」
- B3: 「ブラウザ自動化のベストプラクティスとは?」
- B4: 「Go と ChromeDP の書籍を 5 件ずつリストアップ」

### 出力形式

生成したプロンプトと選択理由 (source) を以下の形式で記録してからシナリオへ進む:

```
Generated Prompts:
- B1: "{prompt}" (source: {reason})
- B2: "{prompt}" (source: {reason})
- B3: "{prompt}" (source: {reason})
- B4: "{prompt}" (source: {reason})
```

## Scenario B1: BFS + Quick Research

**プロンプト**: Prompt Generation セクションで生成した B1 プロンプトを使用する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で実行。

### 成功条件 (L1, L3, L4)

| チェック | 条件 | カバー項目 |
|---------|------|----------|
| BFS モード使用 | `mode="bfs"` で search が呼ばれた | L1 |
| 軽量レスポンス | 結果が id/title/authors のみ (description/chapters 等なし) | L3 |
| book-details チェーン | search 後に `oreilly://book-details/{product_id}` にアクセスした | L4 |

### 判定

- 3/3 → PASS
- 1-2/3 → WARN (部分的準拠)
- 0/3 → FAIL

## Scenario B2: DFS + Deep Research + MEMORY

**プロンプト**: Prompt Generation セクションで生成した B2 プロンプトを使用する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で実行。

### 成功条件 (L2, L6)

| チェック | 条件 | カバー項目 |
|---------|------|----------|
| DFS モード使用 | `mode="dfs"` または `summarize=true` で search が呼ばれた | L2 |
| 比較分析出力 | Summary Template のマーカー (`## Research Summary`, `### Key Findings`) が出力に含まれる | - |
| MEMORY 更新 | 実行後の MEMORY.md 行数が Pre-flight 記録より増加、または新しいエントリ (日付・トピック) が追加されている | L6 |

### 判定

- DFS 使用 → 必須 (不使用で FAIL)
- 比較分析 + MEMORY 更新 → PASS、片方のみ → WARN

## Scenario B3: Q&A Workflow + Citation

**プロンプト**: Prompt Generation セクションで生成した B3 プロンプトを使用する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で実行。

### 成功条件 (L5)

| チェック | 条件 | カバー項目 |
|---------|------|----------|
| ask_question 呼び出し | `oreilly_ask_question` が最初に呼ばれた | L5 |
| search フォローアップ | ask_question 後に `oreilly_search_content` が呼ばれた | L5 |
| Citation 形式 | `[Title] by [Author(s)], O'Reilly Media` の形式で引用が含まれる | - |

### 判定

- ask_question → search チェーン成立 + Citation あり → PASS
- チェーン成立だが Citation なし → WARN
- ask_question 未使用 → FAIL

## Scenario B4: Output Format Compliance

**プロンプト**: Prompt Generation セクションで生成した B4 プロンプトを使用する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で実行。

### 成功条件

| チェック | 条件 |
|---------|------|
| Quick Discovery Template | `## Available Resources` マーカーが出力に含まれる |
| リスト形式 | 番号付きリスト (`1. **[Title]**`) が含まれる |
| Summary Template 不在 | `## Research Summary` が出力に含まれない (リスト要求に Summary は不適切) |

### 判定

- Quick Discovery Template 準拠 → PASS
- リスト形式だがマーカー不在 → WARN
- Summary Template で回答 → FAIL

## VOC Cross-Check (横断観察)

B1-B4 完了後に実行。

### 手順

1. 現在の VOC Issue 数を取得:
   ```bash
   gh issue list -R usadamasa/orm-discovery-mcp-go --label voc --json number --jq length
   ```

2. Pre-flight で記録した件数と比較

3. 新規 Issue がある場合:
   - PII チェック: Issue body に email, password, token 等が含まれないか確認 (L10)
   - ラベル確認: `voc,bug` または `voc,enhancement` または `voc,question` が付いているか (L13, L14)
   - 重複チェック: 同一タイトルの Issue が複数作られていないか (L11)

### 判定 (L8-L11, L13-L14)

| 条件 | ステータス |
|------|----------|
| VOC 増加あり + PII なし + ラベル正常 | PASS |
| VOC 増加なし (機会があったか不明) | WARN |
| PII 検出 | FAIL |
| ラベル不備 | WARN |

## Result Report

```markdown
## Agent Behavioral Evaluation Report

| Scenario | Status | Prompt Used | Details |
|----------|--------|-------------|---------|
| B1: BFS + Quick Research | PASS/WARN/FAIL/SKIP | {generated prompt} | {details} |
| B2: DFS + Deep Research | PASS/WARN/FAIL/SKIP | {generated prompt} | {details} |
| B3: Q&A Workflow | PASS/WARN/FAIL/SKIP | {generated prompt} | {details} |
| B4: Output Format | PASS/WARN/FAIL/SKIP | {generated prompt} | {details} |
| VOC: Cross-Check | PASS/WARN/SKIP | - | {details} |

### Overall: {PASS/WARN/FAIL/SKIP}

### Coverage
- L1 BFS が初期探索で使われる: {status}
- L2 DFS が深掘り調査で使われる: {status}
- L3 BFS レスポンスが軽量 (id/title/authors のみ): {status}
- L4 BFS 結果から book-details へチェーンする: {status}
- L5 Q&A ワークフロー (ask→search チェーン): {status}
- L6 調査後に MEMORY.md が更新される: {status}
- L8 VOC が Issue として記録される: {status}
- L9 VOC 記録がメインワークフローを阻害しない: {status}
- L10 VOC Issue に PII が含まれない: {status}
- L11 VOC Issue が重複作成されない: {status}
- L13 バグ系 VOC に `voc,bug` ラベルが付く: {status}
- L14 改善系 VOC に `voc,enhancement` ラベルが付く: {status}

### Excluded (not testable in single scenario)
- L7: MEMORY compression (threshold-dependent, covered by D4 static check)
- L12: Session Finalization (requires session end observation)

### Recommended Actions
- [FAIL items]: {specific remediation}
```

### Overall 判定ルール

- 全 PASS → PASS
- FAIL なし + WARN あり → WARN
- FAIL 1 件以上 → FAIL
- 全 SKIP (認証失敗) → SKIP

## Execution Notes

- 各シナリオは `orm-discovery-mcp-go:oreilly-researcher` subagent で実行する
- B1 と B2 は独立のため並列実行可能 (グループ A)
- B3 は A と独立 (グループ B)
- B4 は A/B と独立 (グループ C)
- VOC 横断は B1-B4 完了後に実行
- 認証失敗時は全シナリオ SKIP で即終了

## Related Skills

- **mcp-quality-eval**: このスキルを D3 (Behavioral Fidelity) として委譲呼び出しする
- **dogfood-verify**: 機能テスト (ツールが動くか) を担当
