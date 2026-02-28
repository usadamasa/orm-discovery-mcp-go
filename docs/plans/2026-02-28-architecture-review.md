# orm-discovery-mcp-go アーキテクチャ再検討プラン

## Context

orm-discovery-mcp-go は「O'Reilly Learning Platform の MCP サーバー」として始まったが、Track A (AI学習ループ) と Track B (コードレビューエージェント) の追加により「マルチドメイン MCP プラットフォーム」に進化しつつある。

本プランでは現状の構造的問題を分析し、段階的な改善戦略を提示する。

### MVR (Minimal Viable Refactor) 実施済み

2026-02-28 に以下の3ステップを実施し、核心的な構造問題を解決した:

| Step | 内容 | 変更ファイル |
|------|------|-------------|
| 1 | RLock バグ修正 | `research_history.go` |
| 2 | `browser.Client` インターフェース導入 | `browser/types.go`, `server.go`, `main.go` |
| 3 | ReviewPRHandler の `internal/review/` 分離 | `internal/review/register.go` (新規), `server.go`, `tools_args.go`, `.go-arch-lint.yml` |

---

## 1. 現状アーキテクチャ (MVR 実施後)

### 1.1 解決済みの問題

- **RLock バグ**: `research_history.go:Save()` の `RLock` → `Lock` に修正。データ競合を解消
- **BrowserClient の具体型依存**: `browser.Client` インターフェース (9メソッド) を導入。`server.go` はインターフェース経由で依存。全 O'Reilly ハンドラーが mock でテスト可能
- **2ドメインの同居**: ReviewPRHandler を `internal/review/register.go` に分離。`server.go` は純粋に O'Reilly ドメイン専用。main パッケージから `internal/critic`, `internal/git`, `internal/reviewer` への直接依存が消滅
- **アーキテクチャルールの強制**: `.go-arch-lint.yml` で `review` コンポーネントを定義。`internal/review/` → `browser/` の依存を禁止

### 1.2 残存する構造的問題

| 問題 | 深刻度 | 詳細 |
|------|--------|------|
| **research_history が O'Reilly 専用** | Medium | Track B の code_review_event (#102) を収容する汎用性がない |
| **FullResponse 無制限ストレージ** | Low | 検索結果の完全レスポンスが無制限にディスク保存される |

### 1.3 残存する複雑度問題 (tech-debt-tracker)

| 関数 | ファイル | 認知的複雑度 | 対応方針 |
|------|---------|-------------|----------|
| `normalizeSearchResult` | `browser/search.go` | 80 | 分解可能: 8個の extract 関数に分割 |
| `runVisibleLogin` | `browser/login.go` | 64 | 現状維持: プロセス管理の本質的複雑性 |
| `convertAPIBookDetailToLocal` | `browser/book.go` | 48 | 分解可能: 4個の convert 関数に分割 |
| `LoadConfig` | `config.go` | 44 | 現状維持: 12個の env 変数を順次読むフラット構造。分解しても認知的負荷は変わらない |
| `convertFlatTOCArrayToLocal` | `browser/book.go` | - | 分解可能: 元プランで見落とされていた3つ目のコンバーター |

### 1.4 現在のドメイン分離状態

```
main パッケージ (server.go = O'Reilly ファサード)
  ├── browser.Client (インターフェース)
  ├── internal/review/ (ReviewPR ツール登録)
  └── .go-arch-lint.yml で依存ルールを強制
```

**O'Reilly ドメイン** (main + browser/):
- `browser/` (auth, search, book, answers, cookie)
- `browser/generated/api/`
- chromedp (login.go のみ)

**Review ドメイン** (internal/review/ + internal/critic,git,reviewer/):
- `internal/review/` → `internal/critic/`, `internal/git/`, `internal/reviewer/`
- browser/ への依存なし (arch-lint で強制)

**ビジネスロジックの共有はゼロ。分割可能な状態を維持。**

---

## 2. Pattern A vs B: 判断を保留

### 元プランの推奨

元プランは Pattern A (モノリポ再構成) を推奨したが、根拠は B5-1 (#102) と B5-2 (#103) の統合要件。

### 反論: 根拠が脆弱

- B5-1/B5-2 は**未実装**。`internal/store/` と `internal/stats/` は存在しない
- code_review_event を O'Reilly の research_history に混ぜる必要があるのか自体が疑問
- Pattern B でも共有 SQLite (WAL モード) や別々の履歴ストアで統合は可能

### 現時点の判断

| 条件 | 推奨 |
|------|------|
| B5-1/B5-2 が3ヶ月以内に着手 | Pattern A が合理的 |
| B5-1/B5-2 が6ヶ月以上先 | Pattern B がより自然 |
| 現時点 | **判断を保留**。MVR で分割可能な状態は達成済み |

---

## 3. 追加リファクタリングのトリガー条件

MVR で核心的問題は解決済み。以下の条件を満たしたときに追加リファクタリングを実施する。

| トリガー | 実施内容 | 対応する元 Phase |
|---------|---------|-----------------|
| B5-1 (#102) 着手時 | EventStore 設計: `internal/history/` に汎用イベントモデルを導入 | Phase 3 |
| Track A (#87-92) 着手時 | `internal/store/` (SQLite), `internal/stats/` (精度計算) を新規作成 | Phase 6 |
| server.go が 1500行を超えた時 | O'Reilly ハンドラーの `internal/oreilly/` 抽出を検討 | Phase 2 |
| 複雑度削減の必要が出た時 | normalizeSearchResult (CC=80), convertAPIBookDetailToLocal (CC=48) の分解 | Phase 0 残り |

---

## 4. 設計原則 (維持すべきルール)

以下の原則は MVR で確立され、`.go-arch-lint.yml` で強制されている:

1. `internal/review/` は `browser/` をインポートしない
2. `main` パッケージは `internal/critic/`, `internal/git/`, `internal/reviewer/` を直接インポートしない (`internal/review/` 経由)
3. 将来 `internal/oreilly/` を作る場合、`internal/review/` との相互インポートを禁止
4. ドメインハンドラーはインターフェース経由で依存注入 (`browser.Client`)

---

## 5. 元プランからの修正点

| 元プランの主張 | 修正 |
|--------------|------|
| server.go は God Object (1153行/28関数) | 実際は26関数、半数は20行以下。問題は「God Object」ではなく「依存注入の欠如 + ドメイン混在」。MVR で解決済み |
| Phase 1 で `internal/oreilly/client.go` にインターフェース定義 | 実際は `browser/types.go` に配置。Go の慣例に従い実装側に定義 |
| LoadConfig (CC=44) は分解可能 | 順次構造のため分解の効果は限定的。現状維持が妥当 |
| 7フェーズ/35時間のロードマップ | MVR 3ステップ/8時間で80%の価値を達成。残りは具体的な機能要件の着手時に実施 |
| `.go-arch-lint.yml` への影響 | 元プランで未言及。新パッケージ追加時は必ず更新が必要 |
| Go typed nil 問題 | 元プランで未言及。`*BrowserClient` → `browser.Client` 変更時に main.go の修正が必要だった |

---

## 6. 検証方法

追加リファクタリング実施時:

1. `task ci` 全パス (lint + arch-lint + test + build)
2. `go vet ./...` エラーなし
3. インポートグラフの検証:

```bash
# main パッケージの依存確認
go list -f '{{.Imports}}' . | tr ' ' '\n' | grep -E 'internal/(critic|git|reviewer)$'
# → 出力なしであること

# review パッケージの依存確認
go list -f '{{.Imports}}' ./internal/review/ | tr ' ' '\n' | grep browser
# → 出力なしであること
```

4. `/dogfood-verify` スキルでフルフィードバックループ実行
