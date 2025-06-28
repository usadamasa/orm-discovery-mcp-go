package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/browser/cookie"
)

func main() {
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

	// Cookieマネージャーを作成
	cookieManager := cookie.NewCookieManager(cfg.TmpDir)

	// ブラウザクライアントを作成してログイン
	browserClient, err := browser.NewBrowserClient(cfg.OReillyUserID, cfg.OReillyPassword, cookieManager, cfg.Debug, cfg.TmpDir)
	if err != nil {
		slog.Error("ブラウザクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer browserClient.Close() // プロセス終了時にブラウザをクリーンアップ

	slog.Info("ブラウザクライアントの初期化が完了しました")
	s := NewServer(browserClient)

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
