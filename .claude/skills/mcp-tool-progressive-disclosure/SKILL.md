---
name: mcp-tool-progressive-disclosure
description: Claude Code向けMCP記述最適化ガイド。mcp-go SDKでのツール/リソース/プロンプト説明の書き方、コンテキスト効率戦略を提供する。「ツール説明が長い」「description を短くしたい」「MCP記述を最適化」「トークン消費を減らしたい」「tools/list が重い」のようにMCPメタデータのサイズ最適化に関する依頼を受けたときに使用する。
---

# MCP記述の実践的ベストプラクティス (Claude Code + mcp-go)

Claude Code が MCP メタデータをどう消費するかを理解し、description のトークン消費を最小化するガイド。

## Claude Code のメタデータ消費モデル

| リスト応答 | 消費タイミング | LLM コンテキストへの影響 |
|-----------|-------------|----------------------|
| `tools/list` | **毎リクエスト** | `description` が全量注入される |
| `resources/list` | リソース参照時 | `description` が注入される |
| `prompts/list` | プロンプト参照時 | `description` が注入される |

### Claude Code 固有の仕組み

- **Deferred Tools**: `tools/list` に名前だけ表示され、description はゼロトークン。ユーザーが `ToolSearch` で選択して初めて description がロードされる
- **Server Instructions**: `initialize` レスポンスで一度だけ送信。全ツール共通の注意事項 (引用ルール等) はここに集約する
- **title vs description vs name**: `title` は UI 表示用 (人間向け)、`description` は LLM コンテキスト用 (ツール選択判断)、`name` はプログラマティック識別子。同じ内容を複数に書かない

## コンテキスト効率戦略 (インパクト順)

### 1. Server Instructions で横断的関心を集約

各ツールの description に繰り返し書いていた共通ルールを `ServerOptions.Instructions` に一本化する。

```go
mcpServer := mcp.NewServer(
    &mcp.Implementation{Name: "my-server", Version: "1.0.0"},
    &mcp.ServerOptions{
        Instructions: "Use search_content to discover content, " +
            "ask_question for AI-powered Q&A, " +
            "and book-* resources for detailed access. " +
            "Always cite sources with title, author(s), and publisher.",
    },
)
```

**効果**: 各ツールから `IMPORTANT: Cite sources...` を削除でき、N ツール × M 文字の重複を排除。

### 2. ToolAnnotations で振る舞い記述を置き換え

description に `"This is a read-only tool"` `"WARNING: destructive"` と書く代わりに、`ToolAnnotations` の構造化ヒントを使う。

**効果**: 振る舞いの自然言語記述を削除でき、クライアントが機械的に判断できる。

> 実装詳細・コード例は **mcp-go-sdk-practices** スキルの「ToolAnnotations」セクションを参照。

### 3. outputSchema で出力説明を代替

`AddTool[In, Out]` で `Out` 型を定義すると、SDK が outputSchema を自動生成する。description から `"Response includes: ..."` セクションを削除できる。

**効果**: レスポンス構造の説明を description から除去。型定義が Single Source of Truth になる。

> 実装詳細は **mcp-go-sdk-practices** スキルの「AddTool[In, Out] ジェネリクス」「Out 型の設計パターン」セクションを参照。

### 4. inputSchema パラメータとの重複排除

`jsonschema` タグで各パラメータに description を付けている場合、ツール description にパラメータ説明を繰り返さない。
例: `jsonschema:"description=2-5 focused keywords"` と定義済みなら、description に `"query: 2-5 keywords"` と書く必要なし。

## description 記述ルール (3段階)

### Tier 1: 概要 (必須)

- 100文字以内、動詞で始める (Search / Ask / Get / Generate)
- ツール選択に十分な情報のみ。パラメータ詳細は inputSchema に委ねる

```
Good: "Search content and return items with product_id for resource access."
Poor: "Efficiently search the platform's content library. Use 2-5 focused keywords..."
```

### Tier 2: 使用例 (推奨)

- Good/Poor 1ペアのみ。ユーザーが正しい入力パターンを直感的に理解できる程度

```
Example: "Docker containers" (Good) / "How to use Docker" (Poor)
```

### Tier 3: 注意事項 (最小限)

- `IMPORTANT` 注釈は1項目以内
- Server Instructions で代替できないか検討してから追加する
- `ReadOnlyHint` 等で表現できる内容は ToolAnnotations に移す

## リソース / テンプレートの重複排除

Resource と ResourceTemplate で同じ URI パターンを持つ場合:

- **Resource description**: 詳細説明 (何が取得できるか)
- **Template description**: 最小限 (どのパラメータで呼ぶか)

```go
// Resource — 詳細
&mcp.Resource{
    URI:         "myapp://book-details/12345",
    Description: "Get book info (title, ISBN, description, publication date).",
}
// Template — 最小限
&mcp.ResourceTemplate{
    URITemplate: "myapp://book-details/{product_id}",
    Description: "Use product_id from search results to get book details.",
}
```

## チェックリスト

### メタデータ全般

- [ ] Server Instructions に横断的ルール (引用、認証、制限) を集約したか
- [ ] 各 description から Server Instructions と重複するテキストを削除したか

### ツール

- [ ] description は100文字以内で動詞始まりか
- [ ] ToolAnnotations で read-only / destructive / idempotent を設定したか
- [ ] inputSchema の jsonschema タグと description で情報が重複していないか
- [ ] outputSchema (AddTool[In, Out]) で出力説明を代替し、description から削除したか
- [ ] Good/Poor 例は各1つか
- [ ] IMPORTANT 注釈は1項目以内か (Server Instructions で代替を検討)

### リソース / テンプレート

- [ ] Resource に詳細説明、Template は最小限 (参照方法のみ) か
- [ ] 同じ説明が Resource と Template の両方にないか

### プロンプト

- [ ] description は100文字以内か
- [ ] arguments の説明は jsonschema タグに委ね、description と重複していないか

## MCP 仕様リファレンス (mcp-go SDK v1.4.0)

**Tool**: `name`(必須), `title`, `description`, `inputSchema`(必須), `outputSchema`, `annotations: *ToolAnnotations`

**ToolAnnotations**: `readOnlyHint`(default:false), `destructiveHint`(default:true, readOnly=false時のみ), `idempotentHint`(default:false), `openWorldHint`(default:true), `title`

**Resource / ResourceTemplate**: `uri`/`uriTemplate`(必須), `name`(必須), `title`, `description`, `mimeType`, `size`(Resourceのみ), `annotations: *Annotations`

**Prompt**: `name`(必須), `title`, `description`, `arguments`
