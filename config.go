package main

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config はアプリケーションの設定を保持します
type Config struct {
	Port            string
	Debug           bool
	MCPDebug        bool
	Transport       string
	OReillyUserID   string
	OReillyPassword string
	XDGDirs         *XDGDirs // XDG Base Directory準拠のディレクトリパス
	LogLevel        slog.Level
	LogFile         string // ログファイルパス(空の場合はstderrのみ)

	// ログローテーション設定
	LogMaxSizeMB  int // メインログ最大サイズ (MB)、デフォルト: 10
	LogMaxBackups int // メインログ世代数、デフォルト: 3
	LogMaxAgeDays int // メインログ保持日数、デフォルト: 30

	// Research History 設定
	HistoryMaxEntries int // 保持する最大エントリ数、デフォルト: 1000

	// Search Mode 設定
	DefaultSearchMode string // デフォルトの探索モード: "bfs" | "dfs"、デフォルト: "bfs"

	// Sampling 設定
	EnableSampling    bool // Sampling機能を有効にするか、デフォルト: true
	SamplingMaxTokens int  // Sampling時の最大トークン数、デフォルト: 500

	// HTTP サーバー設定
	BindAddress    string   // バインドアドレス、デフォルト: "127.0.0.1"
	AllowedOrigins []string // 許可する Origin のリスト (ALLOWED_ORIGINS 環境変数、カンマ区切り)
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

	// XDGディレクトリの解決
	// ORM_MCP_GO_DEBUG_DIR: デバッグ用に全ディレクトリを上書きする環境変数
	// 注意: この時点ではslogが未初期化のため、エラーはstderrにのみ出力される
	debugDir := getEnv("ORM_MCP_GO_DEBUG_DIR")
	xdgDirs, err := GetXDGDirs(debugDir)
	if err != nil {
		log.Fatalf("XDGディレクトリの解決に失敗しました: %v", err)
	}
	// ディレクトリを作成
	if err := xdgDirs.EnsureExists(); err != nil {
		log.Fatalf("XDGディレクトリの作成に失敗しました: %v", err)
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

	// ログファイルパスの取得（XDG Base Directory Specification準拠）
	logFile := xdgDirs.LogPath()

	// ログローテーション設定の取得
	logMaxSizeMB := 10
	if sizeStr := getEnv("ORM_MCP_GO_LOG_MAX_SIZE_MB"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 {
			logMaxSizeMB = size
		}
	}

	logMaxBackups := 3
	if backupsStr := getEnv("ORM_MCP_GO_LOG_MAX_BACKUPS"); backupsStr != "" {
		if backups, err := strconv.Atoi(backupsStr); err == nil && backups > 0 {
			logMaxBackups = backups
		}
	}

	logMaxAgeDays := 30
	if ageStr := getEnv("ORM_MCP_GO_LOG_MAX_AGE_DAYS"); ageStr != "" {
		if age, err := strconv.Atoi(ageStr); err == nil && age > 0 {
			logMaxAgeDays = age
		}
	}

	// Research History 設定
	historyMaxEntries := 1000
	if entriesStr := getEnv("ORM_MCP_GO_HISTORY_MAX_ENTRIES"); entriesStr != "" {
		if entries, err := strconv.Atoi(entriesStr); err == nil && entries > 0 {
			historyMaxEntries = entries
		}
	}

	// Search Mode 設定
	defaultSearchMode := "bfs"
	if modeStr := getEnv("ORM_MCP_GO_DEFAULT_MODE"); modeStr != "" {
		if modeStr == "bfs" || modeStr == "dfs" {
			defaultSearchMode = modeStr
		}
	}

	// Sampling 設定
	enableSampling := true
	if samplingStr := getEnv("ORM_MCP_GO_ENABLE_SAMPLING"); samplingStr != "" {
		if enabled, err := strconv.ParseBool(samplingStr); err == nil {
			enableSampling = enabled
		}
	}

	samplingMaxTokens := 500
	if tokensStr := getEnv("ORM_MCP_GO_SAMPLING_MAX_TOKENS"); tokensStr != "" {
		if tokens, err := strconv.Atoi(tokensStr); err == nil && tokens > 0 {
			samplingMaxTokens = tokens
		}
	}

	// HTTP サーバー設定
	bindAddress := "127.0.0.1"
	if addr := getEnv("BIND_ADDRESS"); addr != "" {
		bindAddress = addr
	}

	var allowedOrigins []string
	if origins := getEnv("ALLOWED_ORIGINS"); origins != "" {
		for o := range strings.SplitSeq(origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	config := &Config{
		Port:              port,
		Debug:             debug,
		Transport:         transport,
		OReillyUserID:     OReillyUserID,
		OReillyPassword:   OReillyPassword,
		XDGDirs:           xdgDirs,
		LogLevel:          logLevel,
		LogFile:           logFile,
		LogMaxSizeMB:      logMaxSizeMB,
		LogMaxBackups:     logMaxBackups,
		LogMaxAgeDays:     logMaxAgeDays,
		HistoryMaxEntries: historyMaxEntries,
		DefaultSearchMode: defaultSearchMode,
		EnableSampling:    enableSampling,
		SamplingMaxTokens: samplingMaxTokens,
		BindAddress:       bindAddress,
		AllowedOrigins:    allowedOrigins,
	}

	// slogの設定
	setupLogger(config)

	return config, nil
}

// setupLogger はslogの設定を行います
func setupLogger(config *Config) {
	// ログ出力先の決定
	var writer io.Writer = os.Stderr

	// ログファイルが指定されている場合はLumberjackでローテーションを設定
	if config.LogFile != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   config.LogFile,
			MaxSize:    config.LogMaxSizeMB,  // MB
			MaxBackups: config.LogMaxBackups, // 世代数
			MaxAge:     config.LogMaxAgeDays, // 日数
			Compress:   true,                 // 古いログを圧縮
		}
		writer = io.MultiWriter(os.Stderr, lumberjackLogger)
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
		logAttrs = append(logAttrs,
			"log_file", config.LogFile,
			"log_max_size_mb", config.LogMaxSizeMB,
			"log_max_backups", config.LogMaxBackups,
			"log_max_age_days", config.LogMaxAgeDays,
		)
	}
	slog.Info("ログシステムを初期化しました", logAttrs...)

	// XDGディレクトリ情報をログ出力
	if config.XDGDirs != nil {
		slog.Debug("XDGディレクトリを設定しました",
			"state_home", config.XDGDirs.StateHome,
			"cache_home", config.XDGDirs.CacheHome,
			"config_home", config.XDGDirs.ConfigHome,
		)
	}
}

// getEnv は環境変数を取得します（.envファイルの値が優先されます）
func getEnv(key string) string {
	return os.Getenv(key)
}
