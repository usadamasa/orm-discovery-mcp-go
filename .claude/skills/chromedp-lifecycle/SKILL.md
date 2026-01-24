# ChromeDP Lifecycle Management

ChromeDP-based authentication implementation guide with "close-after-authentication" pattern.

## Overview

ChromeDP is only required for initial authentication. All subsequent API calls use HTTP client with cookies. The implementation follows a "close-after-authentication" pattern to **avoid issues with URL operations in production environments**. As a secondary benefit, this also reduces memory usage.

## Core Principles

1. **Minimize browser runtime**: Browser only runs during authentication
2. **Production environment safety**: Avoid URL operation issues by closing browser after auth
3. **Debug mode flexibility**: Keep browser alive for troubleshooting when needed
4. **Automatic recovery**: Reauthenticate automatically on cookie expiration
5. **User browser isolation**: Explicit UserDataDir prevents interference

## Implementation Patterns

### 1. Authentication-Only Browser Usage

**Pattern**: Start ChromeDP → Authenticate → Close immediately (unless debug mode)

```go
// NewBrowserClient() - auth.go
func NewBrowserClient(userID, password string, cookieManager cookie.Manager, debug bool, tmpDir string) (*BrowserClient, error) {
    // 1. Start ChromeDP with explicit UserDataDir
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserDataDir(filepath.Join(tmpDir, "chrome-user-data")), // Explicit isolation
        chromedp.Flag("headless", true),
        // ... other flags
    )

    // 2. Authenticate (either via cookies or password login)
    // ... authentication logic ...

    // 3. Close immediately in non-debug mode
    if !debug {
        slog.Info("非デバッグモード: ブラウザコンテキストをクローズします")
        client.Close()
    }

    return client, nil
}
```

**Benefits**:
- **Avoids URL operation issues** in production environments (primary reason)
- Browser process only runs during authentication
- HTTP API calls work without browser
- 100-300MB memory reduction as secondary benefit

### 2. Debug Mode Persistence

**Pattern**: Keep browser alive in debug mode for screenshot functionality

```go
// Debug mode check before closing
if !debug {
    client.Close()
}
// In debug mode, browser stays alive for debugScreenshot()
```

**Use Case**:
- Development and troubleshooting
- Screenshot capture during authentication flow
- Visual verification of browser state

### 3. Automatic Reauthentication

**Pattern**: Detect 401/403 errors → Restart browser → Re-login → Close

```go
// ReauthenticateIfNeeded() - auth.go
func (bc *BrowserClient) ReauthenticateIfNeeded(userID, password string) error {
    slog.Info("Cookie有効期限切れ検出: 再認証を開始します")

    // 1. Temporarily restart browser
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.UserDataDir(filepath.Join(bc.tmpDir, "chrome-user-data")),
        // ... flags ...
    )
    allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer allocCancel()

    ctx, ctxCancel := chromedp.NewContext(allocCtx)
    defer ctxCancel()

    // 2. Update browser context temporarily
    bc.ctx = ctx
    bc.ctxCancel = ctxCancel
    bc.allocCancel = allocCancel

    // 3. Re-login
    if err := bc.login(userID, password); err != nil {
        return fmt.Errorf("再認証に失敗しました: %w", err)
    }

    // 4. Save cookies
    bc.cookieManager.SaveCookies(&ctx)
    bc.syncCookiesFromBrowser()

    // 5. Close immediately (non-debug mode)
    if !bc.debug {
        bc.Close()
    }

    return nil
}
```

### 4. HTTP API Error Handling

**Pattern**: Detect authentication errors → Trigger reauthentication → Retry

```go
// GetContentFromURL() - auth.go
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return "", fmt.Errorf("authentication error: status %d (cookies may have expired)", resp.StatusCode)
}

// server.go - SearchContentHandler
results, err := s.browserClient.SearchContent(requestParams.Query, options)
if err != nil && isAuthError(err) {
    // Automatic reauthentication
    if reauthErr := s.browserClient.ReauthenticateIfNeeded(s.config.OReillyUserID, s.config.OReillyPassword); reauthErr != nil {
        return mcp.NewToolResultError(fmt.Sprintf("再認証に失敗しました: %v", reauthErr)), nil
    }
    // Retry
    results, err = s.browserClient.SearchContent(requestParams.Query, options)
}
```

## State Management

### Why No isClosed Flag?

**Answer**: Not needed - nil-safe checks are sufficient

```go
// Close() - No state flag required
func (bc *BrowserClient) Close() {
    if bc.ctxCancel != nil {
        bc.ctxCancel()
    }
    if bc.allocCancel != nil {
        bc.allocCancel()
    }
}
```

**Rationale**:
- Go's nil-safe checks prevent double-close issues
- Multiple Close() calls are harmless
- Simpler implementation without additional state

## Chrome Isolation (User Browser Protection)

### Explicit UserDataDir Setting

**Pattern**: Always specify isolated UserDataDir to prevent interference

```go
chromedp.UserDataDir(filepath.Join(tmpDir, "chrome-user-data"))
```

**Benefits**:
- **Explicit isolation**: No accidental access to user's Chrome profile
- **Unified management**: tmpDir controls both cookies and Chrome data
- **Testability**: Easy to specify different directories for testing
- **Visibility**: Clear in logs where Chrome data is stored

**Default vs Explicit**:
- **Default** (implicit): ChromeDP creates `/tmp/chromedp-*` automatically
- **Explicit** (recommended): `filepath.Join(tmpDir, "chrome-user-data")` for clarity

## Defer Pattern for Cleanup

### main.go - Safety Net

```go
defer browserClient.Close() // プロセス終了時にブラウザをクリーンアップ
```

**Purpose**:
- Acts as safety net if Close() wasn't called in NewBrowserClient()
- Ensures cleanup on process termination
- Handles debug mode where browser stays alive

**Key Point**: Only use defer in main.go, NOT in NewBrowserClient()
- NewBrowserClient() uses explicit Close() calls for immediate cleanup
- defer in main.go provides final cleanup guarantee

## Implementation Checklist

When implementing ChromeDP-based features:

- [ ] Use explicit `UserDataDir` in ExecAllocatorOptions
- [ ] Close browser immediately after authentication (non-debug mode)
- [ ] Keep browser alive in debug mode for screenshots
- [ ] Implement 401/403 error detection
- [ ] Add automatic reauthentication with browser restart
- [ ] Use nil-safe checks instead of state flags
- [ ] Add defer browserClient.Close() in main.go only
- [ ] Test both debug and non-debug modes
- [ ] Verify browser closes after authentication in production mode

## Memory Impact

Note: Memory reduction is a secondary benefit. The primary reason for closing the browser after authentication is to avoid URL operation issues in production environments.

| Mode | Browser State | Memory Usage | Use Case |
|------|---------------|--------------|----------|
| **Normal (Production)** | Closed after auth | ~10-30MB | Production deployment |
| **Debug** | Always running | ~100-300MB | Development & troubleshooting |
| **Reauthentication** | Temporary restart | Brief spike (~100-300MB) | Cookie expiration handling |

## Testing

### Verify Browser Lifecycle
```bash
# 1. Non-debug mode - browser should close after auth
task build
./bin/orm-discovery-mcp-go
ps aux | grep chrome  # Should not find chrome process after startup

# 2. Debug mode - browser should stay alive
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go
ps aux | grep chrome  # Should find running chrome process

# 3. Memory comparison
ps aux | grep orm-discovery-mcp-go  # Compare RSS with/without debug mode
```

### Verify Chrome Isolation
```bash
# Check UserDataDir location
ls -la /var/tmp/chrome-user-data  # ChromeDP data isolated here

# Verify no interference with user's Chrome
ls -la ~/.config/google-chrome/Default/  # Should be unchanged
```

### Verify Defer Cleanup
```bash
# Verify defer cleanup on process termination
# Watch logs for cleanup messages when process exits
```

## Common Pitfalls

1. **Using defer in NewBrowserClient()**: Don't use defer for Close() in NewBrowserClient(). Use explicit calls instead for immediate cleanup.

2. **Forgetting UserDataDir**: Always set explicit UserDataDir to prevent interference with user's Chrome.

3. **Not handling reauthentication**: Implement automatic reauthentication for 401/403 errors.

4. **Missing debug mode check**: Always check debug mode before closing browser to enable troubleshooting.

## Related Files

- `browser/auth.go`: Main implementation
- `browser/types.go`: BrowserClient structure
- `browser/cookie/cookie.go`: Cookie management
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
    ChromeDPExecAllocatorTimeout = 45 * time.Second  // ブラウザ起動全体
    LoginTimeout                 = 30 * time.Second  // ログイン処理
    AuthValidationTimeout        = 15 * time.Second  // 認証検証
    CookieOperationTimeout       = 10 * time.Second  // Cookie操作
    APIOperationTimeout          = 30 * time.Second  // API呼び出し
)
```

**タイムアウト時のログ例**:
```
ログインタイムアウト: ブラウザをクローズします timeout=30s
再認証タイムアウト: ブラウザをクローズします timeout=45s
認証検証がタイムアウトしました timeout=15s
```

### Chromeプロセスの強制終了

**プロセス確認**:
```bash
# Chromeプロセスを確認
ps aux | grep -E "(chrome|chromium)" | grep -v grep

# chromedpが起動したChromeを特定 (user-data-dirで判別)
ps aux | grep chrome-user-data
```

**プロセス強制終了**:
```bash
# 特定のプロセスを終了
kill -9 <PID>

# chromedp関連のChromeを一括終了 (注意: 他のChromeも終了する可能性)
pkill -f "chrome.*chrome-user-data"

# macOSでChrome全体を終了
killall "Google Chrome"
```

### SingletonLockファイルの削除

Chromeが異常終了した場合、SingletonLockファイルが残ることがある。

**ロックファイルの場所**:
```bash
# デフォルトの場所
ls -la /var/tmp/chrome-user-data/SingletonLock

# または環境変数で指定したtmpDir
ls -la $ORM_MCP_GO_TMP_DIR/chrome-user-data/SingletonLock
```

**ロックファイルの削除**:
```bash
# ロックファイルを削除
rm -f /var/tmp/chrome-user-data/SingletonLock

# Chrome関連ファイルを全て削除 (再認証が必要になる)
rm -rf /var/tmp/chrome-user-data
```

### デバッグモードでの強制終了

デバッグモードではブラウザが起動したままになる。

**安全な終了方法**:
```bash
# MCPサーバーを終了 (defer cleanup が実行される)
# Ctrl+C または SIGTERM

# プロセスにSIGTERMを送信
kill <PID>
```

**強制終了** (defer cleanup がスキップされる):
```bash
# SIGKILL (最終手段)
kill -9 <PID>

# この場合、手動でChromeプロセスを終了する必要がある
pkill -f "chrome.*chrome-user-data"
```

### よくある問題と解決策

| 問題 | 原因 | 解決策 |
|------|------|--------|
| ブラウザが起動しない | Chrome未インストール | Chrome/Chromiumをインストール |
| タイムアウトが頻発 | ネットワーク遅延/サーバー負荷 | タイムアウト値を増やす or リトライ |
| ログイン失敗 | 認証情報の誤り | OREILLY_USER_ID/PASSWORDを確認 |
| Cookie復元失敗 | Cookieファイル破損 | Cookieファイルを削除して再認証 |
| プロセスが残る | Close()未呼び出し | 手動でプロセス終了 + コード修正 |
| メモリリーク | ブラウザ未クローズ | debug=false でブラウザをクローズ |

### SingletonLock問題の実際のエラー例

```
chrome failed to start:
[76355:14201041:0124/121754.454753:ERROR:chrome/browser/process_singleton_posix.cc:345] Failed to create /private/var/tmp/chrome-user-data/SingletonLock: File exists (17)
[76355:14201041:0124/121754.455879:ERROR:chrome/app/chrome_main_delegate.cc:510] Failed to create a ProcessSingleton for your profile directory. This means that running multiple instances would start multiple browser processes rather than opening a new window in the existing process. Aborting now to avoid profile corruption.
```

**原因**: 前回のセッションでChromeが正常終了しなかった場合に発生

**解決手順**:
```bash
# 1. SingletonLockを削除
rm -f /private/var/tmp/chrome-user-data/SingletonLock

# 2. 残留Chromeプロセスを確認
ps aux | grep -E "[c]hrome.*chrome-user-data"

# 3. 残留プロセスがあれば終了
pkill -f "chrome.*chrome-user-data"
```

### macOSでの起動確認方法

macOSには`timeout`コマンドがないため、代替方法を使用:

```bash
# 方法1: バックグラウンド実行 + sleep + kill
(./bin/orm-discovery-mcp-go 2>&1 & pid=$!; sleep 30; kill $pid 2>/dev/null) || true

# 方法2: gtimeoutをインストール (Homebrew)
brew install coreutils
gtimeout 30 ./bin/orm-discovery-mcp-go

# 方法3: perlを使用
perl -e 'alarm 30; exec @ARGV' ./bin/orm-discovery-mcp-go
```

### 正常起動時のログ例

```
time=2026-01-24T12:17:54.378+09:00 level=INFO msg=ログシステムを初期化しました log_level=INFO debug_mode=false
time=2026-01-24T12:17:54.378+09:00 level=INFO msg=設定を読み込みました
time=2026-01-24T12:17:54.378+09:00 level=INFO msg=ブラウザクライアントを使用してO'Reillyにログインします...
time=2026-01-24T12:17:54.378+09:00 level=INFO msg=O'Reillyへのログインを開始します user_id=xxx@acm.org
time=2026-01-24T12:XX:XX.XXX+09:00 level=INFO msg=ブラウザクライアントの初期化が完了しました
time=2026-01-24T12:XX:XX.XXX+09:00 level=INFO msg=MCPサーバーを標準入出力で起動します
```

### タイムアウト発生時のログ例

```
time=2026-01-24T12:19:19.972+09:00 level=ERROR msg="ログインタイムアウト: ブラウザをクローズします" timeout=30s
time=2026-01-24T12:19:19.981+09:00 level=ERROR msg=ブラウザクライアントの初期化に失敗しました error="failed to login: ログイン処理でエラーが発生しました: context deadline exceeded"
```

**ポイント**: タイムアウト後にブラウザが正しくクローズされているか確認
```bash
ps aux | grep -E "[c]hrome" | head -10
# chromedp起動のプロセスが残っていないことを確認
```

### ログ確認方法

```bash
# デバッグモードで詳細ログを出力
ORM_MCP_GO_DEBUG=true ./bin/orm-discovery-mcp-go

# ログレベルを調整 (slogを使用)
# コード内でslog.SetLogLevel()を使用可能
```

### Cookieファイルのリセット

認証問題が解決しない場合、Cookieファイルをリセット。

```bash
# Cookieファイルを削除
rm -f /var/tmp/orm-mcp-go-cookies.json

# Chrome User Dataも削除 (完全リセット)
rm -rf /var/tmp/chrome-user-data

# サーバーを再起動して再認証
./bin/orm-discovery-mcp-go
```
