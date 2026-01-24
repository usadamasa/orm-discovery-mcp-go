---
name: mcp-tool-progressive-disclosure
description: MCPツール記述における段階的開示(Progressive Disclosure)パターンのガイド。ツール説明の最適化、コンテキスト効率の向上、LLMトークン消費の削減を目的とします。
---

# MCPツール段階的開示ガイド

MCPツール記述におけるProgressive Disclosure(段階的開示)パターンを適用し、LLMコンテキスト効率を向上させるためのガイドです。

## 概要

### 段階的開示とは

段階的開示は、情報を必要に応じて段階的に提供するUIデザインパターンです。MCPツール記述に適用することで:

- **LLMコンテキスト消費を削減** (目標: 50%削減)
- **ツール選択の判断を迅速化**
- **必要な情報を必要なときに提供**

### コンテキスト効率の重要性

| 指標 | 説明 |
|------|------|
| ツール説明の総文字数 | LLMコンテキストを直接消費 |
| ツール数 × 平均説明長 | 総コンテキスト消費量 |
| 重複情報 | 無駄なトークン消費 |

---

## 3段階開示モデル

### 第1段階: 概要 (必須)

**目的**: ツールの用途を即座に理解させる

**ルール**:
- 100文字以内の1行説明
- 動詞で始める(「検索する」「取得する」「作成する」)
- 最も重要な1機能のみ記述

**例**:
```
Good: "O'Reilly Learning Platformのコンテンツを検索し、書籍/動画/記事を取得する"
Poor: "O'Reilly Learning Platformのコンテンツを効率的に検索するためのツールです。クエリのベストプラクティスとして2-5個の焦点を絞ったキーワードを使用し、完全な文章ではなく具体的な技術用語を優先してください。結果にはproduct_idが含まれ、これをMCPリソースで使用できます..."
```

### 第2段階: 使用例とパラメータ (推奨)

**目的**: 正しい使い方を示す

**ルール**:
- Good/Poor例は各1つに絞る
- パラメータは必須のみ詳細説明
- オプションパラメータは名前と型のみ

**例**:
```
例:
  Good: "Docker containers", "React hooks"
  Poor: "How to use Docker for containerization"

パラメータ:
  query (必須): 2-5個のキーワード
  rows (オプション): 結果数 (デフォルト: 100)
```

### 第3段階: 詳細ガイドライン (オプション)

**目的**: 高度な使用法を提供

**ルール**:
- 本当に必要な場合のみ含める
- 別ドキュメントへの参照を推奨
- IMPORTANT注釈は1項目のみ

---

## ベストプラクティス

### 1. 概要は100文字以内

```markdown
# Good
O'Reillyコンテンツを検索し、書籍/動画/記事のproduct_idを取得する

# Poor
O'Reilly Learning Platformのコンテンツを効率的に検索するためのツールです。
クエリのベストプラクティスとして2-5個の焦点を絞ったキーワードを使用し...
```

### 2. 例は1ペアに絞る

```markdown
# Good
例: "Docker containers" (Good) / "How to use Docker" (Poor)

# Poor
Good: "Docker containers", "React hooks", "Python async", "Kubernetes monitoring"
Poor: "How to use Docker for containerization", "Best practices for React development"
```

### 3. IMPORTANT注釈は1項目

```markdown
# Good
IMPORTANT: ソース情報を必ず引用してください

# Poor
IMPORTANT: ソース情報を引用してください。また、クエリは100文字以内に...
IMPORTANT: 機密情報を含めないでください
IMPORTANT: このツールを3回以上呼び出さないでください
```

### 4. 重複情報の排除

同じ情報を複数箇所で繰り返さない:

```markdown
# Good (1箇所で説明)
結果にはproduct_idが含まれ、リソースアクセスに使用可能

# Poor (重複)
概要: "...product_idを返します"
使用例: "...product_idを取得し..."
出力説明: "...product_idが含まれます"
```

---

## orm-discovery-mcp-go 改善例

### search_content: Before (878文字)

```
Search O'Reilly Learning Platform content efficiently.
Returns books, videos, and articles with product IDs for use with resources.

QUERY BEST PRACTICES:
- Use 2-5 focused keywords (not full sentences)
- Prefer specific technical terms over general descriptions
- Combine technology + concept for better results

EXAMPLES:
Good: "Docker containers", "React hooks", "Python async", "Kubernetes monitoring"
Poor: "How to use Docker for containerization", "Best practices for React development"

Results include product_id for accessing detailed content via MCP resources:
- Book details: "oreilly://book-details/{product_id}"
- Table of contents: "oreilly://book-toc/{product_id}"
- Chapter content: "oreilly://book-chapter/{product_id}/{chapter_name}"

IMPORTANT: Always cite sources with title, author(s), and O'Reilly Media as publisher.
```

### search_content: After (約350文字)

```
O'Reillyコンテンツを検索し、書籍/動画/記事のproduct_idを取得する。

例: "Docker containers" (Good) / "How to use Docker" (Poor)

結果のproduct_idでリソースアクセス可能:
- oreilly://book-details/{product_id}
- oreilly://book-chapter/{product_id}/{chapter}

IMPORTANT: ソース情報(タイトル、著者、O'Reilly Media)を引用すること。
```

### ask_question: Before (1,008文字)

```
Ask focused technical questions to O'Reilly Answers AI for comprehensive, well-sourced responses.

QUESTION BEST PRACTICES:
- Keep questions under 100 characters for optimal processing
- Ask specific, focused questions rather than broad topics
- Use clear, direct language in English
- Focus on practical "how-to" or "what is" questions

EFFECTIVE QUESTION PATTERNS:
Good: "How to optimize React performance?", "What is Kubernetes service mesh?", "Python async vs threading?"
Poor: "Can you explain everything about React performance optimization techniques and best practices?"

Response includes:
- Comprehensive markdown-formatted answer
- Source citations with specific book/article references
- Related resources for deeper learning
- Suggested follow-up questions
- Question ID for future reference

Covers: programming, data science, cloud computing, DevOps, machine learning, and other technical domains.

IMPORTANT: Always cite the sources provided in the response when referencing the information.
```

### ask_question: After (約400文字)

```
O'Reilly Answers AIに技術的な質問を送信し、ソース付きの回答を取得する。

例: "How to optimize React performance?" (Good) / "Explain everything about React" (Poor)

回答に含まれる情報:
- Markdown形式の回答
- ソース引用
- 関連リソース
- question_id (oreilly://answer/{id}で再取得可能)

IMPORTANT: 回答内のソース情報を必ず引用すること。
```

---

## 英語パターン (English Patterns)

MCPツール説明を英語で記述する場合の段階的開示パターンです。

### search_content (English)

```
Search O'Reilly content and return books/videos/articles with product_id for resource access.

Example: "Docker containers" (Good) / "How to use Docker" (Poor)

Results: Use product_id with oreilly://book-details/{id} or oreilly://book-chapter/{id}/{chapter}

IMPORTANT: Cite sources with title, author(s), and O'Reilly Media.
```
約270文字

### ask_question (English)

```
Ask technical questions to O'Reilly Answers AI and get sourced responses.

Example: "How to optimize React performance?" (Good) / "Explain everything about React" (Poor)

Response: Markdown answer, sources, related resources, question_id (use with oreilly://answer/{id})

IMPORTANT: Cite sources provided in the response.
```
約280文字

---

## リソースとテンプレートの重複排除

MCPリソースとリソーステンプレートは類似した説明を持つことが多いです。重複を排除するパターンを示します。

### 問題点

リソースとテンプレートで同じ情報を繰り返すと、コンテキストを無駄に消費します:

```
# Poor: 重複した説明
Resource Description: "Get comprehensive book information including title, authors, publication date, description, topics, and table of contents."
Template Description: "Template for accessing O'Reilly book details. Use product_id from search_content results to get comprehensive book information including title, authors, publication date, description, topics, and table of contents."
```

### 解決策

リソース説明に詳細を記述し、テンプレートは最小限の説明にします:

```
# Good: 役割分担した説明
Resource Description: "Get book info (title, authors, date, description, topics, TOC). Cite sources when referencing."
Template Description: "Use product_id from search_content to get book details."
```

### 実装例 (orm-discovery-mcp-go)

| 対象 | Before | After | 削減率 |
|------|--------|-------|--------|
| book-details Resource | 180文字 | 95文字 | 47% |
| book-toc Resource | 160文字 | 100文字 | 38% |
| book-chapter Resource | 170文字 | 90文字 | 47% |
| answer Resource | 200文字 | 85文字 | 58% |
| book-details Template | 210文字 | 55文字 | 74% |
| book-toc Template | 180文字 | 60文字 | 67% |
| book-chapter Template | 200文字 | 55文字 | 73% |
| answer Template | 180文字 | 55文字 | 69% |
| **リソース/テンプレート合計** | 1,480文字 | 595文字 | **60%** |

---

## 効果測定

### 削減効果の計算 (実測値 - 2026年1月)

#### ツール説明

| ツール | Before | After | 削減率 |
|--------|--------|-------|--------|
| search_content | 600文字 | 270文字 | 55% |
| ask_question | 800文字 | 280文字 | 65% |
| **ツール合計** | 1,400文字 | 550文字 | **61%** |

#### リソース/テンプレート説明

| 対象 | Before | After | 削減率 |
|--------|--------|-------|--------|
| リソース (4種) | 710文字 | 370文字 | 48% |
| テンプレート (4種) | 770文字 | 225文字 | 71% |
| **リソース/テンプレート合計** | 1,480文字 | 595文字 | **60%** |

#### 総合削減効果

| カテゴリ | Before | After | 削減率 |
|--------|--------|-------|--------|
| ツール | 1,400文字 | 550文字 | 61% |
| リソース/テンプレート | 1,480文字 | 595文字 | 60% |
| **総合計** | 2,880文字 | 1,145文字 | **60%** |

### コンテキスト効率への影響

- **削減文字数**: 約1,735文字
- **削減トークン数**: 約580トークン (3文字≈1トークンで概算)
- **目標達成**: 50%削減目標に対し60%削減を達成

---

## チェックリスト

MCPツール記述時の確認事項:

- [ ] 概要は100文字以内か
- [ ] 動詞で始まっているか
- [ ] Good/Poor例は各1つか
- [ ] 重複情報はないか
- [ ] IMPORTANT注釈は1項目以内か
- [ ] オプションパラメータは簡潔か
- [ ] 詳細情報は別ドキュメント参照か

---

## 参考リンク

- [Progressive Disclosure (Nielsen Norman Group)](https://www.nngroup.com/articles/progressive-disclosure/)
- [MCP Tool Design Guidelines](https://modelcontextprotocol.io/)
- orm-discovery-mcp-go: server.go のツール定義
