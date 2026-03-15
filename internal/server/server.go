package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/cookie"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/config"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/history"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/mcputil"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/sampling"
)

// errH is the shared error handler for MCP responses.
var errH mcputil.ErrorHandler

// HTTP server timeout constants.
const (
	httpReadTimeout     = 30 * time.Second
	httpWriteTimeout    = 60 * time.Second
	httpIdleTimeout     = 120 * time.Second
	httpShutdownTimeout = 5 * time.Second
)

// Server is the MCP server implementation.
type Server struct {
	clientMu        sync.RWMutex
	browserClient   browser.Client
	server          *mcp.Server
	config          *config.Config
	historyManager  *history.Manager
	samplingManager *sampling.Manager
	cookieManager   cookie.Manager // 再認証時の BrowserClient 再生成に使用
	startedAt       time.Time      // サーバー起動時刻 (MCP 再起動検証用)
	serverVersion   string
}

// NewServer creates a new server instance.
func NewServer(browserClient browser.Client, cfg *config.Config, cookieManager cookie.Manager, serverVersion string) *Server {
	// Create MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "orm-discovery-mcp-go",
			Version: serverVersion,
		},
		&mcp.ServerOptions{
			Instructions: "O'Reilly Learning Platform MCP Server. " +
				"Use oreilly_search_content to discover books/videos/articles, " +
				"oreilly_ask_question for AI-powered Q&A, " +
				"and oreilly://book-* resources for detailed content access. " +
				"Always cite sources with title, author(s), and O'Reilly Media.",
		},
	)

	// Initialize research history manager
	historyManager := history.NewManager(
		cfg.XDGDirs.ResearchHistoryPath(),
		cfg.History.MaxEntries,
	)
	if err := historyManager.Load(); err != nil {
		slog.Warn("調査履歴の読み込みに失敗しました", "error", err)
	}

	// Initialize sampling manager
	samplingManager := sampling.NewManager(cfg)

	srv := &Server{
		browserClient:   browserClient,
		server:          mcpServer,
		config:          cfg,
		historyManager:  historyManager,
		samplingManager: samplingManager,
		cookieManager:   cookieManager,
		startedAt:       time.Now(),
		serverVersion:   serverVersion,
	}

	// Add middleware for logging
	mf := mcputil.MiddlewareFactory{LogLevel: cfg.Log.Level}
	mcpServer.AddReceivingMiddleware(
		mf.Logging(),
		mf.ToolLogging(),
	)

	slog.Info("サーバーを初期化しました")

	srv.registerHandlers()
	slog.Info("ハンドラーを登録しました")

	return srv
}

// getBrowserClient は browserClient を mutex で保護して返します。
func (s *Server) getBrowserClient() browser.Client {
	s.clientMu.RLock()
	defer s.clientMu.RUnlock()
	return s.browserClient
}

// setBrowserClient は browserClient を mutex で保護して設定します。
func (s *Server) setBrowserClient(client browser.Client) {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	s.browserClient = client
}

// Close はサーバーが保持する BrowserClient をクリーンアップします。
// degraded モードで後から設定された BrowserClient も確実に Close されます。
func (s *Server) Close() {
	if client := s.getBrowserClient(); client != nil {
		client.Close()
	}
}

// StartStreamableHTTPServer starts the HTTP server.
func (s *Server) StartStreamableHTTPServer(ctx context.Context, addr string) error {
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	// DNS rebinding protection は go-sdk v1.4.0 の StreamableHTTPHandler が
	// ビルトインで提供する (Host ヘッダー vs 実際のリスニングアドレスを検証)。
	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return s.server
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true,
		},
	)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}

	slog.Info("HTTPサーバーを作成しました")

	// Handle graceful shutdown
	go func() { // #nosec G118 -- shutdown handler intentionally uses context.Background for cleanup after parent ctx is cancelled
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTPサーバーのシャットダウンに失敗しました", "error", err)
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// StartStdioServer starts the stdio server.
func (s *Server) StartStdioServer(ctx context.Context) error {
	slog.Info("MCPサーバーを標準入出力で起動します")
	if err := s.server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	return nil
}

// registerHandlers registers the tool and resource handlers.
func (s *Server) registerHandlers() {
	// Add search tool
	searchTool := &mcp.Tool{
		Name:        "oreilly_search_content",
		Title:       "Search O'Reilly Content",
		Description: descSearchContent,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(true),
		},
	}
	mcp.AddTool(s.server, searchTool, s.SearchContentHandler)

	// Add ask question tool
	askQuestionTool := &mcp.Tool{
		Name:        "oreilly_ask_question",
		Title:       "Ask O'Reilly Answers AI",
		Description: descAskQuestion,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(true),
		},
	}
	mcp.AddTool(s.server, askQuestionTool, s.AskQuestionHandler)

	// Add reauthenticate tool
	reauthTool := &mcp.Tool{
		Name:  "oreilly_reauthenticate",
		Title: "Re-authenticate O'Reilly Session",
		Description: "O'Reillyセッションを再認証します。" +
			"Cookieが有効な場合は認証済みを返します。" +
			"Cookieが期限切れの場合はGoogle Chromeを起動してログインページを開きます。" +
			"ブラウザでログインが完了すると自動的にCookieを保存してサーバーの認証状態を更新します。",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(false),
		},
	}
	mcp.AddTool(s.server, reauthTool, s.ReauthenticateHandler)

	// Register resources
	s.registerResources()

	// Register history resources
	s.registerHistoryResources()

	// Register prompts
	s.registerPrompts()
}

// newSessionLogger creates an MCP session-scoped logger that sends log
// notifications to the connected client. Returns a no-op logger if session is unavailable.
func newSessionLogger(session *mcp.ServerSession, loggerName string) *slog.Logger {
	if session == nil {
		return slog.New(slog.DiscardHandler)
	}
	return slog.New(mcp.NewLoggingHandler(session, &mcp.LoggingHandlerOptions{
		LoggerName: loggerName,
	}))
}

func newToolResultError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func ptrBool(b bool) *bool { return &b }
