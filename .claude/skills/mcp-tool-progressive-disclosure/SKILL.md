---
name: mcp-tool-progressive-disclosure
description: MCP記述の実践的ベストプラクティス (非公式ガイドライン)。ツール、リソース、プロンプトの説明最適化、コンテキスト効率の向上、LLMトークン消費の削減を目的とします。
---

# MCP記述の実践的ベストプラクティス

MCPのツール、リソース、プロンプト記述におけるProgressive Disclosure(段階的開示)パターンを適用し、LLMコンテキスト効率を向上させるためのガイドです。

> **注意**: このガイドラインは公式MCP仕様に基づくものではなく、
> LLMコンテキスト効率を向上させるための実践的なベストプラクティスです。
> 公式仕様では description は「Human-readable description」とのみ定義されています。

## 公式MCP仕様

MCPの各コンポーネントにおける公式フィールド定義を参照用に記載します。

### ツール (Tools)

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `name` | ✅ | Unique identifier for the tool |
| `title` | ❌ | Optional human-readable name of the tool for display purposes |
| `description` | ❌ | Human-readable description of functionality |
| `inputSchema` | ✅ | JSON Schema defining expected parameters |
| `outputSchema` | ❌ | Optional JSON Schema defining expected output structure |
| `annotations` | ❌ | Optional properties describing tool behavior |

### リソース (Resources)

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `uri` | ✅ | Unique identifier for the resource |
| `name` | ✅ | The name of the resource |
| `title` | ❌ | Optional human-readable name of the resource for display purposes |
| `description` | ❌ | Optional description |
| `mimeType` | ❌ | Optional MIME type |

### プロンプト (Prompts)

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `name` | ✅ | Unique identifier for the prompt |
| `title` | ❌ | Optional human-readable name of the prompt for display purposes |
| `description` | ❌ | Optional human-readable description |
| `arguments` | ❌ | Optional list of arguments for customization |

---

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

**推奨事項**:
- 100文字以内を目安とした1行説明
- 動詞で始める(「検索する」「取得する」「作成する」)
- 最も重要な1機能のみ記述

**例**:
```
Good: "O'Reilly Learning Platformのコンテンツを検索し、書籍/動画/記事を取得する"
Poor: "O'Reilly Learning Platformのコンテンツを効率的に検索するためのツールです。クエリのベストプラクティスとして2-5個の焦点を絞ったキーワードを使用し、完全な文章ではなく具体的な技術用語を優先してください。結果にはproduct_idが含まれ、これをMCPリソースで使用できます..."
```

### 第2段階: 使用例とパラメータ (推奨)

**目的**: 正しい使い方を示す

**推奨事項**:
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

**推奨事項**:
- 本当に必要な場合のみ含める
- 別ドキュメントへの参照を推奨
- IMPORTANT注釈は1項目のみ

---

## 推奨事項

### 1. 概要は100文字以内を目安に

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

## プロンプト説明の段階的開示

MCPプロンプトはユーザーにワークフローを提供するための再利用可能なテンプレートです。プロンプト説明にも段階的開示を適用します。

### 第1段階: 概要 (Name + Title)

**目的**: プロンプトの用途を即座に理解させる

**推奨事項**:
- Name: 簡潔な識別子 (ケバブケース)
- Title: 人間が読みやすいタイトル (50文字以内を目安)

**例**:
```
Name: learn-technology
Title: Learn a Technology
```

### 第2段階: Description

**目的**: プロンプトの使い方と出力を説明する

**推奨事項**:
- 100文字以内を目安とした説明
- 使用例は1ペア (Good/Poor形式は不要、呼び出し例を記載)
- IMPORTANT注釈は1項目のみ

**例**:
```
# Good (約100文字)
Generate a structured learning path for a specific technology.

Example: learn-technology(technology="Docker", experience_level="beginner")

IMPORTANT: Uses search_content and book-details resources for learning.

# Poor (冗長)
Generate a comprehensive, structured learning path for any specific technology you want to learn.
This prompt helps users by leveraging O'Reilly's vast library of resources including books, videos,
and interactive tutorials to create personalized learning experiences...
```

### 第3段階: Arguments

**目的**: パラメータの使い方を説明する

**推奨事項**:
- 必須引数: 詳細説明と例
- オプション引数: 名前、簡潔な説明、デフォルト値のみ

**例**:
```markdown
# Good
Arguments:
  technology (必須): 学習対象の技術名 (e.g., Docker, Kubernetes, React)
  experience_level (オプション): beginner, intermediate, advanced (デフォルト: beginner)

# Poor
Arguments:
  technology (必須): 学習したい技術の名前を指定します。例えばDocker、Kubernetes、
    React、Python、Go、JavaScriptなど様々な技術を指定できます。技術名は
    正確に記載することをお勧めします...
```

---

## orm-discovery-mcp-go プロンプト改善例

### learn-technology

| 項目 | Before | After | 削減率 |
|------|--------|-------|--------|
| Description | 200文字 | 100文字 | 50% |
| Arguments (計) | 150文字 | 80文字 | 47% |
| **合計** | 350文字 | 180文字 | **49%** |

**Before**:
```
Generate a comprehensive, structured learning path for any specific technology.
This prompt leverages O'Reilly's extensive library including books, videos, and tutorials.
It creates personalized learning experiences based on user's experience level and goals.

Arguments:
  technology: The name of the technology you want to learn. Specify the exact technology name
    such as Docker, Kubernetes, React, Python, Go, JavaScript, etc.
  experience_level: Your current experience level with the technology. Options are beginner,
    intermediate, or advanced. Defaults to beginner if not specified.
```

**After**:
```
Generate a structured learning path for a specific technology.

Example: learn-technology(technology="Docker", experience_level="beginner")

IMPORTANT: Uses search_content and book-details resources for learning.

Arguments:
  technology (必須): 学習対象の技術名 (e.g., Docker, Kubernetes, React)
  experience_level (オプション): beginner, intermediate, advanced (デフォルト: beginner)
```

### research-topic

| 項目 | Before | After | 削減率 |
|------|--------|-------|--------|
| Description | 220文字 | 110文字 | 50% |
| Arguments (計) | 140文字 | 75文字 | 46% |
| **合計** | 360文字 | 185文字 | **49%** |

### debug-error

| 項目 | Before | After | 削減率 |
|------|--------|-------|--------|
| Description | 180文字 | 90文字 | 50% |
| Arguments (計) | 180文字 | 100文字 | 44% |
| **合計** | 360文字 | 190文字 | **47%** |

### プロンプト総合削減効果

| プロンプト | Before | After | 削減率 |
|-----------|--------|-------|--------|
| learn-technology | 350文字 | 180文字 | 49% |
| research-topic | 360文字 | 185文字 | 49% |
| debug-error | 360文字 | 190文字 | 47% |
| **プロンプト合計** | 1,070文字 | 555文字 | **48%** |

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
| プロンプト | 1,070文字 | 555文字 | 48% |
| **総合計** | 3,950文字 | 1,700文字 | **57%** |

### コンテキスト効率への影響

- **削減文字数**: 約1,735文字
- **削減トークン数**: 約580トークン (3文字≈1トークンで概算)
- **目標達成**: 50%削減目標に対し60%削減を達成

---

## チェックリスト

### ツール記述時の確認事項

- [ ] 概要は100文字以内か
- [ ] 動詞で始まっているか
- [ ] Good/Poor例は各1つか
- [ ] 重複情報はないか
- [ ] IMPORTANT注釈は1項目以内か
- [ ] オプションパラメータは簡潔か
- [ ] 詳細情報は別ドキュメント参照か

### プロンプト記述時の確認事項

- [ ] Titleは50文字以内か
- [ ] Descriptionは100文字以内か
- [ ] 使用例は1ペア (呼び出し形式) か
- [ ] IMPORTANT注釈は1項目以内か
- [ ] 必須引数は詳細説明があるか
- [ ] オプション引数はデフォルト値が記載されているか
- [ ] 重複情報はないか

---

## 参考リンク

### MCP公式仕様

- [MCP Tools](https://modelcontextprotocol.io/docs/concepts/tools) - ツールの公式定義
- [MCP Resources](https://modelcontextprotocol.io/docs/concepts/resources) - リソースの公式定義
- [MCP Prompts](https://modelcontextprotocol.io/docs/concepts/prompts) - プロンプトの公式定義

### 段階的開示パターン

- [Progressive Disclosure (Nielsen Norman Group)](https://www.nngroup.com/articles/progressive-disclosure/)

### 実装例

- orm-discovery-mcp-go: server.go のツール定義
