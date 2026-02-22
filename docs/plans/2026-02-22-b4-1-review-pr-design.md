# B4-1: review_pr MCP Tool Design

Issue: #101

## Overview

PR レビューを実行する MCP ツール `review_pr` を実装する。既存の 3 Critic (MissingTestCritic, InfraChangeCritic, LargeDiffCritic) を統合し、PR レビューの一連のフローを 1 つの MCP ツールとして提供する。

## Design Decisions

| 項目 | 決定 | 理由 |
|------|------|------|
| 入力パラメータ | repo_path + base_branch (B4-1 scope) | ローカル git diff のみ。GitHub API は将来対応 |
| Critic 実行方式 | 逐次実行 + 並行化可能インターフェース | 現在の 3 Critic は軽量。B3 追加時に並行化検討 |
| パッケージ配置 | `internal/reviewer/` | Critic (個別ロジック) と Reviewer (統合) の関心分離 |
| GitHub API 対応 | インターフェースのみ設計、実装は後回し | YAGNI。DiffProvider インターフェースで将来対応可能 |

## Architecture

```
MCP Client (Claude Code etc.)
    │
    ▼
server.go: review_pr tool
    │  ReviewPRArgs { repo_path, base_branch }
    ▼
internal/reviewer/orchestrator.go
    │
    ├── git.DiffProvider.GetDiff()
    │       → git.DiffResult
    │
    ├── git.ExtractChangedFiles()
    │       → []git.ChangedFile
    │
    ├── git.ClassifyChangedFiles()
    │       → []git.ClassifiedFile
    │
    ├── critic.ReviewInput を組み立て
    │
    ├── Critic[0].Review(ctx, input)  → []Finding
    ├── Critic[1].Review(ctx, input)  → []Finding
    ├── Critic[2].Review(ctx, input)  → []Finding
    │
    └── Finding を集約・ソート
            → ReviewResult
```

## Types

### Orchestrator (`internal/reviewer/orchestrator.go`)

```go
type Orchestrator struct {
    diffProvider git.DiffProvider
    critics      []critic.Critic
}

type ReviewResult struct {
    Findings   []model.Finding
    Summary    ReviewSummary
    BaseBranch string
    TotalFiles int
    Errors     []CriticError
}

type ReviewSummary struct {
    CriticalCount int
    WarningCount  int
    InfoCount     int
}

type CriticError struct {
    CriticName string
    Err        error
}

func NewOrchestrator(dp git.DiffProvider, critics ...critic.Critic) *Orchestrator
func (o *Orchestrator) Run(ctx context.Context, repoPath, baseBranch string) (*ReviewResult, error)
```

### MCP Tool (`server.go`)

```go
// tools_args.go
type ReviewPRArgs struct {
    RepoPath   string `json:"repo_path"`
    BaseBranch string `json:"base_branch"`
}

type ReviewPRResult struct {
    Summary  ReviewSummaryJSON `json:"summary"`
    Findings []FindingJSON     `json:"findings"`
    Errors   []string          `json:"errors,omitempty"`
}
```

Tool Annotations:
- `ReadOnlyHint: true`
- `DestructiveHint: false`
- `IdempotentHint: true`

## Error Handling

| エラー種別 | 挙動 |
|-----------|------|
| DiffProvider.GetDiff() 失敗 | 即座にエラー返却 (Critic 実行なし) |
| 個別 Critic エラー | CriticError に記録、他の Critic は続行 |
| context timeout | ctx.Done() チェック、残り Critic スキップ |

## Finding Sort Order

1. Severity (Critical > Warning > Info)
2. 同一 Severity 内は Category でグループ化

## Test Strategy

- `internal/reviewer/orchestrator_test.go`: モック DiffProvider + 実 Critic
- テーブル駆動テスト: 空 diff, Critic エラー, Finding 集約, ソート順
- コンパイル時チェック不要 (Orchestrator は interface を実装しない)
- server.go ハンドラテストは既存パターンに準拠

## Scope (B4-1)

### In Scope
- [x] Orchestrator 型定義と Run() メソッド
- [x] review_pr MCP ツール定義 (server.go)
- [x] 3 Critic (MissingTest, InfraChange, LargeDiff) の統合
- [x] Finding の集約とソート
- [x] テスト

### Out of Scope (将来タスク)
- GitHub API での PR diff 取得
- Critic の有効/無効設定
- B3-1/B3-2 (go test/build lint Critic)
- B5-1 (code_review_event) との連携
- Research History への保存

## Future Considerations (from MEMORY.md)

B4-1 実装時に以下を検討:
- `newCriticFinding()` ヘルパー (CriticName 設定忘れ防止)
- `classifyInfraType` と `isInfraFile` の統合 (ロジック重複排除)
- `NewReviewInput()` コンストラクタ (nil チェック付き)

ただしこれらは B4-1 の主目的ではないため、実装が自然な場合のみ対応する。
