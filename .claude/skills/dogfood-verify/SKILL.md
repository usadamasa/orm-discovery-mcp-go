---
name: dogfood-verify
description: >
  orm-discovery-mcp-goプロジェクトのフルフィードバックループ実行。
  task ci → task install → MCP全ツール検証 → 自己PRレビュー → 結果報告。
  手動トリガー (「ドッグフーディング」「dogfood」「verify」「検証」) または
  draft-pr 後のプロアクティブ呼び出しに対応。
---

# dogfood-verify

orm-discovery-mcp-go 自身をビルド・インストール・MCP ツールとして実行し、フィードバックループを回すスキル。

## トリガー条件

- **手動**: 「ドッグフーディング」「dogfood」「verify」「検証して」等
- **プロアクティブ**: `draft-pr` 完了後に自動的に呼び出す

## Context

- Current branch: !`git branch --show-current`
- Git status: !`git status --short`
- PR status: !`gh pr view --json number,title,state,headRefName 2>/dev/null || echo "No PR found"`
- Install path: !`which orm-discovery-mcp-go 2>/dev/null || echo "Not installed"`

## ワークフロー

### Phase 1: CI 実行 (ゲート)

```bash
task ci
```

- **成功** → Phase 2 へ
- **失敗** → 即停止。CI 優先ポリシーに従い、CI 修正に注力する

### Phase 2: インストール

```bash
task install
```

インストール後、PATH 上の存在を確認:

```bash
which orm-discovery-mcp-go
```

見つからない場合は `$GOPATH/bin` が PATH に含まれているか確認を案内して停止。

### Phase 3: MCP サーバー反映確認

軽量な MCP ツール呼び出しで、セッション中の MCP サーバーが動作しているか確認する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> 「Go」で1件だけ検索して、結果が返るか確認してください

- **成功** → Phase 4 へ
- **失敗** → ユーザーに以下を案内し、**このスキルの実行を停止する**:
  - `task install` でバイナリは更新されたが、Claude Code セッション中の MCP サーバープロセスは古いまま
  - ユーザーが `/mcp` コマンドで MCP サーバーを再起動した後、`/dogfood-verify` を再実行する

### Phase 4: 全 MCP ツールのスモークテスト

各ツールを実際に呼び出し、動作を検証する。

#### 4-1: oreilly_search_content

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> 「Docker」で5件検索してください (BFS モード)

| 成功条件 | 失敗時の対応 |
|---------|------------|
| 結果が返り、認証エラーなし | 環境変数 (OREILLY_USER_ID, OREILLY_PASSWORD) と Cookie 状態を確認案内 |

#### 4-2: oreilly_ask_question

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> 「What is Docker?」と質問してください (最大待機60秒)

| 成功条件 | 失敗時の対応 |
|---------|------------|
| answer フィールドが存在 | タイムアウト → 警告のみ、次へ進む |

#### 4-3: Resources チェーン検証

4-1 の search 結果から `product_id` を1件取得し、リソースアクセスを検証する。

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> 4-1 で取得した書籍の product_id を使って、book-details リソースにアクセスしてください

| 成功条件 | 失敗時の対応 |
|---------|------------|
| 書籍タイトル・著者情報が返る | 認証エラー → Cookie/環境変数確認案内、404 → product_id の形式確認 |

#### 4-4: Prompts 動作確認

`orm-discovery-mcp-go:oreilly-researcher` subagent で以下を実行:

> learn-technology プロンプトで technology="Go" を指定して、学習パスを生成してください

| 成功条件 | 失敗時の対応 |
|---------|------------|
| 学習パスが生成される | プロンプト未登録 → 警告のみ |

#### 4-5: review_pr (存在時のみ)

MCP ツールとして `review_pr` が登録されている場合のみ実行:

```
repo_path: "."
base_branch: "main"
```

| 成功条件 | 失敗時の対応 |
|---------|------------|
| findings が配列 | ツール未登録 → スキップ (⏭️)、その他エラー → 警告のみ |

### Phase 5: 自己 PR レビュー (PR 存在時のみ)

現在のブランチに PR が存在するか確認:

```bash
gh pr view --json number,title,headRefName,baseRefName
```

- **PR あり** → Phase 4-5 の review_pr 結果 (利用可能時) を Severity でグループ化して表示
- **PR なし** → スキップ

**注意**: PR コメントへの反映は行わない (将来の B5-1 scope)。

### Phase 6: 結果報告

以下の構造化レポートを出力する:

```markdown
## Dogfood Verify Result

### CI: ✅/❌
### Installation: ✅/❌
### MCP Tools:
- oreilly_search_content: ✅/❌/⏭️
- oreilly_ask_question: ✅/❌/⏭️
### MCP Resources:
- book-details chain: ✅/❌/⏭️
### MCP Prompts:
- learn-technology: ✅/❌/⏭️
### review_pr: ✅/❌/⏭️

### Self PR Review (PR #XXX): Critical X / Warning Y / Info Z
### Recommendation: finalize-pr 推奨 or 修正必要
```

凡例:
- ✅ = 成功
- ❌ = 失敗
- ⏭️ = スキップ (ツール未登録、PR なし等)

### Phase 7: finalize-pr 推奨 (条件付き)

以下の条件をすべて満たす場合、`/finalize-pr` の実行を推奨する:

1. CI が通っている
2. MCP ツールのスモークテストに Critical な失敗がない
3. Self PR Review で Critical Issue == 0

条件を満たさない場合は、修正すべき項目を明示して停止。

## 注意事項

- CI 失敗時は即停止 (CI 優先ポリシー)
- MCP サーバーの再起動が必要な場合がある (`/mcp` コマンド案内)
- O'Reilly 認証情報 (OREILLY_USER_ID, OREILLY_PASSWORD) が環境変数に設定されていること
- `oreilly_ask_question` のタイムアウトは警告のみ (ネットワーク依存のため)
- review_pr ツールが未登録の場合はスキップ
- Resources/Prompts は代表的な項目のみスモークテスト (全件は dogfood-improve で追跡)
- このスキルの実行中にエラーが発生した場合は、その時点で停止して報告する
