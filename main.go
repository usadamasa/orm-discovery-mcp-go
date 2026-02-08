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
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("設定を読み込みました")

	// Initialize BrowserClient
	slog.Info("ブラウザクライアントを使用してO'Reillyにログインします...")

	// Create cookie manager (using CacheHome)
	cookieManager := cookie.NewCookieManager(cfg.XDGDirs.CacheHome)

	// Create browser client and login (using StateHome for Chrome temp data)
	browserClient, err := browser.NewBrowserClient(cfg.OReillyUserID, cfg.OReillyPassword, cookieManager, cfg.Debug, cfg.XDGDirs.StateHome)
	if err != nil {
		slog.Error("ブラウザクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer browserClient.Close() // Clean up browser on process exit

	slog.Info("ブラウザクライアントの初期化が完了しました")
	s := NewServer(browserClient, cfg)

	if cfg.Transport == "http" {
		if err := s.StartStreamableHTTPServer(ctx, fmt.Sprintf(":%s", cfg.Port)); err != nil {
			slog.Error("HTTPサーバーの起動に失敗しました", "error", err, "port", cfg.Port)
			os.Exit(1)
		}
	} else {
		if err := s.StartStdioServer(ctx); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	slog.Info("サーバーが正常にシャットダウンしました")
}
