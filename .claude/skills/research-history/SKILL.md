---
name: research-history
description: |
  O'Reilly MCP サーバーの Research History 機能についてのガイド。
  過去の調査履歴のデータ構造、検索方法、活用パターンを理解する際に使用。

  Use when:
  - Research History のデータ構造を理解したいとき
  - 過去の調査を検索・参照する方法を知りたいとき
  - review-history / continue-research プロンプトの動作を理解したいとき
  - 調査履歴を活用した開発を行うとき
---

# Research History

O'Reilly MCP サーバーが保存する調査履歴の仕組みと活用方法。

## データ構造

### ファイル位置

```
$XDG_STATE_HOME/research-history.json
(通常: ~/.local/state/orm-mcp-go/research-history.json)
```

### JSON スキーマ

```json
{
  "version": 1,
  "last_updated": "ISO8601 timestamp",
  "entries": [ResearchEntry],
  "index": {
    "by_keyword": {"keyword": ["entry_id", ...]},
    "by_type": {"search|question": ["entry_id", ...]},
    "by_date": {"YYYY-MM-DD": ["entry_id", ...]}
  }
}
```

### ResearchEntry の構造

| フィールド | 型 | 説明 |
|-----------|-----|------|
| `id` | string | 一意識別子 (req_xxx) |
| `timestamp` | string | ISO8601 タイムスタンプ |
| `type` | string | "search" or "question" |
| `query` | string | ユーザーのクエリ文字列 |
| `keywords` | []string | クエリから抽出したキーワード |
| `tool_name` | string | 使用したツール名 |
| `parameters` | object | ツールに渡したパラメータ |
| `result_summary` | object | 結果のサマリー (後述) |
| `duration_ms` | int | 実行時間 (ミリ秒) |

### result_summary (タイプ別)

**search の場合:**
```json
{
  "count": 15,
  "top_results": [
    {"title": "書籍名", "author": "著者", "product_id": "ID"}
  ]
}
```

**question の場合:**
```json
{
  "answer_preview": "回答の冒頭200文字...",
  "sources_count": 5,
  "followup_count": 3
}
```

## MCP リソース

| URI | 説明 |
|-----|------|
| `orm-mcp://history/recent` | 直近20件の調査履歴 |
| `orm-mcp://history/search?keyword={kw}` | キーワード検索 |
| `orm-mcp://history/search?type={type}` | タイプ絞り込み |
| `orm-mcp://history/{id}` | 特定エントリの詳細 |

## MCP プロンプト

### review-history

過去の調査を振り返る。

```
Arguments:
  - keyword (optional): 検索キーワード

Workflow:
1. orm-mcp://history/recent で直近の調査を取得
2. keyword があれば orm-mcp://history/search で絞り込み
3. 過去の調査結果を要約して表示
4. 関連する追加調査を提案
```

### continue-research

過去の調査を深掘りする。

```
Arguments:
  - research_id (required): 過去の調査ID

Workflow:
1. orm-mcp://history/{research_id} で詳細を取得
2. 元のクエリと結果を確認
3. 同じトピックで追加の検索/質問を実行
4. 新しい発見を過去の結果と統合して報告
```

## 活用パターン

### パターン1: 関連調査の発見

```
User: "前に Docker について調べたことある？"
Claude: orm-mcp://history/search?keyword=docker にアクセス
→ 過去の Docker 関連調査を一覧表示
```

### パターン2: 調査の継続

```
User: "昨日の Kubernetes 調査を続けて"
Claude: orm-mcp://history/recent で直近を確認
→ Kubernetes 関連のエントリを特定
→ continue-research プロンプトで深掘り
```

### パターン3: 知識の統合

```
User: "React と Vue の比較結果をまとめて"
Claude:
  orm-mcp://history/search?keyword=react
  orm-mcp://history/search?keyword=vue
→ 両方の過去調査を取得して統合レポート作成
```

## 制限事項

- 最大1000件まで保持 (古いものから自動削除)
- レスポンス全体は保存しない (サマリーのみ)
- キーワードはクエリから単純分割 (形態素解析なし)

## 環境変数

```bash
# Research History 設定
ORM_MCP_GO_HISTORY_MAX_ENTRIES=1000  # 保持する最大エントリ数
```
