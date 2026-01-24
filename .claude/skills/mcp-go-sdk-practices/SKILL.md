# mcp-go-sdk-practices

modelcontextprotocol/go-sdk (mcp-go) を使用したMCPサーバー実装のベストプラクティス集。

---

## StructuredContent の概念

### 概要

MCP SDK v1.2.0 以降では、ツールの戻り値として **StructuredContent** がサポートされている。これにより、クライアントは構造化されたデータを直接受け取ることができる。

### Content vs StructuredContent

| フィールド | 型 | 用途 |
|------------|------|------|
| `content` | `[]mcp.Content` | テキスト/画像などの表示用コンテンツ |
| `structuredContent` | `any` | プログラムが解析可能な構造化データ |

**重要**: `structuredContent` が `nil` または空のオブジェクト `{}` の場合、クライアントは結果が空だと判断する可能性がある。

---

## AddTool[In, Out] ジェネリクス

### 関数シグネチャ

```go
func AddTool[In, Out any](
    server *mcp.Server,
    tool *mcp.Tool,
    handler func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error),
)
```

### 型パラメータ

| パラメータ | 説明 |
|------------|------|
| `In` | ツール引数の型（リクエストパラメータ） |
| `Out` | 構造化出力の型（StructuredContent） |

### ハンドラの戻り値

| 戻り値 | 説明 |
|--------|------|
| `*mcp.CallToolResult` | Contentを手動で設定する場合。`nil` の場合はSDKが自動生成 |
| `Out` | StructuredContentに設定される値。SDKがJSONにシリアライズ |
| `error` | エラーが発生した場合のエラーオブジェクト |

---

## Out 型の設計パターン

### パターン1: 構造体を使用（推奨）

```go
// 出力型を定義
type SearchContentResult struct {
    Count   int                      `json:"count"`
    Total   int                      `json:"total"`
    Results []map[string]interface{} `json:"results"`
}

// ハンドラで使用
func (s *Server) SearchContentHandler(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args SearchContentArgs,
) (*mcp.CallToolResult, *SearchContentResult, error) {
    // ... 処理 ...
    result := &SearchContentResult{
        Count:   len(results),
        Total:   len(results),
        Results: results,
    }
    return nil, result, nil  // SDK が自動で Content と StructuredContent を設定
}
```

### パターン2: ポインタ型を使用

```go
// ポインタ型を使うと nil で空結果を表現可能
func (s *Server) Handler(...) (*mcp.CallToolResult, *MyResult, error) {
    if notFound {
        return nil, nil, nil  // StructuredContent が null になる
    }
    return nil, &MyResult{...}, nil
}
```

### パターン3: any を使用（非推奨）

```go
// any は nil を返しやすく、問題の原因になる
func (s *Server) Handler(...) (*mcp.CallToolResult, any, error) {
    return newToolResultText(jsonString), nil, nil  // NG: structuredContent が {} になる
}
```

---

## SDK の自動処理

### CallToolResult が nil の場合

SDKは以下を自動的に行う:

1. `Out` を JSON にシリアライズ
2. `Content` に `TextContent` として設定
3. `StructuredContent` に元の構造体を設定

```go
// 入力
return nil, &MyResult{Count: 5}, nil

// SDKによる変換後のレスポンス
{
    "content": [{"type": "text", "text": "{\"count\":5}"}],
    "structuredContent": {"count": 5}
}
```

### CallToolResult を手動で設定する場合

`CallToolResult` を返すと、SDKは `Content` を上書きしない。ただし `StructuredContent` は `Out` から設定される。

```go
// 手動でContent設定 + Out指定
return &mcp.CallToolResult{
    Content: []mcp.Content{&mcp.TextContent{Text: "カスタムメッセージ"}},
}, &MyResult{Count: 5}, nil

// 結果
{
    "content": [{"type": "text", "text": "カスタムメッセージ"}],
    "structuredContent": {"count": 5}
}
```

---

## エラーハンドリング

### パターン1: error を返す

```go
func (s *Server) Handler(...) (*mcp.CallToolResult, *MyResult, error) {
    if err != nil {
        return nil, nil, fmt.Errorf("処理に失敗しました: %w", err)
    }
    return nil, result, nil
}
```

### パターン2: IsError フラグを使用

```go
func (s *Server) Handler(...) (*mcp.CallToolResult, *MyResult, error) {
    if validationError {
        return &mcp.CallToolResult{
            Content: []mcp.Content{&mcp.TextContent{Text: "バリデーションエラー: " + msg}},
            IsError: true,
        }, nil, nil
    }
    return nil, result, nil
}
```

### 使い分け

| パターン | 使用場面 |
|----------|----------|
| `error` を返す | 予期しないシステムエラー |
| `IsError: true` | ユーザー入力のバリデーションエラーなど |

---

## 移行チェックリスト

既存コードを StructuredContent 対応に移行する際のチェックリスト:

- [ ] 出力型の構造体を定義する（`tools_args.go` など）
- [ ] ハンドラの戻り値の型を `any` から具体的な型に変更
- [ ] `return newToolResultText(json), nil, nil` を `return nil, result, nil` に変更
- [ ] `newToolResultText()` ヘルパー関数が不要なら削除
- [ ] `newToolResultError()` は `IsError: true` 用に残すかどうか検討
- [ ] ビルドとテストで動作確認

---

## 実装例

### Before（問題のあるコード）

```go
func (s *Server) SearchContentHandler(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args SearchContentArgs,
) (*mcp.CallToolResult, any, error) {
    // ... 処理 ...
    jsonBytes, _ := json.Marshal(response)
    return newToolResultText(string(jsonBytes)), nil, nil  // Out が nil
}
```

### After（修正後のコード）

```go
func (s *Server) SearchContentHandler(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args SearchContentArgs,
) (*mcp.CallToolResult, *SearchContentResult, error) {
    // ... 処理 ...
    result := &SearchContentResult{
        Count:   len(results),
        Total:   len(results),
        Results: results,
    }
    return nil, result, nil  // SDK が自動で設定
}
```

---

## 検証方法

### MCP Inspector を使用

```bash
# MCP Inspector でツールを呼び出し
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_content","arguments":{"query":"Go"}}}' \
| ./bin/orm-discovery-mcp-go 2>&1 | jq '.result.structuredContent'
```

### 期待される結果

```json
{
  "count": 10,
  "total": 10,
  "results": [...]
}
```

`structuredContent` が空のオブジェクト `{}` ではなく、実際のデータが入っていることを確認する。

---

## 参考リンク

- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- [MCP Specification - Tool Results](https://spec.modelcontextprotocol.io/specification/server/tools/)
