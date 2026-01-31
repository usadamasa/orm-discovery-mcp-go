---
paths:
  - "Taskfile.yml"
  - "aqua.yaml"
---

# 開発ツール

このルールは、開発ツール関連ファイルを編集する際に適用される。

## aqua (Package Manager)

[aqua](https://aquaproj.github.io/) によるツール管理:

```bash
# aqua.yaml で定義されたツールをインストール
aqua install

# 利用可能なツールを表示
aqua list
```

### Managed Tools

| Tool | Version | 役割 |
|------|---------|------|
| `go-task/task` | v3.44.0 | タスクランナー |
| `golang/tools/goimports` | v0.34.0 | Go imports フォーマッタ |
| `golangci/golangci-lint` | v2.1.6 | Go リンター |
| `deepmap/oapi-codegen` | v2.4.1 | OpenAPI コード生成 |

## Task (Task Runner)

[Task](https://taskfile.dev/) によるビルド自動化:

```bash
# タスク一覧
task --list

# 開発ワークフロー
task dev              # Format + Lint + Test
task ci               # Complete CI workflow
task check            # Quick code quality check

# 個別タスク
task format           # Format code
task lint             # Run linter
task test             # Run tests
task build            # Build binary
task generate:api:oreilly  # Generate OpenAPI client code

# クリーニング
task clean:all        # Clean everything
task clean:generated  # Clean generated code only
```

## Task Categories and Dependencies

### Code Generation

- `generate:api:oreilly` - OpenAPI spec からクライアント生成

### Code Quality

- `format` - goimports と go fmt でコード整形
- `lint` - golangci-lint 実行 (format に依存)

### Testing

- `test` - Go テスト実行 (generate:api:oreilly に依存)
- `test:coverage` - カバレッジレポート付きテスト

### Building

- `build` - 標準ビルド (generate:api:oreilly, lint に依存)
- `build:release` - 最適化リリースビルド (generate:api:oreilly, lint, test に依存)

### Composite Workflows

| Workflow | 構成 |
|----------|------|
| `dev` | format + lint + test |
| `ci` | generate + format + lint + test:coverage + build |
| `check` | format + lint |

## OpenAPI Code Generation

OpenAPI 仕様とコード生成:

| 項目 | パス |
|------|------|
| Spec file | `browser/openapi.yaml` |
| Config file | `browser/oapi-codegen.yaml` |
| Output directory | `browser/generated/api/` |
| Tool | [oapi-codegen](https://github.com/deepmap/oapi-codegen) |

## Build and Development Commands

### Build the project

```bash
task build
```

### Run the MCP server (stdio mode)

```bash
bin/orm-discovery-mcp-go
```

### Run HTTP server mode

```bash
source .env
bin/orm-discovery-mcp-go
```

### Update dependencies

```bash
task format
```
