package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	Port          string
	Debug         bool
	Transport     string
	OReillyCookie string
	OReillyJWT    string
	SessionID     string
	RefreshToken  string
	OReillyUserID string
	OReillyPassword string
}

// LoadConfig は.envファイルと環境変数から設定を読み込みます
func LoadConfig() (*Config, error) {
	// .envファイルを読み込み（存在しない場合はエラーを無視）
	if err := godotenv.Load(); err != nil {
		log.Printf(".envファイルが見つからないか読み込めませんでした: %v", err)
		log.Printf("実行時の環境変数を使用します")
	} else {
		log.Printf(".envファイルを読み込みました")
	}

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

	return &Config{
		Port:            port,
		Debug:           debug,
		Transport:       transport,
		OReillyCookie:   getEnv("OREILLY_COOKIE"),
		OReillyJWT:      getEnv("OREILLY_JWT"),
		SessionID:       getEnv("OREILLY_SESSION_ID"),
		RefreshToken:    getEnv("OREILLY_REFRESH_TOKEN"),
		OReillyUserID:   getEnv("OREILLY_USER_ID"),
		OReillyPassword: getEnv("OREILLY_PASSWORD"),
	}, nil
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
