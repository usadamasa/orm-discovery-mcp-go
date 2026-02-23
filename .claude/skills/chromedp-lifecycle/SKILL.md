# ChromeDP Lifecycle Management

ChromeDP-based authentication implementation guide with "close-after-authentication" pattern.

## Overview

ChromeDP is only required for initial authentication. All subsequent API calls use HTTP client with cookies. The implementation follows a "close-after-authentication" pattern to **avoid issues with URL operations in production environments**. As a secondary benefit, this also reduces memory usage.

## パッケージ構成

ChromeDPライフサイクル管理は `browser/chromedp/` パッケージに切り出されています:

```
browser/
└── chromedp/
    └── lifecycle.go   # Manager構造体とクリーンアップ関数
```

## Manager構造体

`chromedp.Manager` はChromeDPブラウザインスタンスのライフサイクルを管理します:

```go
import cdp "github.com/usadamasa/orm-discovery-mcp-go/browser/chromedp"

// 新しいマネージャーを作成(古いディレクトリを自動クリーンアップ)
manager, err := cdp.NewManager(tmpDir, debug)
if err != nil {
    return nil, err
}

// ブラウザ操作用のコンテキストを取得
ctx := manager.Context()

// ブラウザを閉じてユーザーデータディレクトリを削除
manager.Close()
```

### 主要メソッド

- `NewManager(cacheDir, headless)` - 新しいマネージャーを作成(古いディレクトリを自動クリーンアップ)
  - `headless=true`: ヘッドレスモード (Cookie検証など非対話処理用)
  - `headless=false`: ビジブルモード (手動ログインなど対話処理用)
- `Context()` - ブラウザ操作用のコンテキストを返す
- `Close()` - ブラウザを閉じてユーザーデータディレクトリを削除

## SingletonLock問題の自動解決

サーバーは起動時に以下の処理を自動的に行います:

1. **プロセス固有のユーザーデータディレクトリ**: 各インスタンスは `chrome-user-data-{PID}` 形式のディレクトリを使用
2. **安全な古いディレクトリのクリーンアップ**:
   - プロセスが存在しない → 削除
   - プロセスが存在するが別のプログラム → 削除
   - **orm-discovery-mcp-goが実行中** → スキップ
3. **旧形式ディレクトリ(chrome-user-data)の安全な削除**:
   - SingletonLockがない → 削除
   - SingletonLockがある → スキップ(警告ログ出力)
4. **終了時のクリーンアップ**: Close()で自身のディレクトリを削除

### 複数インスタンス対応

複数のClaude Codeインスタンスから同時に起動しても競合しません:

```bash
# 例: 2つのインスタンスを同時起動
./bin/orm-discovery-mcp-go &   # chrome-user-data-12345
./bin/orm-discovery-mcp-go &   # chrome-user-data-12346
```

## Core Principles

1. **Minimize browser runtime**: Browser only runs during authentication
2. **Production environment safety**: Avoid URL operation issues by closing browser after auth
3. **Debug mode flexibility**: Keep browser alive for troubleshooting when needed
4. **Automatic recovery**: Reauthenticate automatically on cookie expiration
5. **User browser isolation**: Process-specific UserDataDir prevents interference
6. **Automatic cleanup**: Old Chrome data directories are cleaned up on startup

## Implementation Patterns

### 1. Authentication-Only Browser Usage

**Pattern**: Start ChromeDP via Manager → Authenticate → Close immediately (unless debug mode)

```go
// NewBrowserClient() - auth.go
func NewBrowserClient(cookieManager cookie.Manager, debug bool, stateDir string) (*BrowserClient, error) {
    client := &BrowserClient{
        httpClient: &http.Client{...},
        stateDir:   stateDir,
        debug:      debug,
    }

    // 1. Cookie復元 → HTTP検証
    if cookieManager.CookieFileExists() {
        cookieManager.LoadCookies()
        client.cookieManager = cookieManager
        if client.validateAuthenticationViaHTTP() {
            return client, nil // Cookie有効: ブラウザ不要
        }
    }

    // 2. Cookie無効: ビジブルブラウザで手動ログイン
    client.cookieManager = cookieManager
    cookies, err := client.loginWithVisibleBrowser() // login.go
    if err != nil {
        return nil, fmt.Errorf("failed to login: %w", err)
    }

    // 3. Cookie保存
    cookieManager.SaveCookiesFromData(cookies)
    return client, nil
}

// loginWithVisibleBrowser() - login.go
// ビジブルChrome (headless=false) を起動し、ユーザーが learning.oreilly.com に到達するまで待機
func (bc *BrowserClient) loginWithVisibleBrowser() ([]*http.Cookie, error) {
    manager, err := cdp.NewManager(bc.stateDir, false) // headless=false
    if err != nil {
        return nil, err
    }
    defer manager.Close()

    loginCtx, cancel := context.WithTimeout(manager.Context(), VisibleLoginTimeout) // 5分
    defer cancel()

    chromedp.Run(loginCtx, chromedp.Navigate("https://www.oreilly.com/member/login/"))

    // 2秒間隔でポーリング
    for {
        select {
        case <-loginCtx.Done():
            return nil, fmt.Errorf("タイムアウト")
        case <-ticker.C:
            var url string
            chromedp.Run(loginCtx, chromedp.Location(&url))
            if strings.Contains(url, "learning.oreilly.com") {
                // Cookie取得して返す
            }
        }
    }
}
```

**Benefits**:
- **Avoids SingletonLock errors** by using process-specific directories
- **Avoids URL operation issues** in production environments
- Browser process only runs during authentication
- HTTP API calls work without browser
- 100-300MB memory reduction as secondary benefit
- **Automatic cleanup** of old Chrome data directories

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

**Pattern**: Detect 401/403 errors → Launch visible browser → Re-login → Close

```go
// Reauthenticate() - auth.go
func (bc *BrowserClient) Reauthenticate() error {
    slog.Info("Cookie有効期限切れ検出: 再認証を開始します")

    // 1. ビジブルブラウザで再ログイン
    cookies, err := bc.loginWithVisibleBrowser() // login.go
    if err != nil {
        return fmt.Errorf("再認証に失敗しました: %w", err)
    }

    // 2. Cookie保存
    bc.cookieManager.SaveCookiesFromData(cookies)
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
    // Automatic reauthentication (visible browser)
    if reauthErr := s.browserClient.Reauthenticate(); reauthErr != nil {
        return mcp.NewToolResultError(fmt.Sprintf("再認証に失敗しました: %v", reauthErr)), nil
    }
    // Retry
    results, err = s.browserClient.SearchContent(requestParams.Query, options)
}
```

## State Management

### Why No isClosed Flag?

**Answer**: Not needed - Manager handles cleanup internally

```go
// Close() - Manager handles all cleanup
func (bc *BrowserClient) Close() {
    if bc.chromedpManager != nil {
        bc.chromedpManager.Close()
        return
    }
    // Fallback for backward compatibility
    if bc.ctxCancel != nil {
        bc.ctxCancel()
    }
    if bc.allocCancel != nil {
        bc.allocCancel()
    }
}
```

**Rationale**:
- Manager handles browser close and directory cleanup atomically
- Go's nil-safe checks prevent double-close issues
- Multiple Close() calls are harmless
- Simpler implementation without additional state

## Chrome Isolation (User Browser Protection)

### Process-Specific UserDataDir Setting

**Pattern**: Each process uses isolated UserDataDir to prevent SingletonLock conflicts

```go
// Manager creates process-specific directory
chromeDataDir := filepath.Join(tmpDir, fmt.Sprintf("chrome-user-data-%d", os.Getpid()))
chromedp.UserDataDir(chromeDataDir)
```

**Benefits**:
- **No SingletonLock conflicts**: Each process has its own directory
- **Multiple instance support**: Multiple MCP servers can run simultaneously
- **Automatic cleanup**: Old directories are cleaned up on startup
- **Explicit isolation**: No accidental access to user's Chrome profile
- **Unified management**: tmpDir controls both cookies and Chrome data

**Old vs New**:
- **Old** (fixed): `filepath.Join(tmpDir, "chrome-user-data")` - causes SingletonLock conflicts
- **New** (dynamic): `filepath.Join(tmpDir, "chrome-user-data-{PID}")` - no conflicts

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

- [ ] Use `cdp.NewManager()` for ChromeDP lifecycle management
- [ ] Store Manager in `chromedpManager` field
- [ ] Close browser immediately after authentication (non-debug mode)
- [ ] Keep browser alive in debug mode for screenshots
- [ ] Implement 401/403 error detection
- [ ] Add automatic reauthentication with new Manager
- [ ] Use Manager.Close() for cleanup
- [ ] Add defer browserClient.Close() in main.go only
- [ ] Test both debug and non-debug modes
- [ ] Test multiple instances running simultaneously
- [ ] Verify old Chrome data directories are cleaned up

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

### Verify Multiple Instance Support
```bash
# 1. Build the project
task build

# 2. Start multiple instances simultaneously
./bin/orm-discovery-mcp-go &
PID1=$!
./bin/orm-discovery-mcp-go &
PID2=$!

# 3. Verify both directories exist with different PIDs
ls /var/tmp/ | grep chrome-user-data
# Expected: chrome-user-data-{PID1}, chrome-user-data-{PID2}

# 4. Verify cleanup after termination
kill $PID1 $PID2
ls /var/tmp/ | grep chrome-user-data
# Expected: directories should be cleaned up
```

### Verify Chrome Isolation
```bash
# Check UserDataDir location (process-specific)
ls -la /var/tmp/ | grep chrome-user-data
# Expected: chrome-user-data-{PID} directories for running processes

# Verify no interference with user's Chrome
ls -la ~/.config/google-chrome/Default/  # Should be unchanged
```

### Verify Defer Cleanup
```bash
# Verify defer cleanup on process termination
# Watch logs for cleanup messages when process exits
```

## Remote Chrome Connection Pattern (--login)

`--login` コマンドのように既存Chromeに接続して手動操作を監視する場合に使うパターン。

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
cmd := exec.Command(
    chromePath,
    "--remote-debugging-port=9222",
    "--user-data-dir="+setupDataDir,
    "--no-first-run",            // 初回セットアップウィザードを抑制
    "--no-default-browser-check", // デフォルトブラウザチェックを抑制
)

// 2. CDP接続待機 (http://localhost:9222/json/version をポーリング)
wsURL, _ := waitForCDP("9222")

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

1. **Using defer in NewBrowserClient()**: Don't use defer for Close() in NewBrowserClient(). Use explicit calls instead for immediate cleanup.

2. **Forgetting UserDataDir**: Always set explicit UserDataDir to prevent interference with user's Chrome.

3. **Not handling reauthentication**: Implement automatic reauthentication for 401/403 errors.

4. **Missing debug mode check**: Always check debug mode before closing browser to enable troubleshooting.

5. **Passing URL to Chrome args when using chromedp to monitor**: Chrome opens a tab at that URL, but `chromedp.NewContext` creates a DIFFERENT tab at `about:blank`. The tabs are separate. Navigate the chromedp tab instead.

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

**新しい実装では自動解決**: プロセス固有のディレクトリ(`chrome-user-data-{PID}`)を使用するため、SingletonLock 問題は通常発生しません。サーバー起動時に古いディレクトリは自動的にクリーンアップされます。

**旧形式ディレクトリが残っている場合**:

旧形式の `chrome-user-data` ディレクトリにSingletonLockがある場合、サーバーは以下の警告を出力してスキップします:

```
旧形式のChromeデータディレクトリにSingletonLockが存在するため削除をスキップ
path=/var/tmp/chrome-user-data hint=手動で削除してください: rm -rf /var/tmp/chrome-user-data
```

**手動削除が必要な場合**:
```bash
# 旧形式ディレクトリの確認
ls -la /var/tmp/chrome-user-data

# Chrome関連ファイルを全て削除
rm -rf /var/tmp/chrome-user-data

# プロセス固有ディレクトリの削除(通常は不要 - 終了時に自動削除)
rm -rf /var/tmp/chrome-user-data-*
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
