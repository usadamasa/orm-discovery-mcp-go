# MVR (Minimal Viable Refactor) 進捗管理

## 概要

7フェーズの大規模リファクタリングの代わりに、3ステップの MVR で80%の価値を20%の工数で実現する。

## ステップ一覧

### Step 1: RLock バグ修正 (1 PR, ~1h)

- **ステータス**: 完了
- **ファイル**: `research_history.go:121`
- **内容**: `RLock` → `Lock`、`RUnlock` → `Unlock` に変更
- **リスク**: 低 (構造変更なし、import 変更なし)

### Step 2: BrowserClient インターフェース導入 (1 PR, ~4h)

- **ステータス**: 完了
- **ファイル**: `browser/types.go` (Client インターフェース追加), `server.go:34` (型変更), `main.go` (typed nil 対策)
- **内容**: BrowserClient の公開メソッドをインターフェースとして定義し、server.go で具体型→インターフェース型に変更
- **効果**: 全 O'Reilly ハンドラーが mock でテスト可能に
- **注意点**: Go の typed nil 問題を回避するため、main.go で明示的に interface nil を渡すように変更

### Step 3: ReviewPRHandler の分離 (1 PR, ~3h)

- **ステータス**: 完了
- **ファイル**: `internal/review/register.go` (新規), `server.go` (ReviewPR 部分を削除), `.go-arch-lint.yml` (review コンポーネント追加)
- **内容**: ReviewPR 関連のコードを `internal/review/` に移動し、server.go から分離
- **効果**: server.go が純粋に O'Reilly ドメイン専用に。main パッケージから internal/critic, internal/git, internal/reviewer への直接依存が消滅

## 検証結果

- `task ci` 全パス (lint 0 issues, arch-lint OK, test 全パス, build 成功)
- `go vet ./...` エラーなし
- `server.go` の `browserClient` フィールド型: `browser.Client` (インターフェース)
- `go list -f '{{.Imports}}' .` に `internal/critic`, `internal/git`, `internal/reviewer` 含まれないことを確認済み

## 変更ファイルサマリー

| ファイル | 変更内容 |
|---------|---------|
| `research_history.go` | Step 1: RLock → Lock, RUnlock → Unlock |
| `browser/types.go` | Step 2: Client インターフェース追加 + コンパイルチェック |
| `server.go` | Step 2: 型変更 (具体型→インターフェース), Step 3: ReviewPR 削除 + review.RegisterTools 呼び出し |
| `main.go` | Step 2: typed nil 対策 |
| `tools_args.go` | Step 3: ReviewPR 関連型を削除 |
| `internal/review/register.go` | Step 3: 新規作成 (ReviewPR ツール登録とハンドラー) |
| `.go-arch-lint.yml` | Step 3: review コンポーネント追加、main の依存更新 |

## MVR 後の判断ポイント

MVR 完了後、以下の状況に応じて追加リファクタリングを判断:

- B5-1 (#102) 着手時 → EventStore 設計を検討
- Track A (#87-92) 着手時 → `internal/store/`, `internal/stats/` を新規作成
- server.go の O'Reilly ハンドラーが増えて管理困難になった時 → `internal/oreilly/` 抽出を検討
