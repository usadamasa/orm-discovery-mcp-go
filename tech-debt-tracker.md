# MCP Best Practices 監査レポート

> **監査日**: 2026-02-14
> **対象**: orm-discovery-mcp-go (O'Reilly Learning Platform MCP Server)
> **監査基準**: [MCP Server Best Practices](https://modelcontextprotocol.io/specification/2025-03-26)

## サマリー

| 優先度 | 件数 | 内容 |
|--------|------|------|
| P0 (Critical) | 3 | ツール命名/アノテーション、ページネーション、HTTPセキュリティ |
| P1 (High) | 2 | URIパース、レスポンス形式 |
| P2 (Medium) | 3 | エラーメッセージ、デフォルト値、入力バリデーション |
| P3 (Low) | 5 | ドキュメント、命名、i18n |
| **合計** | **13** | |

---

## P0: Critical

### P0-001: ツール名にサービスプレフィックスがない + アノテーション未設定

**カテゴリ**: Tool Naming / Tool Annotations
**対象ファイル**: `server.go:129-152`
**ベストプラクティス参照**: [MCP Spec - Tool Annotations](https://modelcontextprotocol.io/specification/2025-03-26/server/tools#annotations)

**現状**:

```go
searchTool := &mcp.Tool{
    Name: "search_content",
    // ...
}
askQuestionTool := &mcp.Tool{
    Name: "ask_question",
    // ...
}
```

- ツール名にサービスプレフィックスがない (`search_content` → `oreilly_search_content`)
- `readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint` が未設定

**推奨対応**:

```go
searchTool := &mcp.Tool{
    Name: "oreilly_search_content",
    Annotations: &mcp.ToolAnnotations{
        ReadOnlyHint:    boolPtr(true),
        DestructiveHint: boolPtr(false),
        IdempotentHint:  boolPtr(true),
        OpenWorldHint:   boolPtr(true),
    },
    // ...
}
```

**影響**: 他の MCP サーバーとツール名が衝突するリスク。LLM がツールの副作用を正しく推測できない。

---

### P0-002: ページネーション未対応

**カテゴリ**: Pagination
**対象ファイル**: `server.go:252-320`, `tools_args.go:17-31`
**ベストプラクティス参照**: [MCP Spec - Pagination](https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/pagination)

**現状**:

- `search_content` に `offset` / `cursor` パラメータがない
- レスポンスに `has_more`, `next_offset`, `nextCursor` がない
- デフォルト `rows=100` で全件を一括返却 (line 262-263)

**推奨対応**:

1. `SearchContentArgs` に `offset int` と `limit int` を追加
2. `SearchContentResult` に `HasMore bool` と `NextOffset int` を追加
3. デフォルト limit を 20-50 に変更
4. BFS モードでもページネーションをサポート

**影響**: 大量結果時のコンテキストウィンドウ圧迫。LLM が必要以上のトークンを消費する。

---

### P0-003: HTTP サーバーが全インターフェースにバインド

**カテゴリ**: Security / Network Binding
**対象ファイル**: `main.go:74`, `server.go:71-104`
**ベストプラクティス参照**: [MCP Spec - Security Best Practices](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#security)

**現状**:

```go
// main.go:74
s.StartStreamableHTTPServer(ctx, fmt.Sprintf(":%s", cfg.Port))

// server.go:84
httpServer := &http.Server{
    Addr:    port, // ":8080" = 全インターフェース
    Handler: handler,
}
```

- `:8080` は `0.0.0.0:8080` と同等で、外部からアクセス可能
- Origin ヘッダー検証なし (DNS rebinding 対策未実装)

**推奨対応**:

1. ローカル用途の場合: `127.0.0.1:8080` にバインド
2. Origin ヘッダー検証ミドルウェアを追加
3. 環境変数 `BIND_ADDRESS` で設定可能にする

**影響**: ローカルマシンの認証情報が外部ネットワークに露出する可能性。

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

### P2-002: デフォルト rows=100 が大きすぎる

**カテゴリ**: Performance / Context Window
**対象ファイル**: `server.go:262-263`
**ベストプラクティス参照**: MCP Best Practices - Pagination Default Size

**現状**:

```go
if args.Rows <= 0 {
    args.Rows = 100
}
```

- ベストプラクティスでは 20-50 が推奨
- BFS モードでも 100 件のメタデータを返すとコンテキストを圧迫

**推奨対応**:

```go
if args.Rows <= 0 {
    args.Rows = 25 // ベストプラクティス推奨値
}
if args.Rows > 100 {
    args.Rows = 100 // 上限
}
```

**影響**: LLM のコンテキストウィンドウを不必要に消費。

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

### Phase 1: セキュリティ/安定性 (P0)

- [x] P0-001: ツール名プレフィックス追加 + アノテーション設定
- [ ] P0-002: ページネーション実装
- [ ] P0-003: HTTP バインドアドレス修正 + Origin 検証

### Phase 2: 堅牢性 (P1)

- [ ] P1-001: URI パースを `url.Parse()` ベースに統一
- [ ] P1-002: Markdown レスポンス形式サポート

### Phase 3: 品質改善 (P2)

- [ ] P2-001: エラーメッセージの分離
- [ ] P2-002: デフォルト rows 値の変更
- [ ] P2-003: 入力バリデーション強化

### Phase 4: ドキュメント/仕上げ (P3)

- [ ] P3-001: ツール説明の例を拡充
- [ ] P3-002: レート制限ドキュメント追加
- [ ] P3-003: パフォーマンス特性ドキュメント追加
- [ ] P3-004: サーバー実装名の修正
- [ ] P3-005: 日本語ストップワード対応
