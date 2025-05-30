package main

import (
	"os"
	"strconv"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	Port       int
	Debug      bool
	Transport  string
	OReillyJWT string
}

// LoadConfig は環境変数から設定を読み込みます
func LoadConfig() (*Config, error) {
	// ポート番号の取得（デフォルト: 8080）
	port := 8080
	if portStr := os.Getenv("PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	// デバッグモードの取得（デフォルト: false）
	debug := false
	if debugStr := os.Getenv("DEBUG"); debugStr != "" {
		if d, err := strconv.ParseBool(debugStr); err == nil {
			debug = d
		}
	}

	transport := "http"
	if transportStr := os.Getenv("TRANSPORT"); transportStr != "" {
		transport = transportStr
	}

	return &Config{
		Port:       port,
		Debug:      debug,
		Transport:  transport,
		OReillyJWT: os.Getenv("ORM_JWT"),
	}, nil
}
