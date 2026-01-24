package main

import (
	"io"
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
	LogFile         string // ログファイルパス(空の場合はstderrのみ)
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

	// ログファイルパスの取得
	logFile := getEnv("ORM_MCP_GO_LOG_FILE")

	config := &Config{
		Port:            port,
		Debug:           debug,
		Transport:       transport,
		OReillyUserID:   OReillyUserID,
		OReillyPassword: OReillyPassword,
		TmpDir:          tmpDir,
		LogLevel:        logLevel,
		LogFile:         logFile,
	}

	// slogの設定
	setupLogger(config)

	return config, nil
}

// setupLogger はslogの設定を行います
func setupLogger(config *Config) {
	// ログ出力先の決定
	var writer io.Writer = os.Stderr

	// ログファイルが指定されている場合はMultiWriterを使用
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("ログファイルを開けません: %v (stderrのみ使用)", err)
		} else {
			writer = io.MultiWriter(os.Stderr, file)
		}
	}

	// シンプルなテキストハンドラー設定
	opts := &slog.HandlerOptions{
		Level:     config.LogLevel,
		AddSource: true, // 関数名と行番号を表示
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// ソースパスを簡潔に表示
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					// github.com/usadamasa/orm-discovery-mcp-go/ 部分を削除
					if idx := strings.LastIndex(source.File, "orm-discovery-mcp-go/"); idx != -1 {
						source.File = source.File[idx+len("orm-discovery-mcp-go/"):]
					}
				}
			}
			return a
		},
	}

	// テキストハンドラーを作成
	handler := slog.NewTextHandler(writer, opts)

	// デフォルトロガーを設定
	slog.SetDefault(slog.New(handler))

	// 設定完了後にデバッグ情報をログ出力
	logAttrs := []any{
		"log_level", config.LogLevel.String(),
		"debug_mode", config.Debug,
	}
	if config.LogFile != "" {
		logAttrs = append(logAttrs, "log_file", config.LogFile)
	}
	slog.Info("ログシステムを初期化しました", logAttrs...)
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
