# ChromeDP Lifecycle Management

exec.Command + NewRemoteAllocator ベースの認証実装ガイド。"close-after-authentication" パターンを採用。

## Overview

ChromeDP is only required for initial authentication. All subsequent API calls use HTTP client with cookies. The implementation follows a "close-after-authentication" pattern. Chrome は `exec.Command` でネイティブ起動し、`chromedp.NewRemoteAllocator` で CDP 接続する。`chromedp.NewExecAllocator` は Akamai にボットとして検知されるため使用しない。メモリ使用量の削減は副次的な利点。

## パッケージ構成

ChromeDP ライフサイクル管理は `browser/` パッケージ内に統合されています:

```
browser/
├── login.go   # RunVisibleLogin / runVisibleLogin (Chrome起動とログイン待機)
├── auth.go    # NewBrowserClient / Reauthenticate / Close
└── types.go   # BrowserClient構造体、タイムアウト定数
```

## Core Principles

1. **Minimize browser runtime**: Browser only runs during authentication
2. **Production environment safety**: Avoid URL operation issues by closing browser after auth
3. **Automatic recovery**: Reauthenticate automatically on cookie expiration
4. **User browser isolation**: Temporary UserDataDir prevents interference
5. **Process exit detection**: Chrome closure is detected immediately via goroutine

## Implementation Patterns

### 1. Authentication-Only Browser Usage

**Pattern**: exec.Command で Chrome を起動 → NewRemoteAllocator で CDP 接続 → 認証 → プロセス Kill

```go
// NewBrowserClient() - auth.go
func NewBrowserClient(cookieManager cookie.Manager, debug bool, stateDir string) (*BrowserClient, error) {
    client := &BrowserClient{
        httpClient: &http.Client{...},
        userAgent:  "Mozilla/5.0 ...",
        stateDir:   stateDir,
        debug:      debug,
    }

    // 1. Cookie復元 → HTTP検証
    if cookieManager.CookieFileExists() {
        cookieManager.LoadCookies()
        client.cookieManager = cookieManager
        if client.validateAuthenticationViaHTTP() == nil {
            return client, nil // Cookie有効: ブラウザ不要
        }
    }

    // 2. Cookie無効: ビジブルブラウザで手動ログイン
    client.cookieManager = cookieManager
    if err := RunVisibleLogin(filepath.Join(stateDir, "chrome-setup"), cookieManager); err != nil {
        return nil, fmt.Errorf("failed to login: %w", err)
    }
    return client, nil
}
```

### 2. Visible Login Flow (login.go)

**Pattern**: exec.Command + NewRemoteAllocator + processDone チャネル

```go
// runVisibleLogin() - login.go
func runVisibleLogin(tempDir string) ([]*http.Cookie, error) {
    chromePath, _ := FindSystemChrome() // macOS / Linux のみ

    // 1. exec.Command で Chrome をネイティブ起動
    cmd := exec.Command(chromePath,
        "--remote-debugging-port="+CDPDebugPort,
        "--user-data-dir="+tempDir,
        "--no-first-run",
        "--no-default-browser-check",
    )
    cmd.Start()

    // 2. goroutine で Chrome プロセスの終了を監視
    processDone := make(chan error, 1)
    go func() { processDone <- cmd.Wait() }()
    processExited := false

    defer func() {
        if !processExited {
            cmd.Process.Kill()
            <-processDone // ゾンビプロセス回収
        }
        os.RemoveAll(tempDir)
    }()

    // 3. CDP 接続待機 → NewRemoteAllocator
    wsURL, _ := WaitForCDPWithTimeout(CDPDebugPort, CDPWaitTimeout)
    allocCtx, _ := chromedp.NewRemoteAllocator(context.Background(), wsURL)
    chromeCtx, _ := chromedp.NewContext(allocCtx)
    loginCtx, _ := context.WithTimeout(chromeCtx, VisibleLoginTimeout) // 5分

    // 4. ログインページにナビゲート
    chromedp.Run(loginCtx, chromedp.Navigate("https://www.oreilly.com/member/login/"))

    // 5. ポーリングループ
    for {
        select {
        case <-loginCtx.Done():
            return nil, fmt.Errorf("タイムアウト")
        case waitErr := <-processDone:
            processExited = true
            return nil, fmt.Errorf("chromeが予期せず終了: %w", waitErr)
        case <-ticker.C:
            chromedp.Run(loginCtx, chromedp.Location(&url))
            if strings.Contains(url, "learning.oreilly.com") {
                // Cookie取得して返す
            }
        }
    }
}
```

**Benefits**:
- **exec.Command**: Akamai のボット検知を回避 (chromedp.NewExecAllocator は検知される)
- **processDone channel**: Chrome を閉じると即座にエラーを返す (5分待たない)
- **processExited flag**: defer での二重 Wait を回避

### 3. Automatic Reauthentication

**Pattern**: Detect 401/403 errors → Launch visible browser → Re-login

```go
// Reauthenticate() - auth.go
func (bc *BrowserClient) Reauthenticate() error {
    slog.Info("Cookie有効期限切れ検出: ビジブルブラウザで再認証を開始します")
    if err := RunVisibleLogin(filepath.Join(bc.stateDir, "chrome-setup"), bc.cookieManager); err != nil {
        return fmt.Errorf("再認証に失敗しました: %w", err)
    }
    return nil
}
```

### 4. HTTP API Error Handling

**Pattern**: Detect authentication errors → Trigger reauthentication → Retry

```go
// server.go - SearchContentHandler
results, err := s.browserClient.SearchContent(requestParams.Query, options)
if err != nil && isAuthError(err) {
    if reauthErr := s.browserClient.Reauthenticate(); reauthErr != nil {
        return mcp.NewToolResultError(fmt.Sprintf("再認証に失敗しました: %v", reauthErr)), nil
    }
    // Retry
    results, err = s.browserClient.SearchContent(requestParams.Query, options)
}
```

## State Management

### Why No isClosed Flag?

**Answer**: Not needed - BrowserClient.Close() is a no-op

```go
// Close() - auth.go
func (bc *BrowserClient) Close() {
    // httpClient と cookieManager はクリーンアップ不要
}
```

**Rationale**:
- Chrome プロセスは `runVisibleLogin` の defer で自動クリーンアップ
- `BrowserClient` 自体はブラウザプロセスを保持しない
- HTTP クライアントと Cookie マネージャーは GC で回収される

## Chrome Isolation (User Browser Protection)

### Temporary UserDataDir

**Pattern**: 一時ディレクトリで Chrome を起動し、終了後に削除

```go
// login.go - RunVisibleLogin に渡す tempDir
RunVisibleLogin(filepath.Join(stateDir, "chrome-setup"), cookieManager)

// runVisibleLogin 内
cmd := exec.Command(chromePath,
    "--user-data-dir="+tempDir,  // 一時プロファイル
    // ...
)
defer os.RemoveAll(tempDir) // 終了後に削除
```

**Benefits**:
- **Explicit isolation**: ユーザーの Chrome プロファイルに影響しない
- **Automatic cleanup**: defer で一時ディレクトリを必ず削除
- **No SingletonLock**: 毎回新しいディレクトリを使用

## Defer Pattern for Cleanup

### main.go - Safety Net

```go
defer browserClient.Close() // 現在は no-op だが将来の安全弁
```

**Purpose**:
- Acts as safety net for future changes
- Ensures cleanup contract is maintained

## Implementation Checklist

When implementing ChromeDP-based features:

- [ ] Use `exec.Command` + `NewRemoteAllocator` for Chrome lifecycle
- [ ] Add `processDone` channel for Chrome process exit detection
- [ ] Use `processExited` flag to avoid double-Wait in defer
- [ ] Close browser and clean up temp dir in defer
- [ ] Implement 401/403 error detection
- [ ] Add automatic reauthentication via `RunVisibleLogin`
- [ ] Test browser closure during login (should return error immediately)
- [ ] Verify temp directory cleanup after login

## Memory Impact

Note: Memory reduction is a secondary benefit. The primary reason for closing the browser after authentication is to avoid URL operation issues in production environments.

| Mode | Browser State | Memory Usage | Use Case |
|------|---------------|--------------|----------|
| **Normal (Production)** | Closed after auth | ~10-30MB | Production deployment |
| **Reauthentication** | Temporary restart | Brief spike (~100-300MB) | Cookie expiration handling |

## Timeout Constants (types.go)

```go
const (
    AuthValidationTimeout = 15 * time.Second
    APIOperationTimeout   = 30 * time.Second
    VisibleLoginTimeout   = 5 * time.Minute  // 手動ログイン待機
)
```

## CDP Constants (login.go)

```go
const (
    CDPWaitTimeout    = 30 * time.Second  // CDP 接続待機
    cdpPollInterval   = 1 * time.Second
    cdpRequestTimeout = 3 * time.Second
)
// CDPデバッグポートは findAvailablePort() で動的に割り当て
```

## Remote Chrome Connection Pattern

`RunVisibleLogin` のように既存 Chrome に接続して手動操作を監視する場合に使うパターン。

### 重要な罠: chromedp.NewContext は新しいタブを作る

```
Chrome起動 → タブ1: oreilly.com/member/login/  ← ユーザーが操作するタブ
chromedp   → タブ2: about:blank                  ← NewContext が作った新しいタブ
```

`chromedp.NewContext(allocCtx)` は既存タブに「接続」するのではなく、**新しい空のタブを作成**する。既存タブのURL変化は監視できない。

**症状**: `chromedp.Location` が永遠に `about:blank` を返す

### 正しいパターン: 新しいタブをナビゲートして監視する

```go
// 1. Chrome を起動 (URLなし。chromedpでナビゲートするから不要)
cmd := exec.Command(chromePath,
    "--remote-debugging-port=9222",
    "--user-data-dir="+tempDir,
    "--no-first-run",
    "--no-default-browser-check",
)

// 2. CDP接続待機 (http://127.0.0.1:9222/json/version をポーリング)
wsURL, _ := WaitForCDPWithTimeout("9222", CDPWaitTimeout)

// 3. ブラウザレベルで接続
allocCtx, _ := chromedp.NewRemoteAllocator(context.Background(), wsURL)

// 4. 新しいタブを作成 (about:blank から始まる)
ctx, _ := chromedp.NewContext(allocCtx)

// 5. このタブをログインページへナビゲート ← ここが重要！
chromedp.Run(ctx, chromedp.Navigate(loginURL))

// 6. このタブのURL変化を監視 → 正しく learning.oreilly.com を検出できる
chromedp.Run(ctx, chromedp.Location(&currentURL))
```

### NG パターン: ChromeにURLを渡してchromedpで監視

```go
// ❌ Chrome起動時にURLを渡す → タブ1がlogin.oreilly.com
cmd := exec.Command(chromePath, "--remote-debugging-port=9222", loginURL)

// chromedp.NewContext はタブ2(about:blank)を作る
ctx, _ := chromedp.NewContext(allocCtx)

// タブ2のURLを監視 → 永遠にabout:blank！
chromedp.Run(ctx, chromedp.Location(&url)) // about:blank ...
```

### CDP エンドポイントの使い分け

| エンドポイント | 返す情報 | 用途 |
|--------------|---------|------|
| `/json/version` | ブラウザのwebSocketDebuggerUrl | `NewRemoteAllocator` に渡すWSURL取得 |
| `/json` または `/json/list` | 全タブ(target)の一覧 | 特定タブへの接続 (`chromedp.WithTargetID`) |

## Common Pitfalls

1. **Using chromedp.NewExecAllocator**: Akamai がボットとして検知する。exec.Command + NewRemoteAllocator を使うこと。

2. **Forgetting processDone channel**: Chrome がクラッシュやユーザーにより閉じられた場合、タイムアウトまで待ち続ける。processDone channel で即座に検知すること。

3. **Double Wait on processDone**: select で processDone を消費した後、defer で再度 `<-processDone` するとデッドロックする。`processExited` フラグで回避すること。

4. **Passing URL to Chrome args when using chromedp to monitor**: Chrome がそのURLでタブを開くが、`chromedp.NewContext` は別のタブ (about:blank) を作る。chromedp のタブをナビゲートすること。

5. **Not handling reauthentication**: 401/403 エラーを検知して `Reauthenticate()` で自動再認証すること。

## Related Files

- `browser/login.go`: RunVisibleLogin / runVisibleLogin (Chrome 起動とログイン待機)
- `browser/auth.go`: NewBrowserClient / Reauthenticate / Close
- `browser/types.go`: BrowserClient 構造体、タイムアウト定数
- `browser/cookie/cookie.go`: Cookie 管理
- `main.go`: defer cleanup pattern
- `server.go`: Error handling and reauthentication trigger

## References

- ChromeDP documentation: https://github.com/chromedp/chromedp
- Original implementation guide: `browser/CLAUDE.md`

## Troubleshooting

### タイムアウト発生時の対処

タイムアウトが発生した場合、ブラウザプロセスが残っている可能性がある。

**タイムアウト値の確認** (browser/types.go):
```go
const (
    AuthValidationTimeout = 15 * time.Second  // 認証検証
    APIOperationTimeout   = 30 * time.Second  // API呼び出し
    VisibleLoginTimeout   = 5 * time.Minute   // 手動ログイン待機
)
```

### Chromeプロセスの強制終了

**プロセス確認**:
```bash
# Chromeプロセスを確認
ps aux | grep -E "(chrome|chromium)" | grep -v grep

# chromedpが起動したChromeを特定 (user-data-dirで判別)
ps aux | grep chrome-setup
```

**プロセス強制終了**:
```bash
# 特定のプロセスを終了
kill -9 <PID>

# chromedp関連のChromeを一括終了
pkill -f "chrome.*chrome-setup"
```

### よくある問題と解決策

| 問題 | 原因 | 解決策 |
|------|------|--------|
| ブラウザが起動しない | Chrome未インストール | Chrome/Chromiumをインストール |
| タイムアウトが頻発 | ネットワーク遅延/サーバー負荷 | タイムアウト値を増やす or リトライ |
| Cookie復元失敗 | Cookieファイル破損 | Cookieファイルを削除して再認証 |
| Chrome閉じてもエラーが出ない | processDone channel の実装漏れ | processDone channel を確認 |

### Cookieファイルのリセット

認証問題が解決しない場合、Cookieファイルをリセット。

```bash
# Cookieファイルを削除
rm -f /var/tmp/orm-mcp-go-cookies.json

# サーバーを再起動して再認証
./bin/orm-discovery-mcp-go
```
