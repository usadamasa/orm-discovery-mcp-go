package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
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
	// 実行バイナリのディレクトリを取得
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("実行バイナリのパスを取得できませんでした: %v", err)
		log.Printf("カレントディレクトリの.envファイルを試行します")
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	envPath := filepath.Join(execDir, ".env")

	// 実行バイナリのディレクトリの.envファイルを読み込み（存在しない場合はエラーを無視）
	if err := godotenv.Load(envPath); err != nil {
		log.Printf(".envファイルが見つからないか読み込めませんでした (%s): %v", envPath, err)
		log.Printf("実行時の環境変数を使用します")
	} else {
		log.Printf(".envファイルを読み込みました: %s", envPath)
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
		OReillyUserID:   getEnv("OREILLY_USER_ID"),
		OReillyPassword: getEnv("OREILLY_PASSWORD"),
	}, nil
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
