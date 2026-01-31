---
paths:
  - "**.go"
---

# コード品質保証

このルールは、Go コードファイルを編集する際に適用される。

## CRITICAL REQUIREMENT: Test and Build Verification

**MANDATORY**: 開発タスク完了前に、テストとビルドの成功を必ず確認すること。これはコード品質とプロジェクト安定性を維持するための非交渉要件。

## Required Verification Steps

**いかなるコード変更に対しても、必ず実行:**

```bash
task ci    # テストとビルドを含む完全 CI ワークフロー
```

**`task ci` が失敗した場合、すべての問題が解決するまでタスクは完了していない。**

## Alternative Verification Commands

個別ステップを実行する必要がある場合:

```bash
# Step 1: コード品質確認
task check              # Format + Lint

# Step 2: 機能確認
task test              # すべてのテスト実行

# Step 3: ビルド確認
task build             # プロジェクトビルド
```

## What Must Pass

### 1. Code Quality Checks

- `task format` - コードフォーマットの一貫性
- `task lint` - すべてのリンティングルールをパス (0 issues)

### 2. Functionality Tests

- `task test` - すべてのテストがエラーなくパス
- テスト失敗やパニックは許可されない

### 3. Build Verification

- `task build` - プロジェクトが正常にコンパイル
- コンパイルエラーは許可されない

## When to Run Verification

**以下の後は必ず検証を実行:**

- 新しいコードや機能の追加
- 既存コードの修正
- リファクタリング
- 依存関係の更新
- 設定変更
- コミット前

## Failure Resolution

**検証ステップが失敗した場合:**

1. **即座に問題を修正** - 他のタスクに進まない
2. **失敗したステップを再実行** して修正を確認
3. **`task ci` を実行** してプロジェクト全体の健全性を確保
4. **その後にのみタスク完了とみなす**

## Exception Policy

**この要件に例外はない。** 以下の場合も含む:

- ドキュメントのみの変更 (build/generate タスクに影響する可能性)
- 設定の更新 (機能に影響する可能性)
- "マイナー" なコード変更 (予期せぬ副作用の可能性)

## CI Integration

GitHub Actions CI パイプラインも同じ要件を強制:

- `task ci` のすべてのタスクが PR マージの条件
- ローカル検証で CI 失敗を防ぎ、開発を加速

## Task Completion Checklist

- [ ] コード変更を実装
- [ ] `task ci` が正常に実行された
- [ ] すべてのテストがパス
- [ ] ビルドが成功
- [ ] リンティングエラーなし
- [ ] タスク完了

**Remember: タスクは `task ci` がエラーなくパスして初めて完了。**

## Code Quality Requirements

- すべてのコードは `golangci-lint` で 0 issues をパス
- `goimports` と `go fmt` によるコードフォーマット
- タスク完了前にすべてのテストをパス
