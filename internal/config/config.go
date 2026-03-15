package config

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ServerOpts はサーバー設定を保持する
type ServerOpts struct {
	Port        string
	Transport   string
	BindAddress string
}

// debugOpts はデバッグ設定を保持する (外部パッケージからの構築不要)
type debugOpts struct {
	Enabled  bool
	MCPDebug bool
}

// LogOpts はログ設定を保持する
type LogOpts struct {
	Level    slog.Level
	File     string
	Rotation logRotation
}

// logRotation はログローテーション設定を保持する (外部パッケージからの構築不要)
type logRotation struct {
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
}

// HistoryOpts は調査履歴設定を保持する
type HistoryOpts struct {
	MaxEntries int
}

// SamplingOpts はサンプリング設定を保持する
type SamplingOpts struct {
	Enabled   bool
	MaxTokens int
}

// Config はアプリケーションの設定を保持します
type Config struct {
	Server   ServerOpts
	Debug    debugOpts
	XDGDirs  *XDGDirs
	Log      LogOpts
	History  HistoryOpts
	Sampling SamplingOpts
}

// envString returns the environment variable value, or defaultVal if unset.
func envString(key, defaultVal string) string {
	if v := getEnv(key); v != "" {
		return v
	}
	return defaultVal
}

// envBool returns the environment variable parsed as bool, or defaultVal on error/unset.
func envBool(key string, defaultVal bool) bool {
	if v := getEnv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

// envInt returns the environment variable parsed as int, or defaultVal if invalid/unset.
// Values below minVal are treated as invalid and return defaultVal.
func envInt(key string, defaultVal, minVal int) int {
	if v := getEnv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= minVal {
			return n
		}
	}
	return defaultVal
}

// parseLogLevel converts a log level string to slog.Level.
// Supports standard levels (DEBUG, INFO, WARN, ERROR) plus "WARNING" as alias.
func parseLogLevel(s string) slog.Level {
	// "WARNING" is a common alias not supported by slog.Level.UnmarshalText
	if strings.EqualFold(s, "WARNING") {
		return slog.LevelWarn
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		log.Printf("不明なログレベル: %s (INFOを使用)", s)
		return slog.LevelInfo
	}
	return level
}

// LoadConfig は.envファイルと環境変数から設定を読み込みます
func LoadConfig() (*Config, error) {
	// XDGディレクトリの解決
	debugDir := getEnv("ORM_MCP_GO_DEBUG_DIR")
	xdgDirs, err := GetXDGDirs(debugDir)
	if err != nil {
		log.Fatalf("XDGディレクトリの解決に失敗しました: %v", err)
	}
	if err := xdgDirs.EnsureExists(); err != nil {
		log.Fatalf("XDGディレクトリの作成に失敗しました: %v", err)
	}

	logLevel := slog.LevelInfo
	if s := getEnv("ORM_MCP_GO_LOG_LEVEL"); s != "" {
		logLevel = parseLogLevel(s)
	}

	config := &Config{
		Server: ServerOpts{
			Port:        envString("PORT", "8080"),
			Transport:   envString("TRANSPORT", "stdio"),
			BindAddress: envString("BIND_ADDRESS", "127.0.0.1"),
		},
		Debug: debugOpts{
			Enabled: envBool("ORM_MCP_GO_DEBUG", false),
		},
		XDGDirs: xdgDirs,
		Log: LogOpts{
			Level: logLevel,
			File:  xdgDirs.LogPath(),
			Rotation: logRotation{
				MaxSizeMB:  envInt("ORM_MCP_GO_LOG_MAX_SIZE_MB", 10, 1),
				MaxBackups: envInt("ORM_MCP_GO_LOG_MAX_BACKUPS", 3, 1),
				MaxAgeDays: envInt("ORM_MCP_GO_LOG_MAX_AGE_DAYS", 30, 1),
			},
		},
		History: HistoryOpts{
			MaxEntries: envInt("ORM_MCP_GO_HISTORY_MAX_ENTRIES", 1000, 1),
		},
		Sampling: SamplingOpts{
			Enabled:   envBool("ORM_MCP_GO_ENABLE_SAMPLING", true),
			MaxTokens: envInt("ORM_MCP_GO_SAMPLING_MAX_TOKENS", 500, 1),
		},
	}

	setupLogger(config)
	return config, nil
}

// setupLogger はslogの設定を行います
func setupLogger(config *Config) {
	// ログ出力先の決定
	var writer io.Writer = os.Stderr

	// ログファイルが指定されている場合はLumberjackでローテーションを設定
	if config.Log.File != "" {
		lumberjackLogger := &lumberjack.Logger{
			Filename:   config.Log.File,
			MaxSize:    config.Log.Rotation.MaxSizeMB,
			MaxBackups: config.Log.Rotation.MaxBackups,
			MaxAge:     config.Log.Rotation.MaxAgeDays,
			Compress:   true,
		}
		writer = io.MultiWriter(os.Stderr, lumberjackLogger)
	}

	// シンプルなテキストハンドラー設定
	opts := &slog.HandlerOptions{
		Level:     config.Log.Level,
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
		"log_level", config.Log.Level.String(),
		"debug_mode", config.Debug.Enabled,
	}
	if config.Log.File != "" {
		logAttrs = append(logAttrs,
			"log_file", config.Log.File,
			"log_max_size_mb", config.Log.Rotation.MaxSizeMB,
			"log_max_backups", config.Log.Rotation.MaxBackups,
			"log_max_age_days", config.Log.Rotation.MaxAgeDays,
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
