package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/config"
	versionpkg "github.com/usadamasa/orm-discovery-mcp-go/internal/version"
)

// Version information embedded by GoReleaser.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		info := versionpkg.Resolve(version, commit, date)
		fmt.Printf("orm-discovery-mcp-go %s\n", info.DisplayString())
		os.Exit(0)
	}

	// Handle --login flag (手動ログインからCookieを保存)
	// OREILLY_USER_ID / OREILLY_PASSWORD は不要
	if len(os.Args) > 1 && os.Args[1] == "--login" {
		if err := runLogin(); err != nil {
			fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
			os.Exit(1)
		}
		return
	}

	runMCPServer()
}

func runMCPServer() {
	// Create context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("シグナルを受信しました", "signal", sig)
		cancel()
	}()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("設定を読み込みました")

	// Initialize BrowserClient
	slog.Info("ブラウザクライアントを使用してO'Reillyにログインします...")

	// Create cookie manager (using CacheHome)
	cookieManager := cookie.NewCookieManager(cfg.XDGDirs.CacheHome)

	// デバッグモード: 共有 XDG パスからデバッグ用 cookie をシード
	if debugDir := os.Getenv("ORM_MCP_GO_DEBUG_DIR"); debugDir != "" {
		defaultDirs, err := config.GetXDGDirs("")
		if err != nil {
			slog.Warn("デフォルトXDGディレクトリの取得に失敗しました", "error", err)
		} else if err := cookieManager.SeedDebugCookieIfNeeded(defaultDirs.CookiePath()); err != nil {
			slog.Warn("デバッグ用Cookieのシードに失敗しました", "error", err)
		}
	}

	// Create browser client and login (using StateHome for Chrome temp data)
	// browser.Client インターフェースとして宣言し、エラー時は nil (interface nil) のまま渡す。
	// typed nil (*BrowserClient(nil)) を渡すと == nil チェックが正しく動作しないため。
	var browserClient browser.Client
	bc, err := browser.NewBrowserClient(cookieManager, cfg.Debug, cfg.XDGDirs.StateHome)
	if err != nil {
		slog.Warn("ブラウザクライアントの初期化に失敗しました。degraded モードで起動します。"+
			"oreilly_reauthenticate ツールで再認証してください。", "error", err)
	} else {
		browserClient = bc
		slog.Info("ブラウザクライアントの初期化が完了しました")
	}
	s := NewServer(browserClient, cfg, cookieManager, version)
	defer s.Close() // Clean up browser on process exit (includes clients created in degraded mode)

	if cfg.Transport == "http" {
		if err := s.StartStreamableHTTPServer(ctx, fmt.Sprintf("%s:%s", cfg.BindAddress, cfg.Port)); err != nil {
			slog.Error("HTTPサーバーの起動に失敗しました", "error", err, "addr", cfg.BindAddress, "port", cfg.Port)
			os.Exit(1)
		}
	} else {
		if err := s.StartStdioServer(ctx); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	slog.Info("サーバーが正常にシャットダウンしました")
}
