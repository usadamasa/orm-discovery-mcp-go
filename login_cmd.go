package main

import (
	"fmt"
	"io"
	"os"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// runLogin は手動ログインからCookieを保存するフローを実行します
// OREILLY_USER_ID / OREILLY_PASSWORD は不要（手動ログインのため）
// CLI から呼ばれるエントリポイント (stdout に出力)
func runLogin() error {
	return runLoginWithOutput(os.Stdout)
}

// runLoginWithOutput は出力先を指定して実行します
// サーバー内部から呼ぶ際は stderr を渡すことで stdio モードの MCP stream を汚染しません
func runLoginWithOutput(out io.Writer) error {
	fmt.Fprintln(out, "=== O'Reilly Cookie セットアップ ===")
	fmt.Fprintln(out)

	// XDGディレクトリを解決（OREILLY_USER_ID/PASSWORDは不要）
	xdgDirs, err := GetXDGDirs(os.Getenv("ORM_MCP_GO_DEBUG_DIR"))
	if err != nil {
		return fmt.Errorf("XDGディレクトリの解決に失敗しました: %w", err)
	}

	if err := xdgDirs.EnsureExists(); err != nil {
		return fmt.Errorf("XDGディレクトリの作成に失敗しました: %w", err)
	}

	fmt.Fprintln(out, "Chrome を起動してログインページを開きます。ログインするとCookieを自動保存します。")

	cm := cookie.NewCookieManager(xdgDirs.CacheHome)
	if err := browser.RunVisibleLogin(xdgDirs.ChromeSetupDataDir(), cm); err != nil {
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "✓ Cookieを保存しました: %s\n", xdgDirs.CookiePath())
	fmt.Fprintln(out, "次回から `orm-discovery-mcp-go` を実行すると、Cookieでログインできます。")
	return nil
}
