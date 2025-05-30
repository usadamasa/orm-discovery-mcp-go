package main

import (
	"fmt"
	"log"
)

func main() {
	// 設定の読み込み
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}
	log.Printf("設定を読み込みました")

	// O'Reillyクライアントの初期化
	oreillyClient := NewOreillyClient(cfg.OReillyJWT)
	s := NewServer(oreillyClient)

	if cfg.Transport == "http" {
		log.Printf("HTTPサーバーを起動します")
		if err := s.StartStreamableHTTPServer(fmt.Sprintf(":%d", cfg.Port)); err != nil {
			log.Fatalf("HTTPサーバーの起動に失敗しました: %v", err)
		}
	} else {
		log.Printf("サーバーを起動します")
		if err := s.StartStdioServer(); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}
	log.Println("サーバーが正常にシャットダウンしました")
}
