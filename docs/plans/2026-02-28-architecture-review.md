# orm-discovery-mcp-go アーキテクチャ再検討プラン

## Context

orm-discovery-mcp-go は「O'Reilly Learning Platform の MCP サーバー」として始まったが、Track A (AI学習ループ) と Track B (コードレビューエージェント) の追加により「マルチドメイン MCP プラットフォーム」に進化しつつある。現在のフラットなアーキテクチャはこの進化に対応できておらず、構造的負債が蓄積している。

本プランでは **2つのアーキテクチャパターン** (モノリポ再構成 vs リポジトリ分割) を比較し、推奨アプローチとマイグレーション戦略を提示する。

---

## 1. 現状アーキテクチャの批判的分析

### 1.1 構造的問題

| 問題 | 深刻度 | 詳細 |
|------|--------|------|
| **server.go がゴッドオブジェクト** | Critical | 1153行/28関数。HTTP起動、全ツールハンドラー(4個)、全リソースハンドラー(4+個)、URI解析、履歴記録が同居 |
| **BrowserClient にインターフェースがない** | High | `*browser.BrowserClient` への具体型依存。サーバー層ハンドラーのユニットテスト不可 |
| **無関係な2ドメインが1サーバーに同居** | High | O'Reillyコンテンツ発見とコードレビューが何も共有しないのに server.go で結合 |
| **フラットなルートパッケージ** | Medium | 10+ファイルが root に混在。Track A で SQLite/stats 追加時にさらに悪化 |
| **research_history が O'Reilly 専用** | Medium | Track B の code_review_event (#102) を収容する汎用性がない |
| **RLock バグ** | Medium | `research_history.go:Save()` で RLock 中に LastUpdated 書き込み |
| **FullResponse 無制限ストレージ** | Low | 検索結果の完全レスポンスが無制限にディスク保存される |

### 1.2 複雑度問題 (tech-debt-tracker M1-M9)

| 関数 | ファイル | 認知的複雑度 | 問題の性質 |
|------|---------|-------------|-----------|
| `normalizeSearchResult` | `browser/search.go` | 80 | API レスポンスの多形性処理 → 分解可能 |
| `runVisibleLogin` | `browser/login.go` | 64 | プロセス管理の本質的複雑性 → 現状維持 |
| `convertAPIBookDetailToLocal` | `browser/book.go` | 48 | フィールド変換 → 分解可能 |
| `LoadConfig` | `config.go` | 44 | 環境変数読み込み → 分解可能 |
| `parseHTMLNode` | `browser/book.go` | 26 | HTML パース → 本質的複雑性 |
| `convertAPIFlatTOCToLocal` | `browser/book.go` | 25 | フォーマット変換 → 分解可能 |
| `SearchContent` | `browser/search.go` | 22 | 検索オプション構築 → 軽度 |
| `GetDiff` | `internal/git/diff.go` | 22 | diff パース → 軽度 |
| `registerPrompts` | `prompts.go` | funlen | 登録ロジック → 分割で解消 |

### 1.3 2ドメイン間の結合度分析

**O'Reilly ドメイン** が使うパッケージ:
- `browser/` (auth, search, book, answers, cookie)
- `browser/generated/api/`
- chromedp (login.go のみ)
- golang.org/x/net/html

**Review ドメイン** が使うパッケージ:
- `internal/critic/`
- `internal/git/`
- `internal/model/`
- `internal/reviewer/`

**共有しているもの**: `server.go` (MCP登録)、`config.go`、`mcp.Server` インスタンスのみ。**ビジネスロジックの共有はゼロ。**

ただし Track B の計画では:
- B5-1 (#102): `code_review_event` を `research_history` に記録 → 履歴システムを共有
- B5-2 (#103): `ReviewRecord` を SQLite に保存 → Track A の `internal/store/` を使用
- Track A の学習ループ全体が Review ドメインの出力 (Finding) を入力とする

→ **Track A/B の統合により、将来的には2ドメインが「データフロー」で接続される設計になっている。**

---

## 2. アーキテクチャパターンの比較

### パターン A: モノリポ再構成 (internal/ へのドメイン分離)

```
orm-discovery-mcp-go/
├── main.go, config.go, xdg.go, login_cmd.go  (そのまま)
├── internal/
│   ├── server/           # NEW: 薄いMCPサーバー合成ルート (~150行)
│   │   ├── server.go     # Server struct + Start + Close
│   │   └── middleware.go  # ログミドルウェア
│   ├── oreilly/          # NEW: O'Reillyドメインハンドラー
│   │   ├── client.go     # Client インターフェース定義
│   │   ├── handlers.go   # SearchContent, AskQuestion, Reauth
│   │   ├── resources.go  # book-details, book-toc, book-chapter, answer
│   │   ├── prompts.go    # MCPプロンプト
│   │   └── args.go       # ツール引数型
│   ├── review/           # NEW: レビュードメインハンドラー
│   │   ├── handlers.go   # ReviewPR ハンドラー
│   │   └── args.go       # ReviewPR引数型
│   ├── history/          # NEW: 汎用イベントストア
│   │   ├── event.go      # Event モデル (oreilly.search, code_review 両対応)
│   │   ├── store.go      # EventStore インターフェース + JSON実装
│   │   └── resources.go  # orm-mcp://history/* リソース
│   ├── store/            # Track A: SQLite 永続化層
│   ├── stats/            # Track A: 精度計算・信頼度補正
│   ├── critic/           # そのまま
│   ├── git/              # そのまま
│   ├── model/            # そのまま (+ ReviewRecord, Outcome 追加)
│   ├── reviewer/         # そのまま
│   └── version/          # そのまま
├── browser/              # そのまま (複雑度削減のみ)
```

### パターン B: リポジトリ分割

```
[Repo 1] orm-discovery-mcp-go (O'Reilly MCP サーバー)
├── main.go, config.go, xdg.go, login_cmd.go
├── server.go           # O'Reilly ツール/リソースのみ
├── research_history.go  # O'Reilly 調査履歴のみ
├── browser/            # そのまま
└── prompts.go          # O'Reilly プロンプト

[Repo 2] review-agent-mcp-go (コードレビュー MCP サーバー)
├── main.go             # 新規: MCP サーバー起動
├── server.go           # review_pr ツール + 将来のツール
├── config.go           # レビュー用設定
├── internal/
│   ├── critic/         # 移動
│   ├── git/            # 移動
│   ├── model/          # 移動
│   ├── reviewer/       # 移動
│   ├── store/          # Track A: SQLite
│   └── stats/          # Track A: 精度/信頼度
└── go.mod              # chromedp 依存なし!

[Option: Repo 3] review-shared-go (共有ライブラリ)
├── model/finding.go    # Finding 型 (両方が使う場合)
└── go.mod
```

---

## 3. Pros & Cons 比較

### パターン A: モノリポ再構成

| Pros | Cons |
|------|------|
| デプロイが1バイナリで単純 | ドメイン境界が `internal/` の規約に依存 (強制力なし) |
| ユーザーの MCP 設定が1サーバーのみ | chromedp 依存がレビューツールのみ使うユーザーにも必要 |
| Track A/B の統合 (B5-1, B5-2) が関数呼び出しで完結 | O'Reilly 認証失敗がレビューツールの可用性に影響 (現状の degraded mode) |
| 共有コード (`model.Finding`) の重複なし | バイナリサイズが大きい (chromedp + 全依存) |
| CI/CD が1パイプライン | 一方のドメインの変更が他方のテストも走らせる |
| リファクタリングがアトミックに完結 | 将来3つ目のドメイン追加時に再び肥大化リスク |
| Track A の学習ループ (Finding → Store → Stats → Confidence) が自然に構成できる | |

### パターン B: リポジトリ分割

| Pros | Cons |
|------|------|
| ドメイン境界がリポジトリ境界で厳格に強制 | 2リポジトリのメンテナンス (CI/CD×2、リリース×2) |
| review-agent は chromedp 不要、軽量バイナリ | `model.Finding` の共有に共有ライブラリ or 重複が必要 |
| O'Reilly 認証なしでレビューツール使用可能 | Track A/B 統合 (B5-1: code_review_event, B5-2: ReviewRecord) の設計が大幅に複雑化 |
| 各リポジトリが小さく理解しやすい | ユーザーが2つの MCP サーバーを設定する必要 |
| 独立バージョニング・独立リリースサイクル | Claude Code プラグインを2つ登録する必要 |
| Unix 哲学: 1ツール = 1ジョブ | Track A の学習ループが repo 横断になる (Finding は review-agent で生成、Store/Stats も review-agent、しかし O'Reilly 検索結果の学習は orm-discovery 側) |
| 新チームメンバーのオンボーディングが容易 | 共有ライブラリの変更が2リポジトリの互換性に影響 |

### 判断の鍵: Track A/B 統合の影響

Track A の計画を改めて見ると:

```
[Review ドメイン]        [学習ループ]         [O'Reilly ドメイン]
review_pr 実行
  → Finding 生成  →  ReviewRecord 保存
                      → Outcome 紐付け
                      → PredictionAccuracy
                      → Confidence 補正   →  将来: 検索品質の学習?
```

**リポジトリ分割した場合、Track A の学習ループは review-agent-mcp-go に完結する。** O'Reilly ドメインへの Confidence フィードバックは当面計画されていない (#92 の scope は review_pr の Finding のみ)。

しかし、**B5-1 (#102) の code_review_event は research_history への統合を明示的に要求している**。分割すると、この要件は:
- a) レビューイベントは review-agent 側の独自履歴に記録 (research_history との統合を諦める)
- b) 共有イベントストアを共有ライブラリとして切り出す

のどちらかになり、設計が複雑化する。

---

## 4. 推奨アプローチ

### 推奨: パターン A (モノリポ再構成)、将来の分割に備えた設計

**理由:**

1. **Track A/B の統合要件が分割を困難にする**: B5-1 の code_review_event の research_history 統合、B5-2 の ReviewRecord 保存は、両ドメインがプロセス内で連携する前提で設計されている。分割するとこれらの Issue の再設計が必要。

2. **分割のメリットの大部分はモノリポ再構成でも得られる**: `internal/oreilly/` と `internal/review/` のパッケージ分離により、ドメイン境界は Go のインポートグラフで強制される (circular import はコンパイルエラー)。

3. **分割は後からでもできる**: `internal/oreilly/` と `internal/review/` に正しく分離されていれば、将来の分割は「パッケージを別リポジトリに移動 + main.go を書く」だけで済む。逆に、中途半端な分割を先にやると統合が困難。

4. **プラグインマーケットプレイスの観点**: 1つの MCP サーバー = 1つの設定で両方のツールが使えるのはユーザー体験として優れている。

### ただし、以下の設計原則を厳守する:

- `internal/oreilly/` は `internal/review/` をインポートしない (逆も同様)
- `internal/review/` は `browser/` をインポートしない
- 共有データは `internal/model/` と `internal/history/` のみ
- 各ドメインハンドラーはインターフェース経由で依存注入

→ **この原則を守れば、いつでもリポジトリ分割可能な状態が保たれる。**

---

## 5. マイグレーションロードマップ

### Phase 0: バグ修正 + 複雑度削減 (前提条件、1-2 PR)

構造変更なし。安全なリファクタリングのみ。

- [ ] `research_history.go:Save()` の RLock バグ修正 (RLock → Lock)
- [ ] `browser/search.go:normalizeSearchResult` を 8 個の extract 関数に分解 (認知的複雑度 80→~8)
- [ ] `browser/book.go:convertAPIBookDetailToLocal` を 4 個の convert 関数に分解 (認知的複雑度 48→~10)

**対象ファイル**: `research_history.go`, `browser/search.go`, `browser/book.go`

### Phase 1: OreillyCLient インターフェース導入 (1 PR)

- [ ] `internal/oreilly/client.go` に `Client` インターフェース定義
- [ ] `browser/types.go` に `var _ oreilly.Client = (*BrowserClient)(nil)` コンパイルチェック
- [ ] (行動変更なし、テスタビリティの基盤)

**対象ファイル**: `internal/oreilly/client.go` (新規), `browser/types.go`

### Phase 2: ドメインパッケージ抽出 (2-3 PR)

- [ ] **2a**: 型定義の移動 (`tools_args.go` → `internal/oreilly/args.go` + `internal/review/args.go`)
- [ ] **2b**: O'Reilly ハンドラー移動 (SearchContent, AskQuestion, Reauth → `internal/oreilly/handlers.go`)
- [ ] **2c**: O'Reilly リソース移動 (book-details等 → `internal/oreilly/resources.go`)

**対象ファイル**: `server.go` (分解元), `internal/oreilly/*.go` (新規), `internal/review/handlers.go` (新規)

### Phase 3: 汎用イベントストア (1-2 PR)

- [ ] **3a**: `internal/history/event.go` に汎用 Event モデル (`oreilly.search`, `oreilly.question`, `code_review` の EventType)
- [ ] **3b**: `internal/history/store.go` に EventStore インターフェース + JSON 実装 (FullResponse 1MB 上限)
- [ ] **3c**: `internal/history/resources.go` に `orm-mcp://history/*` リソース移動
- [ ] **3d**: 旧ファイル (`research_history.go`, `history_resources.go`) 削除

→ **Track B #102 (code_review_event) はこの Phase 後に着手可能**

**対象ファイル**: `internal/history/*.go` (新規), `research_history.go` (削除), `history_resources.go` (削除)

### Phase 4: Review ドメイン独立 (1 PR)

- [ ] `ReviewPRHandler` を `internal/review/handlers.go` に移動
- [ ] Review ハンドラーの MCP 登録を Registrar パターンで実装

→ **Track B #103 (ReviewRecord save) はこの Phase 後に着手可能**

**対象ファイル**: `server.go` (ReviewPR 部分削除), `internal/review/handlers.go`

### Phase 5: サーバー合成ルート (1 PR)

- [ ] `internal/server/server.go` (~150行) に Server struct + Start + Close
- [ ] `internal/server/middleware.go` にミドルウェア移動
- [ ] `prompts.go` → `internal/oreilly/prompts.go`
- [ ] `sampling_manager.go` → `internal/oreilly/sampling.go`
- [ ] ルートの `server.go` 削除、`main.go` が `server.NewServer()` を呼ぶ

**対象ファイル**: `internal/server/*.go` (新規), `server.go` (削除), `main.go` (更新)

### Phase 6: Track A パッケージ (Track A Issue に合わせて)

- [ ] `internal/store/` - SQLite ReviewRecord 永続化 (#87)
- [ ] `internal/stats/` - PredictionAccuracy (#90), Confidence EMA (#92)

**これらは新規パッケージであり、マイグレーションコストなし。**

### Phase 間の依存関係

```
Phase 0 (バグ修正・複雑度)
  ↓
Phase 1 (Client インターフェース)
  ↓
Phase 2a (型移動) → Phase 2b (ハンドラー移動) → Phase 2c (リソース移動)
  ↓                     ↓
Phase 3 (EventStore)    Track B #99, #100 (新 Critic) ← ここで並行可能
  ↓
Phase 4 (Review 独立)   Track B #102 (code_review_event) ← Phase 3 後
  ↓
Phase 5 (Server 合成)   Track B #103 (ReviewRecord save) ← Phase 4 後
  ↓
Phase 6 (Track A)       Track A #87-#92 ← Phase 5 後
```

---

## 6. 検証方法

各 Phase の完了時:

1. **`task ci` が全パス** (lint + test + build)
2. **MCP ツール動作確認** (Claude Code から `oreilly_search_content`, `review_pr` 実行)
3. **既存テストが全パス** (移動したテストも含む)
4. **インポートグラフの検証**: `internal/oreilly/` と `internal/review/` 間に直接依存がないこと

```bash
# インポートグラフ確認
go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/oreilly/...
go list -f '{{.ImportPath}}: {{.Imports}}' ./internal/review/...
# → 相互参照がないことを確認
```

5. **最終検証**: `/dogfood-verify` スキルでフルフィードバックループ実行

---

## 7. リスクと対策

| リスク | 対策 |
|--------|------|
| Phase 2-3 でインポート循環 | `internal/oreilly/` → `browser/` は OK、逆方向禁止。Client インターフェースは消費者側 (oreilly) で定義 |
| 型移動時のコンパイルエラー連鎖 | Phase 2a で型のみ先行移動、ハンドラーは後続 PR で移動 |
| 既存履歴ファイルのスキーマ互換性 | `JSONEventStore.Load()` で旧スキーマ自動検出・アップグレード |
| テストカバレッジの一時的低下 | ハンドラー移動と同時に mock Client を使った新テスト追加 |
| Track B 並行作業との衝突 | Phase 2b 完了前は Track B の新 Critic (#99, #100) は `internal/critic/` のみの変更で競合なし |
