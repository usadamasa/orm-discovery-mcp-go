# MCP Best Practices 監査レポート

> **監査日**: 2026-02-14
> **対象**: orm-discovery-mcp-go (O'Reilly Learning Platform MCP Server)
> **監査基準**: [MCP Server Best Practices](https://modelcontextprotocol.io/specification/2025-03-26)

## サマリー

| 優先度 | 件数 | 内容 |
|--------|------|------|
| P1 (High) | 2 | URIパース、レスポンス形式 |
| P2 (Medium) | 4 | エラーメッセージ、入力バリデーション、エラー判定、timeout |
| P3 (Low) | 9 | ドキュメント、命名、i18n、magic string、SDK新機能活用 |
| Testing | 3 | cookie、middleware、testify移行 |
| **合計** | **18** | |

---

## P1: High

### P1-001: URI パースが文字列分割ベース

**カテゴリ**: URI Handling / Robustness
**対象ファイル**: `server.go:719-744`, `history_resources.go:312-362`
**ベストプラクティス参照**: [RFC 3986 - URI Generic Syntax](https://datatracker.ietf.org/doc/html/rfc3986)

**現状**:

```go
// server.go:721
func extractProductIDFromURI(uri string) string {
    parts := strings.Split(uri, "/")
    if len(parts) >= 3 {
        return parts[len(parts)-1]
    }
    return ""
}
```

- `strings.Split()` による分割で URL エンコードされた文字 (`%2F` 等) を処理できない
- `history_resources.go` では `url.ParseQuery` を部分的に使用 (line 336) しているが、パスの解析は `strings.Split` のまま (line 321, 353)
- 一貫性がない

**推奨対応**:

```go
func extractProductIDFromURI(uri string) string {
    u, err := url.Parse(uri)
    if err != nil {
        return ""
    }
    parts := strings.Split(u.Path, "/")
    // ...
}
```

**影響**: 特殊文字を含む product_id や chapter_name が正しく解析されない。

---

### P1-002: レスポンス形式オプションなし

**カテゴリ**: Response Format
**対象ファイル**: `server.go:129-152`, `tools_args.go:17-37`
**ベストプラクティス参照**: MCP Best Practices - Response Format Options

**現状**:

- ツールレスポンスは JSON のみ
- Markdown 形式のサポートなし
- クライアントが希望する形式を指定できない

**推奨対応**:

1. `SearchContentArgs` と `AskQuestionArgs` に `format string` フィールドを追加 (`"json"` | `"markdown"`)
2. Markdown 形式の場合、人間が読みやすいフォーマットでレスポンスを返す
3. デフォルトは `"json"` を維持

**影響**: LLM が JSON を再整形する必要があり、トークンを無駄に消費する場面がある。

**SDK 実装手段 (v1.3.1)**:

SDK v1.3.1 では、新たに `OutputSchema`（および関連機能）が追加されている。一方、`StructuredContent` 自体は SDK v1.2.0 以降で既にサポートされており、本リポジトリでも structured response（例: `server.go`）として利用済みである。ツール定義に `OutputSchema` を設定することで、クライアントが期待するレスポンス構造を宣言的に指定でき、`StructuredContent` でそのスキーマに準拠した機械可読な構造化レスポンスを返せる。これにより、LLM 側での JSON／Markdown への再整形処理を減らし、型付き出力の保証とトークン消費削減が期待できる。なお、`OutputSchema`／`StructuredContent` は Markdown 形式サポートそのものの代替ではなく、人間向けの Markdown 出力オプションと併用し得る補完的な仕組みとして位置付けるとよい。

---

## P2: Medium

### P2-001: エラーメッセージが内部詳細を露出

**カテゴリ**: Error Handling / Security
**対象ファイル**: `server.go` (複数箇所)
**ベストプラクティス参照**: MCP Best Practices - Error Handling

**現状**:

```go
// server.go:303
return newToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil, nil

// server.go:428
return newToolResultError(fmt.Sprintf("failed to ask question: %v", err)), nil, nil

// server.go:484
Text: fmt.Sprintf(`{"error": "failed to get book details: %v"}`, err),
```

- 内部エラー詳細がそのままクライアントに返される
- アクション可能なガイダンスがない (例: 「認証情報を確認してください」)

**推奨対応**:

1. ユーザー向けエラーメッセージと内部ログを分離
2. エラーカテゴリ (認証、ネットワーク、バリデーション等) に応じた汎用メッセージ
3. 内部詳細は `slog.Error` にのみ出力

**影響**: 内部実装の詳細がクライアントに漏洩。セキュリティリスクは低いが、ベストプラクティスに反する。

---

### P2-003: 入力スキーマバリデーション不足

**カテゴリ**: Input Validation
**対象ファイル**: `server.go:252-260`, `tools_args.go:17-37`
**ベストプラクティス参照**: MCP Best Practices - Input Validation

**現状**:

- `query` の長さ制限なし (空チェックのみ: line 257-259)
- `rows` の上限制限なし (負値チェックのみ: line 262)
- `question` の長さ制限なし (空チェックのみ: line 404)
- JSON Schema の `maxLength`, `minimum`, `maximum` 等の制約アノテーションなし

**推奨対応**:

1. `query`: maxLength=500, minLength=1
2. `rows`: minimum=1, maximum=100
3. `question`: maxLength=500, minLength=1
4. JSON Schema の `jsonschema` タグに制約を追加

**影響**: 予期しない入力による API 呼び出しの失敗やリソース浪費。

**SDK 実装手段 (v1.1.0)**:

SDK v1.1.0 で追加された `server.SchemaCache` を活用することで、JSON Schema のバリデーションを高速化できる。スキーマのコンパイル結果をキャッシュし、リクエストごとのバリデーションオーバーヘッドを削減する。

---

### P2-004: `os.IsNotExist` が非推奨パターン

**カテゴリ**: Error Handling / Correctness
**対象ファイル**: `research_history.go:92`
**ベストプラクティス参照**: Go 1.13+ errors パッケージ

**現状**:

```go
if os.IsNotExist(err) {
```

- `os.IsNotExist()` はラップされたエラーを検出できない
- Go 1.13 以降は `errors.Is(err, os.ErrNotExist)` が推奨

**推奨対応**:

```go
if errors.Is(err, os.ErrNotExist) {
```

**影響**: 現時点では `os.ReadFile` が直接返すため実害なし。ただしリファクタリングでエラーラップが導入された場合にバグになる。

---

### P2-005: Hardcoded timeout magic numbers

**カテゴリ**: Code Quality / Readability
**対象ファイル**: `server.go:133-135,143`
**ベストプラクティス参照**: Go Code Review Comments - Magic Numbers

**現状**:

```go
ReadTimeout:  30 * time.Second,
WriteTimeout: 60 * time.Second,
IdleTimeout:  120 * time.Second,
// ...
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
```

- timeout 値が意図の説明なくリテラルとして埋め込まれている
- 変更時に複数箇所を確認する必要がある

**推奨対応**:

```go
const (
    httpReadTimeout     = 30 * time.Second
    httpWriteTimeout    = 60 * time.Second
    httpIdleTimeout     = 120 * time.Second
    httpShutdownTimeout = 5 * time.Second
)
```

**影響**: 可読性と保守性の改善。

---

## P3: Low

### P3-001: ツール説明にワーキングサンプルが不足

**カテゴリ**: Documentation / Discoverability
**対象ファイル**: `server.go:131-137`, `server.go:144-150`
**ベストプラクティス参照**: MCP Best Practices - Tool Descriptions

**現状**:

ツール説明に Good/Poor の例が1つずつあるが、ベストプラクティスでは3件以上の具体例が推奨される。

**推奨対応**:

- 異なるユースケースの例を追加 (技術名検索、比較検索、トピック検索等)
- 期待されるレスポンス形式の簡略例を含める

---

### P3-002: レート制限のドキュメントなし

**カテゴリ**: Documentation
**対象ファイル**: ツール説明 (`server.go:131-137`, `server.go:144-150`)
**ベストプラクティス参照**: MCP Best Practices - Rate Limiting Documentation

**現状**:

O'Reilly API のレート制限やリクエスト間隔の推奨値がツール説明に含まれていない。

**推奨対応**:

ツール説明に「Rate limit: ~10 requests/minute recommended」等のガイダンスを追加。

---

### P3-003: パフォーマンス特性のドキュメントなし

**カテゴリ**: Documentation
**対象ファイル**: ツール説明 (`server.go:131-137`, `server.go:144-150`)
**ベストプラクティス参照**: MCP Best Practices - Performance Documentation

**現状**:

- `search_content` の BFS/DFS モードによるレスポンスサイズの違いがツール説明に含まれていない
- `ask_question` のデフォルトタイムアウト (300秒) がツール説明に明記されていない

**推奨対応**:

ツール説明に「BFS: ~2-5KB, DFS: ~50-100KB」「Default timeout: 5 minutes」等を追加。

---

### P3-004: サーバー実装名が慣例に沿っていない

**カテゴリ**: Naming Convention
**対象ファイル**: `server.go:30`
**ベストプラクティス参照**: MCP Best Practices - Server Implementation Name

**現状**:

```go
mcpServer := mcp.NewServer(
    &mcp.Implementation{
        Name:    "Search O'Reilly Learning Platform",
        Version: "1.0.0",
    },
    nil,
)
```

- 実装名がフルフレーズ。慣例では短い識別子 (例: `orm-discovery-mcp-go`)
- バージョンが `1.0.0` ハードコード。GoReleaser のバージョン変数 (`main.go:18`) と連動していない

**推奨対応**:

```go
Name:    "orm-discovery-mcp-go",
Version: version, // GoReleaser から注入される値を使用
```

**追加対応 (SDK v1.1.0)**:

`ServerOptions.Instructions` フィールドを設定し、サーバーの利用ガイダンスをクライアントに提供する。実装名修正と合わせて、`Instructions` にサーバーの概要・使い方・制約事項を記載する。

---

### P3-005: キーワード抽出が英語のみ

**カテゴリ**: i18n / Search Quality
**対象ファイル**: `research_history.go` (SearchByKeyword 実装)
**ベストプラクティス参照**: N/A (品質改善)

**現状**:

- 検索履歴のキーワードマッチが英語ストップワードのみ対応
- 日本語コンテンツが `languages: ["en", "ja"]` でデフォルト検索されるが、日本語ストップワードが未定義

**推奨対応**:

- 日本語ストップワードリストを追加
- Unicode 正規化 (NFKC) の適用を検討

---

### P3-006: Magic string constants in middleware

**カテゴリ**: Code Quality / Readability
**対象ファイル**: `middleware.go:73,86`
**ベストプラクティス参照**: Go Code Review Comments - Constants

**現状**:

```go
if method == "tools/call" {
// ...
if method == "resources/read" {
```

- MCP メソッド名がリテラル文字列として埋め込まれている
- タイポや不整合のリスク

**推奨対応**:

```go
const (
    mcpMethodToolsCall     = "tools/call"
    mcpMethodResourcesRead = "resources/read"
)
```

**影響**: 可読性の改善、タイポ防止。

---

### P3-007: Tool.Title / Prompt.Icons フィールド移行

**カテゴリ**: SDK Migration
**対象ファイル**: `server.go` (ツール定義箇所)
**ベストプラクティス参照**: MCP SDK v1.2.0 - Tool.Title, Prompt.Icons

**現状**:

- ツールのタイトルが `Annotations.Title` に設定されている
- SDK v1.2.0 でトップレベル `Title` フィールドが追加された

**推奨対応**:

- `Annotations.Title` → トップレベル `Title` フィールドへ移行
- Prompt 定義に `Icons` フィールドを設定し、クライアント UI での視認性を向上

---

### P3-008: ResourceLink コンテンツタイプ導入

**カテゴリ**: SDK Feature Adoption
**対象ファイル**: `server.go` (ツールレスポンス生成箇所)
**ベストプラクティス参照**: MCP SDK v1.2.0+ - ResourceLink

**現状**:

- ツール結果はテキストコンテンツのみを返す
- 関連するリソース URI (`oreilly://book-details/{product_id}` 等) への直接リンクがない

**推奨対応**:

- `oreilly_search_content` の結果に `ResourceLink` を埋め込み、`oreilly://book-details/{product_id}` へのリンクを提供
- クライアントが検索結果から直接リソースにアクセスできるようにする
- ツール結果とリソースの連携を強化

---

### P3-009: LoggingHandler による MCP ログ送信

**カテゴリ**: SDK Feature Adoption
**対象ファイル**: `server.go`, `config.go`
**ベストプラクティス参照**: MCP SDK v1.1.0 - NewLoggingHandler

**現状**:

- ログは `slog` 経由でファイル/stderr にのみ出力
- MCP クライアントへのログ配信機能なし

**推奨対応**:

- SDK v1.1.0 の `NewLoggingHandler` を使用して、`slog.Handler` を MCP ログ通知に変換
- クライアントがサーバーのログをリアルタイムで受信可能にする
- 既存の `slog` ハンドラとの `MultiHandler` 構成で併用

---

## Testing Debt

### T1: browser/cookie テストなし

**カテゴリ**: Test Coverage
**対象ファイル**: `browser/cookie/cookie.go` (~150行)
**優先度**: High

**現状**:

- Cookie 管理ロジック (保存、読み込み、有効期限チェック) にユニットテストがない
- セキュリティ関連コード (パーミッション `0600`) の検証がない

**推奨対応**:

- Cookie の保存/読み込みラウンドトリップテスト
- 有効期限切れ Cookie の処理テスト
- ファイルパーミッションの検証テスト

---

### T2: middleware テストなし

**カテゴリ**: Test Coverage
**対象ファイル**: `middleware.go` (95行)
**優先度**: Medium

**現状**:

- ロギングミドルウェアにユニットテストがない
- ログ出力の条件分岐 (LogLevel による分岐) が未検証

**推奨対応**:

- Debug/Info レベルでの出力切り替えテスト
- ツール呼び出し/リソース読み込みの識別テスト

---

### T3: browser テストの testify 移行

**カテゴリ**: Test Consistency
**対象ファイル**: `browser/auth_test.go`, `browser/answers_test.go`
**優先度**: Low

**現状**:

- プロジェクト全体では `testify` を使用しているが、browser パッケージの一部テストは標準の `testing` パッケージのみ
- アサーション形式の不統一

**推奨対応**:

- `assert` / `require` パッケージへの段階的移行
- プロジェクト全体での一貫性確保

---

## 対応不要と判定した項目

以下の項目は分析で検討したが、対応不要と判定した:

| 問題 | 判定理由 |
|------|----------|
| bare error returns (server.go:151, browser/auth.go:290,376) | server.go:151 は `ListenAndServe` のトップレベルエラー。browser/auth.go は `chromedp.ActionFunc` 内でラップ不適切 |
| `_ = options["languages"]` 未使用 (server.go) | コメントで「将来の拡張用」と明示。意図的な保持 |
| server.go が大きい (965行) | 機能追加に支障なし。分割のリスク > メリット |
| browser/book.go, search.go テストなし | E2E でカバー済み。ユニットテスト追加は大工数で別フェーズ |

---

## 準拠している項目 (Good Practices)

以下の項目はベストプラクティスに準拠していることを確認:

| 項目 | 対象箇所 | 説明 |
|------|---------|------|
| stdio ログ出力先 | `config.go:185` | `os.Stderr` に正しく出力。stdio トランスポートの stdout を汚染しない |
| 認証情報管理 | `config.go:64-65` | 環境変数のみ。ハードコードなし |
| XDG Base Directory 準拠 | `config.go:74-82` | `XDG_STATE_HOME`, `XDG_CACHE_HOME`, `XDG_CONFIG_HOME` を使用 |
| Graceful shutdown | `server.go:90-98` | Context キャンセルによる HTTP サーバーの graceful shutdown |
| Cookie パーミッション | `browser/cookie/` | `0600` パーミッションでローカルに保存 |
| スレッドセーフ | `research_history.go` | `sync.RWMutex` による並行アクセス保護 |
| シグナルハンドリング | `main.go:40-46` | `SIGINT`, `SIGTERM` を適切にハンドリング |
| ログローテーション | `config.go:188-196` | Lumberjack によるローテーション対応 |

---

## 改善ロードマップ

### Phase 1: セキュリティ/安定性 (P0) - 完了

- [x] P0-001: ツール名プレフィックス追加 + アノテーション設定
- [x] P0-002: ページネーション実装
- [x] P0-003: HTTP バインドアドレス修正 + Origin 検証

### Phase 2: 堅牢性 (P1)

- [x] P1-001: URI パースを `url.Parse()` ベースに統一
- [ ] P1-002: Markdown レスポンス形式サポート

### Phase 3: 品質改善 (P2)

- [ ] P2-001: エラーメッセージの分離
- [x] P2-002: デフォルト rows 値の変更
- [ ] P2-003: 入力バリデーション強化
- [x] P2-004: `os.IsNotExist` → `errors.Is` 修正
- [x] P2-005: Timeout magic numbers の定数化

### Phase 4: ドキュメント/仕上げ (P3)

- [ ] P3-001: ツール説明の例を拡充
- [ ] P3-002: レート制限ドキュメント追加
- [ ] P3-003: パフォーマンス特性ドキュメント追加
- [ ] P3-004: サーバー実装名の修正 + Instructions 設定
- [ ] P3-005: 日本語ストップワード対応
- [x] P3-006: Middleware の magic string 定数化
- [ ] P3-007: Tool.Title / Prompt.Icons フィールド移行
- [ ] P3-008: ResourceLink コンテンツタイプ導入
- [ ] P3-009: LoggingHandler による MCP ログ送信

### Phase 5: テストカバレッジ改善

- [ ] T1: browser/cookie テスト追加
- [ ] T2: middleware テスト追加
- [ ] T3: browser テストの testify 移行

### Phase 6: アーキテクチャメトリクス改善 (introduce-go-metric で新規追加)

golangci-lint + go-arch-lint 導入時に exclusion 設定で暫定回避した技術的負債。
exclusion を削除してメトリクスを改善することが目標。

- [ ] M1: `browser/search.go` のリファクタリング
- [ ] M2: `browser/book.go` の gocognit/gocyclo 削減
- [ ] M3: `browser/login.go` の gocognit/gocyclo 削減
- [ ] M4: `config.go` の gocognit/gocyclo/funlen 削減
- [ ] M5: `prompts.go` の funlen 削減
- [ ] M6: `internal/git/diff.go` の gocognit 削減
- [ ] M7: `browser/search.go` SearchContent の gocognit/gocyclo 削減
- [ ] M8: `browser/book.go` convertAPIFlatTOCToLocal の gocognit 削減
- [ ] M9: `browser/book.go` parseHTMLNode の gocognit 削減

---

## アーキテクチャメトリクス技術的負債

> **追加日**: 2026-02-23
> **背景**: golangci-lint + go-arch-lint 導入時 (.golangci.yml に exclusion で暫定回避)

### M1: browser/search.go normalizeSearchResult の複雑度

**カテゴリ**: テスト可能性 / 保守性
**対象ファイル**: `browser/search.go:13`
**除外リンター**: gocognit (80 > 20), gocyclo (74 > 20), funlen, maintidx (15 < 20)

**現状**:
- 認知的複雑度 80: API レスポンスの nil-safe 正規化で多数の if-else チェーン
- maintidx が 15 で閾値の 20 を下回る

**推奨対応**:
- フィールド別の小さな正規化関数に分解してテスト可能にする

---

### M2: browser/book.go convertAPIBookDetailToLocal の gocyclo

**カテゴリ**: テスト可能性
**対象ファイル**: `browser/book.go:127`
**除外リンター**: gocognit (48 > 20), gocyclo (28 > 20)

**推奨対応**:
- フィールドカテゴリ別の変換ヘルパーに分割

---

### M3: browser/login.go runVisibleLogin の複雑度

**カテゴリ**: テスト可能性
**対象ファイル**: `browser/login.go:152`
**除外リンター**: gocognit (64 > 20), gocyclo (27 > 20), funlen, nestif

**推奨対応**:
- Chrome 起動・Cookie 待機・後処理を分離した関数に切り出す

---

### M4: config.go LoadConfig の複雑度

**カテゴリ**: テスト可能性
**対象ファイル**: `config.go:45`
**除外リンター**: gocognit (44 > 20), gocyclo (36 > 20), funlen (69 statements > 60)

**推奨対応**:
- セクション別 (HTTP, Auth, Logging, XDG) の設定ロード関数に分割

---

### M5: prompts.go registerPrompts の funlen

**カテゴリ**: 保守性
**対象ファイル**: `prompts.go:12`
**除外リンター**: funlen (128 lines > 100)

**推奨対応**:
- プロンプトごとの登録関数に分割し registerPrompts から呼び出す

---

### M6: internal/git/diff.go GetDiff の gocognit

**カテゴリ**: テスト可能性
**対象ファイル**: `internal/git/diff.go:64`
**除外リンター**: gocognit (22 > 20, 閾値超過は軽微)

**推奨対応**:
- オプションビルダーパターンに変更してネストを削減

---

### M7: browser/search.go SearchContent の複雑度

**カテゴリ**: テスト可能性
**対象ファイル**: `browser/search.go:239`
**除外リンター**: gocognit (22 > 20), gocyclo (23 > 20)

**背景**: PR #120 Copilot レビュー対応で exclusion ルールのスコープを `normalizeSearchResult` に絞り込んだ際に顕在化。

**推奨対応**:
- 検索オプションのビルド処理を分離した関数に切り出す

---

### M8: browser/book.go convertAPIFlatTOCToLocal の gocognit

**カテゴリ**: テスト可能性
**対象ファイル**: `browser/book.go:295`
**除外リンター**: gocognit (25 > 20)

**背景**: PR #120 Copilot レビュー対応で exclusion ルールのスコープを `convertAPIBookDetailToLocal` に絞り込んだ際に顕在化。

**推奨対応**:
- TOC エントリの変換ロジックを小さな関数に分解する

---

### M9: browser/book.go parseHTMLNode の gocognit

**カテゴリ**: テスト可能性
**対象ファイル**: `browser/book.go:483`
**除外リンター**: gocognit (26 > 20)

**背景**: PR #120 Copilot レビュー対応で exclusion ルールのスコープを `convertAPIBookDetailToLocal` に絞り込んだ際に顕在化。

**推奨対応**:
- HTML ノードタイプ別の処理を個別関数に分離する
