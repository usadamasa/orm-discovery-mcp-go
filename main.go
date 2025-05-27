package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 設定の読み込み
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// O'Reillyクライアントの初期化
	oreillyClient := NewOreillyClient(cfg.OReillyJWT)

	// サーバーの初期化
	srv := NewServer(oreillyClient)

	// シグナルハンドリングの設定
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// サーバーを別ゴルーチンで起動
	go func() {
		httpServer := srv.CreateNewServer()
		port := fmt.Sprintf(":%d", cfg.Port)
		log.Printf("HTTP server listening on %s", port)
		if err := httpServer.Start(port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// シグナルを受信するまで待機
	sig := <-sigChan
	log.Printf("シグナルを受信しました: %v. シャットダウンしています...", sig)

	// グレースフルシャットダウン
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("サーバーのシャットダウン中にエラーが発生しました: %v", err)
	}

	log.Println("サーバーが正常にシャットダウンしました")
}
