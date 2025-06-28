package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	Port            string
	Debug           bool
	MCPDebug        bool
	Transport       string
	OReillyUserID   string
	OReillyPassword string
	TmpDir          string
	LogLevel        slog.Level
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
	if debugStr := getEnv("ORM_MCP_GO_DEBUG"); debugStr != "" {
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
		// この時点ではまだslogが設定されていないため、標準的なログ出力を使用
		log.Fatalf("OREILLY_USER_ID と OREILLY_PASSWORD が設定されていません")
	}

	// 一時ディレクトリの取得
	tmpDir := getEnv("ORM_MCP_GO_TMP_DIR")
	if tmpDir == "" {
		// 環境変数が設定されていない場合は/var/tmpを使用
		tmpDir = "/var/tmp/"
	}

	// ログレベルの設定（デフォルト: INFO）
	logLevel := slog.LevelInfo
	if logLevelStr := getEnv("ORM_MCP_GO_LOG_LEVEL"); logLevelStr != "" {
		switch strings.ToUpper(logLevelStr) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN", "WARNING":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		default:
			// この時点ではまだslogが設定されていないため、標準的なログ出力を使用
			log.Printf("不明なログレベル: %s (INFOを使用)", logLevelStr)
		}
	}

	config := &Config{
		Port:            port,
		Debug:           debug,
		Transport:       transport,
		OReillyUserID:   OReillyUserID,
		OReillyPassword: OReillyPassword,
		TmpDir:          tmpDir,
		LogLevel:        logLevel,
	}

	// slogの設定
	setupLogger(config)

	return config, nil
}

// setupLogger はslogの設定を行います
func setupLogger(config *Config) {
	// テキストハンドラーを使用（JSONではなく）
	opts := &slog.HandlerOptions{
		Level: config.LogLevel,
		// カスタマイズ可能な属性（必要に応じて）
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// レベルを短縮形で表示
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				switch level {
				case slog.LevelDebug:
					a.Value = slog.StringValue("DBG")
				case slog.LevelInfo:
					a.Value = slog.StringValue("INF")
				case slog.LevelWarn:
					a.Value = slog.StringValue("WRN")
				case slog.LevelError:
					a.Value = slog.StringValue("ERR")
				}
			}
			return a
		},
	}

	// テキストハンドラーを作成
	handler := slog.NewTextHandler(os.Stdout, opts)

	// デフォルトロガーを設定
	slog.SetDefault(slog.New(handler))

	// 設定完了後にデバッグ情報をログ出力
	slog.Info("ログシステムを初期化しました",
		"log_level", config.LogLevel.String(),
		"debug_mode", config.Debug)
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
