---
name: login-diagnosis
description: O'Reillyログイン失敗時の診断と対処ガイド。CDN (Akamai) ブロック、Cookie有効期限切れ、ChromeDP設定問題を段階的に切り分ける。「ログインできない」「Access Denied」「タイムアウト」等のトリガーに対応。
---

# O'Reilly ログイン診断ガイド

ログイン失敗時の体系的な診断と対処を提供する。

## 前提知識

### 認証アーキテクチャ

```
Cookie復元 → HTTP検証 → [成功: Cookie有効]
                      → [失敗: ChromeDP起動 → ブラウザログイン → Cookie保存]
```

- **Cookie-first**: 保存済みCookieで認証を試行 (ChromeDP不要)
- **フォールバック**: Cookie無効時のみChromeDPでブラウザログインを実行
- **認証後クローズ**: 非デバッグモードではブラウザを即座にクローズ

### Akamai Bot Manager

O'Reilly は Akamai CDN の Bot Manager を使用。以下の検出メカニズムがある:

| 検出手法 | 説明 |
|---------|------|
| `_abck` Cookie | ボット判定用Cookie。センサーJS実行後に設定される |
| `DefaultExecAllocatorOptions` | ChromeDPのデフォルトフラグ群 (自動化指標として検出) |
| TLS fingerprint | 自動化ツール固有のTLSハンドシェイク |
| レート制限 | 短時間の連続リクエストでブロック |

## 診断フロー

### Phase 1: 症状の特定

```bash
# ログ確認
ORM_MCP_GO_DEBUG=true ORM_MCP_GO_LOG_LEVEL=DEBUG bin/orm-discovery-mcp-go 2>&1 | head -20
```

| 症状 | 推定原因 | 次のステップ |
|------|---------|-------------|
| `ログインタイムアウト` + DEBUGログなし | ログレベル未設定 | `ORM_MCP_GO_LOG_LEVEL=DEBUG` を追加 |
| `ログインページに移動しました` → タイムアウト | CDNブロック or ページ構造変更 | Phase 2へ |
| `Cookie復元に失敗` | Cookie破損 | Phase 3へ |
| `認証が無効です` (status 401/403) | Cookie有効期限切れ | Phase 3へ |
| `Cookieを使用してログインが完了しました` | 正常 (問題なし) | - |

### Phase 2: CDN ブロックの確認

**手動ブラウザテスト** (最も確実):

1. 通常のブラウザ (Chrome/Safari) で `https://www.oreilly.com/member/login/` を開く
2. 結果を判定:

| 結果 | 判定 | 対処 |
|------|------|------|
| ログインフォーム表示 | CDNは正常、ChromeDP検出の問題 | Phase 2a へ |
| "Access Denied" | IPレベルのブロック | Phase 2b へ |
| CAPTCHAチャレンジ | Akamai Bot Manager 発動 | Phase 2c へ |

#### Phase 2a: ChromeDP検出の対処

**既知の事実** (2026-02-22 検証済み):
- `headless=new` だけでは不十分
- `DefaultExecAllocatorOptions` のフラグ群が検出される
- 非headlessモードでも `DefaultExecAllocatorOptions` 使用時はブロックされる
- システムChrome + 最小フラグ + stealth JS で一時的に成功するが安定しない

**推奨対処**:
1. Cookie認証を最大限活用 (ChromeDPの使用を最小化)
2. 初回ログインは手動で行い、Cookieを保存
3. Cookie有効期限内は自動認証で運用

#### Phase 2b: IP ブロックへの対処

1. しばらく待つ (Akamaiのブロックは通常一時的)
2. ネットワーク変更 (VPN/テザリング等)
3. Akamai error reference番号を記録 (`Reference #18.xxxxx`)

#### Phase 2c: CAPTCHA チャレンジ

手動でブラウザからログインし、Cookieを取得して保存する。

### Phase 3: Cookie管理

```bash
# Cookie保存場所の確認
ls -la ~/.cache/orm-mcp-go/orm-mcp-go-cookies.json

# Cookie削除 (クリーン再認証)
rm -f ~/.cache/orm-mcp-go/orm-mcp-go-cookies.json

# XDG環境変数が設定されている場合
ls -la ${XDG_CACHE_HOME:-~/.cache}/orm-mcp-go/orm-mcp-go-cookies.json
```

**Cookie有効期限の目安**:
- `orm-jwt`: 数時間〜1日
- `groot_sessionid`: 数日
- `orm-rt` (リフレッシュトークン): 数週間

## リグレッションテスト方針

### Cookie認証 (ログイン済み前提) のテスト

リグレッションテストはログイン済み (Cookie有効) を前提とする。

```bash
# 1. Cookie が存在し、MCP サーバーが起動できること
ls ~/.cache/orm-mcp-go/orm-mcp-go-cookies.json && echo "Cookie exists"

# 2. サーバー起動 (Cookie復元 → HTTP検証 → 成功)
bin/orm-discovery-mcp-go  # "Cookieを使用してログインが完了しました" を確認

# 3. dogfood-verify で MCP ツール動作確認
```

### ブラウザログインのテスト (リスクベース)

以下の状況でのみ実施:
- ChromeDP / lifecycle.go の変更時
- 認証フロー (auth.go) の変更時
- O'Reilly側のログインページ構造変更が疑われる時

```bash
# Cookie削除してクリーン起動テスト
rm -f ~/.cache/orm-mcp-go/orm-mcp-go-cookies.json
ORM_MCP_GO_DEBUG=true ORM_MCP_GO_LOG_LEVEL=DEBUG bin/orm-discovery-mcp-go
```

## lifecycle.go 設定リファレンス

### 現在の推奨設定 (2026-02-22)

```go
opts := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.UserDataDir(chromeDataDir),
    chromedp.Flag("headless", "new"),           // 新headlessモード (Chrome 109+)
    chromedp.Flag("disable-gpu", true),
    chromedp.Flag("enable-automation", false),   // navigator.webdriver 隠蔽
    chromedp.Flag("disable-blink-features", "AutomationControlled"),
    chromedp.Flag("no-sandbox", true),
    chromedp.Flag("disable-web-security", true),
    chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees,VizDisplayCompositor"),
    chromedp.UserAgent("...Chrome/131.0.0.0..."),
)
```

### 検証済みの知見

| 設定 | 結果 | 備考 |
|------|------|------|
| `DefaultExecAllocatorOptions` + `headless=new` | ❌ ブロック | デフォルトフラグが検出される |
| 最小フラグ + システムChrome + `headless=new` | ⚠️ 不安定 | 一時的に成功するが再現不安定 |
| 最小フラグ + 非headless | ❌ ブロック | フラグ問題は headless に限らない |
| Cookie認証 (ChromeDP不使用) | ✅ 安定 | 推奨アプローチ |

## トラブルシューティング早見表

| 問題 | 最速の対処 |
|------|-----------|
| 初回ログインが通らない | 手動ブラウザでログイン → Cookie保存 |
| Cookie期限切れ | Cookie削除 → 手動ログイン → Cookie保存 |
| CDNブロック | 時間を置く or ネットワーク変更 |
| `DefaultExecAllocatorOptions` 問題 | Cookie-first 運用で回避 |
