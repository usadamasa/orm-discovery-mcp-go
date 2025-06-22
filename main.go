package main

import (
	"fmt"
	"log"
)

func main() {
	runMCPServer()
}

func runMCPServer() {
	// 設定の読み込み
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}
	log.Printf("設定を読み込みました")

	// O'Reillyクライアントの初期化（ブラウザクライアントを使用）
	log.Printf("ブラウザクライアントを使用してO'Reillyにログインします...")

	oreillyClient, err := NewOreillyClient(cfg.OReillyUserID, cfg.OReillyPassword, cfg.Debug, cfg.TmpDir)
	if err != nil {
		log.Fatalf("ブラウザクライアントの初期化に失敗しました: %v", err)
	}
	defer oreillyClient.Close() // プロセス終了時にブラウザをクリーンアップ

	log.Printf("O'Reillyクライアントの初期化が完了しました")
	s := NewServer(oreillyClient)

	if cfg.Transport == "http" {
		if err := s.StartStreamableHTTPServer(fmt.Sprintf(":%s", cfg.Port)); err != nil {
			log.Fatalf("HTTPサーバーの起動に失敗しました: %v", err)
		}
	} else {
		if err := s.StartStdioServer(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	log.Println("サーバーが正常にシャットダウンしました")
}
