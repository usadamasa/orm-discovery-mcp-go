package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

// GoReleaserによって埋め込まれるバージョン情報
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// --versionフラグの処理
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("orm-discovery-mcp-go %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	runMCPServer()
}

func runMCPServer() {
	// 設定の読み込み
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("設定を読み込みました")

	// BrowserClientの直接初期化
	slog.Info("ブラウザクライアントを使用してO'Reillyにログインします...")

	// Cookieマネージャーを作成 (CacheHome を使用)
	cookieManager := cookie.NewCookieManager(cfg.XDGDirs.CacheHome)

	// ブラウザクライアントを作成してログイン (StateHome を使用 - Chrome一時データ用)
	browserClient, err := browser.NewBrowserClient(cfg.OReillyUserID, cfg.OReillyPassword, cookieManager, cfg.Debug, cfg.XDGDirs.StateHome)
	if err != nil {
		slog.Error("ブラウザクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer browserClient.Close() // プロセス終了時にブラウザをクリーンアップ

	slog.Info("ブラウザクライアントの初期化が完了しました")
	s := NewServer(browserClient, cfg)

	if cfg.Transport == "http" {
		if err := s.StartStreamableHTTPServer(fmt.Sprintf(":%s", cfg.Port)); err != nil {
			slog.Error("HTTPサーバーの起動に失敗しました", "error", err, "port", cfg.Port)
			os.Exit(1)
		}
	} else {
		if err := s.StartStdioServer(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	slog.Info("サーバーが正常にシャットダウンしました")
}
