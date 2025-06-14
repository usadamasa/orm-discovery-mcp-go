package main

import (
	"log"
	"os"
	"strconv"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	Port            string
	Debug           bool
	Transport       string
	OReillyUserID   string
	OReillyPassword string
}

// LoadConfig は.envファイルと環境変数から設定を読み込みます
func LoadConfig() (*Config, error) {
	// ポート番号の取得（デフォルト: 8080）
	port := "8080"
	if portStr := getEnv("PORT"); portStr != "" {
		port = portStr
	}

	// デバッグモードの取得（デフォルト: false）
	debug := false
	if debugStr := getEnv("DEBUG"); debugStr != "" {
		if d, err := strconv.ParseBool(debugStr); err == nil {
			debug = d
		}
	}

	transport := "stdio"
	if transportStr := getEnv("TRANSPORT"); transportStr != "" {
		transport = transportStr
	}

	// 認証情報の確認
	OReillyUserID := getEnv("OREILLY_USER_ID")
	OReillyPassword := getEnv("OREILLY_PASSWORD")
	if OReillyUserID == "" || OReillyPassword == "" {
		log.Fatalf("OREILLY_USER_ID と OREILLY_PASSWORD が設定されていません")
	}

	return &Config{
		Port:            port,
		Debug:           debug,
		Transport:       transport,
		OReillyUserID:   OReillyUserID,
		OReillyPassword: OReillyPassword,
	}, nil
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
