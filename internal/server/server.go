package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser/cookie"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/cache"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/config"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/history"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/mcputil"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/sampling"
)

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
		cfg.HistoryMaxEntries,
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
	mcpServer.AddReceivingMiddleware(
		mcputil.CreateLoggingMiddleware(cfg.LogLevel),
		mcputil.CreateToolLoggingMiddleware(cfg.LogLevel),
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
	go func() {
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

// registerResources registers the resource handlers.
func (s *Server) registerResources() {
	// Book details resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-details/{product_id}",
			Name:        "O'Reilly Book Details",
			Description: descResBookDetails,
			MIMEType:    "application/json",
		},
		s.GetBookDetailsResource,
	)

	// Book TOC resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-toc/{product_id}",
			Name:        "O'Reilly Book Table of Contents",
			Description: descResBookTOC,
			MIMEType:    "application/json",
		},
		s.GetBookTOCResource,
	)

	// Book chapter content resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-chapter/{product_id}/{chapter_name}",
			Name:        "O'Reilly Book Chapter Content",
			Description: descResBookChapter,
			MIMEType:    "application/json",
		},
		s.GetBookChapterContentResource,
	)

	// Answer resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://answer/{question_id}",
			Name:        "O'Reilly Answers Response",
			Description: descResAnswer,
			MIMEType:    "application/json",
		},
		s.GetAnswerResource,
	)

	// Server status resource (for MCP restart verification)
	s.server.AddResource(
		&mcp.Resource{
			URI:         "orm-mcp://server/status",
			Name:        "MCP Server Status",
			Description: "Server startup time and version for restart verification",
			MIMEType:    "application/json",
		},
		s.GetServerStatusResource,
	)

	// Resource Templates for dynamic discovery
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-details/{product_id}",
			Name:        "O'Reilly Book Details Template",
			Description: descTmplBookDetails,
			MIMEType:    "application/json",
		},
		s.GetBookDetailsResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-toc/{product_id}",
			Name:        "O'Reilly Book TOC Template",
			Description: descTmplBookTOC,
			MIMEType:    "application/json",
		},
		s.GetBookTOCResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-chapter/{product_id}/{chapter_name}",
			Name:        "O'Reilly Book Chapter Template",
			Description: descTmplBookChapter,
			MIMEType:    "application/json",
		},
		s.GetBookChapterContentResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://answer/{question_id}",
			Name:        "O'Reilly Answers Template",
			Description: descTmplAnswer,
			MIMEType:    "application/json",
		},
		s.GetAnswerResource,
	)
}

// GetServerStatusResource returns server startup time and version for restart verification.
func (s *Server) GetServerStatusResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	status := map[string]string{
		"started_at": s.startedAt.UTC().Format(time.RFC3339),
		"version":    s.serverVersion,
	}
	jsonBytes, _ := json.Marshal(status)
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// SearchContentHandler handles search requests.
func (s *Server) SearchContentHandler(ctx context.Context, req *mcp.CallToolRequest, args SearchContentArgs) (*mcp.CallToolResult, *SearchContentResult, error) {
	slog.Debug("検索リクエスト受信")
	sessionLog := newSessionLogger(req.Session, "oreilly-search")
	start := time.Now()

	if s.getBrowserClient() == nil {
		return newToolResultError("O'Reilly セッションが認証されていません。" +
			"oreilly_reauthenticate ツールを呼び出してログインしてください。"), nil, nil
	}

	if args.Query == "" {
		return newToolResultError(mcputil.UserFacingErrorMessage(mcputil.ErrorCategoryValidation)), nil, nil
	}
	if len(args.Query) > maxQueryLength {
		return newToolResultError(fmt.Sprintf("Query is too long. Please use %d characters or fewer.", maxQueryLength)), nil, nil
	}

	// Set default values
	if args.Rows <= 0 {
		args.Rows = 25
	}
	if args.Rows > maxRows {
		args.Rows = maxRows
	}
	if args.Offset < 0 {
		args.Offset = 0
	}
	if len(args.Languages) == 0 {
		args.Languages = []string{"en", "ja"}
	}

	// Prepare options for BrowserClient
	options := map[string]any{
		"rows":          args.Rows,
		"offset":        args.Offset,
		"languages":     args.Languages,
		"tzOffset":      args.TzOffset,
		"aia_only":      args.AiaOnly,
		"feature_flags": args.FeatureFlags,
		"report":        args.Report,
		"isTopics":      args.IsTopics,
	}

	// Execute search using BrowserClient
	slog.Debug("BrowserClient検索開始", "query", args.Query, "offset", args.Offset, "rows", args.Rows)
	results, totalResults, err := s.getBrowserClient().SearchContent(args.Query, options)
	if err != nil && mcputil.CategorizeError(err) == mcputil.ErrorCategoryAuth {
		// Attempt re-authentication
		slog.Info("認証エラー検出: 再認証を試みます")
		if reauthErr := s.getBrowserClient().Reauthenticate(); reauthErr != nil {
			return newToolResultError(mcputil.SanitizeError(reauthErr, "operation", "reauthenticate")), nil, nil
		}

		// Retry
		results, totalResults, err = s.getBrowserClient().SearchContent(args.Query, options)
	}
	if err != nil {
		return newToolResultError(mcputil.SanitizeError(err, "operation", "search", "query", args.Query)), nil, nil
	}
	slog.Info("検索完了", "query", args.Query, "result_count", len(results), "total_results", totalResults)
	sessionLog.InfoContext(ctx, "検索完了", "query", args.Query, "result_count", len(results), "total_results", totalResults)

	// Generate history ID upfront so cache file includes it (single-save pattern)
	historyID := history.GenerateRequestID()

	// Save full results to cache file (single save with history ID)
	cacheDir := s.config.XDGDirs.ResponseCachePath()
	filePath, cacheErr := cache.SaveResponseAsMarkdown(cacheDir, args.Query, results, historyID, totalResults)
	if cacheErr != nil {
		slog.Warn("レスポンスキャッシュの保存に失敗しました", "error", cacheErr)
	}

	// Record to research history (pass pre-generated historyID)
	s.recordSearchHistory(args.Query, options, results, filePath, time.Since(start), historyID)

	// Build lightweight response
	toolResult, structured := s.buildLightweightResponse(results, historyID, filePath, args.Offset, totalResults)

	// Return Markdown format if requested
	if args.Format == ResponseFormatMarkdown && structured != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatSearchResultsMarkdown(structured)}},
		}, structured, nil
	}

	return toolResult, structured, nil
}

// buildLightweightResponse builds a lightweight response with file path for lazy loading.
// Book results include ResourceLink entries for direct resource navigation.
// Returns up to 5 results in the text summary.
func (s *Server) buildLightweightResponse(results []map[string]any, historyID, filePath string, offset, totalResults int) (*mcp.CallToolResult, *SearchContentResult) {
	total := cache.EffectiveTotalResults(totalResults, len(results))

	lightweightResults := make([]map[string]any, 0, len(results))
	var resourceLinks []mcp.Content

	for _, result := range results {
		lightweight := make(map[string]any)

		id := cache.ExtractStringField(result, "product_id", "isbn", "id")
		if id != "" {
			lightweight["id"] = id
		}

		title, _ := result["title"].(string)
		if title != "" {
			lightweight["title"] = title
		}

		if names := extractAuthorSlice(result["authors"]); len(names) > 0 {
			lightweight["authors"] = names
		}

		lightweightResults = append(lightweightResults, lightweight)

		// Add ResourceLink for book content types
		if ct, _ := result["content_type"].(string); id != "" && ct == browser.ContentTypeBook {
			name := title
			if name == "" {
				name = id
			}
			resourceLinks = append(resourceLinks, &mcp.ResourceLink{
				URI:      "oreilly://book-details/" + id,
				Name:     name,
				MIMEType: "application/json",
			})
		}
	}

	hasMore, nextOffset := calcPagination(offset, len(results), total)

	// Limit structured results for context efficiency
	const inlineSummaryLimit = 5
	topResults := lightweightResults
	if len(topResults) > inlineSummaryLimit {
		topResults = topResults[:inlineSummaryLimit]
	}

	// Build text summary with top results and file path
	var textParts []string
	for i, r := range topResults {
		title, _ := r["title"].(string)
		id, _ := r["id"].(string)
		line := fmt.Sprintf("%d. %s (ID: %s)", i+1, title, id)
		textParts = append(textParts, line)
	}
	if len(lightweightResults) > inlineSummaryLimit {
		textParts = append(textParts, fmt.Sprintf("... and %d more results", len(lightweightResults)-5))
	}

	if filePath != "" {
		textParts = append(textParts, fmt.Sprintf("\nFull details saved to: %s\nUse the Read tool to access detailed results.", filePath))
	}

	if hasMore {
		textParts = append(textParts, fmt.Sprintf("\nMore results available: use offset=%d to get next page.", nextOffset))
	}

	structured := &SearchContentResult{
		Count:        len(results),
		Total:        len(results),
		TotalResults: total,
		HasMore:      hasMore,
		NextOffset:   nextOffset,
		Results:      topResults,
		HistoryID:    historyID,
		FilePath:     filePath,
	}

	var content []mcp.Content
	if len(textParts) > 0 {
		content = append(content, &mcp.TextContent{Text: strings.Join(textParts, "\n")})
	}
	content = append(content, resourceLinks...)

	if len(content) > 0 {
		return &mcp.CallToolResult{Content: content}, structured
	}
	return nil, structured
}

// AskQuestionHandler processes question requests for O'Reilly Answers.
func (s *Server) AskQuestionHandler(ctx context.Context, req *mcp.CallToolRequest, args AskQuestionArgs) (*mcp.CallToolResult, *AskQuestionResult, error) {
	slog.Debug("質問リクエスト受信")
	sessionLog := newSessionLogger(req.Session, "oreilly-ask")
	start := time.Now()

	if args.Question == "" {
		return newToolResultError(mcputil.UserFacingErrorMessage(mcputil.ErrorCategoryValidation)), nil, nil
	}
	if len(args.Question) > maxQuestionLength {
		return newToolResultError(fmt.Sprintf("Question is too long. Please use %d characters or fewer.", maxQuestionLength)), nil, nil
	}

	// Default timeout (5 minutes)
	maxWaitTime := 300 * time.Second
	if args.MaxWaitTimeSeconds > 0 {
		if args.MaxWaitTimeSeconds > 600 { // Max 10 minutes
			args.MaxWaitTimeSeconds = 600
		}
		maxWaitTime = time.Duration(args.MaxWaitTimeSeconds) * time.Second
	}

	// Check browser client
	if s.getBrowserClient() == nil {
		return newToolResultError("browser client is not available"), nil, nil
	}

	slog.Info("質問処理開始", "question", args.Question, "max_wait_time", maxWaitTime)
	sessionLog.InfoContext(ctx, "質問処理開始", "question", args.Question, "max_wait_time", maxWaitTime)

	// Execute question (with polling)
	answer, err := s.getBrowserClient().AskQuestion(args.Question, maxWaitTime)
	if err != nil {
		return newToolResultError(mcputil.SanitizeError(err, "operation", "ask_question", "question", args.Question)), nil, nil
	}

	slog.Info("質問に対する回答を取得しました", "question", args.Question, "question_id", answer.QuestionID)
	sessionLog.InfoContext(ctx, "回答取得完了", "question", args.Question, "question_id", answer.QuestionID)

	// Record to research history
	s.recordQuestionHistory(args.Question, answer, time.Since(start))

	// Build StructuredContent response
	structured := &AskQuestionResult{
		QuestionID:          answer.QuestionID,
		Question:            args.Question,
		Answer:              answer.MisoResponse.Data.Answer,
		IsFinished:          answer.IsFinished,
		Sources:             answer.MisoResponse.Data.Sources,
		RelatedResources:    answer.MisoResponse.Data.RelatedResources,
		AffiliationProducts: answer.MisoResponse.Data.AffiliationProducts,
		FollowupQuestions:   answer.MisoResponse.Data.FollowupQuestions,
		CitationNote:        "IMPORTANT: When referencing this information, always cite the sources listed above with proper attribution to O'Reilly Media.",
	}

	// Return Markdown format if requested
	if args.Format == ResponseFormatMarkdown {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: formatAskQuestionMarkdown(structured)}},
		}, structured, nil
	}

	return nil, structured, nil
}

// GetBookDetailsResource handles book detail resource requests.
func (s *Server) GetBookDetailsResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("書籍詳細リソース取得リクエスト受信", "uri", req.Params.URI)

	// Extract product_id from URI
	productID := extractProductIDFromURI(req.Params.URI)
	if productID == "" {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id not found in URI"}`,
			}},
		}, nil
	}

	// Check browser client
	if s.getBrowserClient() == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	bookOverview, err := s.getBrowserClient().GetBookDetails(productID)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "get_book_details", "product_id", productID), nil
	}

	jsonBytes, err := json.Marshal(bookOverview)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "marshal_book_details"), nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// GetBookTOCResource handles book TOC resource requests.
func (s *Server) GetBookTOCResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("書籍目次リソース取得リクエスト受信", "uri", req.Params.URI)

	// Extract product_id from URI
	productID := extractProductIDFromURI(req.Params.URI)
	if productID == "" {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id not found in URI"}`,
			}},
		}, nil
	}

	// Check browser client
	if s.getBrowserClient() == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	tocResponse, err := s.getBrowserClient().GetBookTOC(productID)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "get_book_toc", "product_id", productID), nil
	}

	jsonBytes, err := json.Marshal(tocResponse)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "marshal_book_toc"), nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// GetBookChapterContentResource handles book chapter content resource requests.
func (s *Server) GetBookChapterContentResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("書籍チャプター本文リソース取得リクエスト受信", "uri", req.Params.URI)

	// Extract product_id and chapter_name from URI
	productID, chapterName := extractProductIDAndChapterFromURI(req.Params.URI)
	if productID == "" || chapterName == "" {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id or chapter_name not found in URI"}`,
			}},
		}, nil
	}

	// Check browser client
	if s.getBrowserClient() == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	chapterResponse, err := s.getBrowserClient().GetBookChapterContent(productID, chapterName)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "get_chapter", "product_id", productID, "chapter_name", chapterName), nil
	}

	jsonBytes, err := json.Marshal(chapterResponse)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "marshal_chapter"), nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// GetAnswerResource handles answer resource requests.
func (s *Server) GetAnswerResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("回答リソース取得リクエスト受信", "uri", req.Params.URI)

	// Extract question_id from URI
	questionID := extractQuestionIDFromURI(req.Params.URI)
	if questionID == "" {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "question_id not found in URI"}`,
			}},
		}, nil
	}

	// Check browser client
	if s.getBrowserClient() == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	// Get answer
	answer, err := s.getBrowserClient().GetQuestionByID(questionID)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "get_answer", "question_id", questionID), nil
	}

	// Build response
	response := struct {
		QuestionID          string                       `json:"question_id"`
		Answer              string                       `json:"answer"`
		IsFinished          bool                         `json:"is_finished"`
		Sources             []browser.AnswerSource       `json:"sources"`
		RelatedResources    []browser.RelatedResource    `json:"related_resources"`
		AffiliationProducts []browser.AffiliationProduct `json:"affiliation_products"`
		FollowupQuestions   []string                     `json:"followup_questions"`
		CitationNote        string                       `json:"citation_note"`
	}{
		QuestionID:          answer.QuestionID,
		Answer:              answer.MisoResponse.Data.Answer,
		IsFinished:          answer.IsFinished,
		Sources:             answer.MisoResponse.Data.Sources,
		RelatedResources:    answer.MisoResponse.Data.RelatedResources,
		AffiliationProducts: answer.MisoResponse.Data.AffiliationProducts,
		FollowupQuestions:   answer.MisoResponse.Data.FollowupQuestions,
		CitationNote:        "IMPORTANT: When referencing this information, always cite the sources listed above with proper attribution to O'Reilly Media.",
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcputil.ErrorResourceContents(req.Params.URI, err, "operation", "marshal_answer"), nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// ReauthenticateHandler handles the oreilly_reauthenticate MCP tool.
// Cookie が有効ならそのまま返し、期限切れなら BrowserClient.Reauthenticate() で
// ビジブルブラウザを起動して再認証します。
// browserClient が nil の場合 (degraded モード) は NewBrowserClient() で
// 新しい BrowserClient を生成します (内部でビジブルログインを実行)。
func (s *Server) ReauthenticateHandler(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ struct{},
) (*mcp.CallToolResult, *ReauthResult, error) {
	// degraded モード: browserClient が nil = サーバーが認証なしで起動した状態
	if s.getBrowserClient() == nil {
		slog.Info("oreilly_reauthenticate: degraded モード - NewBrowserClient で認証を開始します")
		client, err := browser.NewBrowserClient(
			s.cookieManager,
			s.config.Debug,
			s.config.XDGDirs.StateHome,
		)
		if err != nil {
			return newToolResultError(mcputil.SanitizeError(err, "operation", "create_browser_client")), nil, nil
		}
		s.setBrowserClient(client)
		return nil, &ReauthResult{
			Status:  "setup_completed",
			Message: "再認証が完了しました。O'Reilly セッションが更新されました。",
		}, nil
	}

	// 通常モード: 1. 現在の Cookie で認証チェック
	if err := s.getBrowserClient().CheckAndResetAuth(); err == nil {
		return nil, &ReauthResult{
			Status:  "authenticated",
			Message: "O'Reilly セッションは有効です。",
		}, nil
	}

	// 2. Reauthenticate() でビジブルブラウザを起動して再認証
	slog.Info("oreilly_reauthenticate: Reauthenticate() で再認証を開始します")
	if err := s.getBrowserClient().Reauthenticate(); err != nil {
		return newToolResultError(mcputil.SanitizeError(err, "operation", "reauthenticate")), nil, nil
	}

	return nil, &ReauthResult{
		Status:  "setup_completed",
		Message: "再認証が完了しました。O'Reilly セッションが更新されました。",
	}, nil
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

// Helper functions for tool results

func newToolResultError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

// Helper functions to extract parameters from URIs

func extractProductIDFromURI(uri string) string {
	// Extract product_id from URIs like "oreilly://book-details/{product_id}" or "oreilly://book-toc/{product_id}"
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	id := strings.TrimPrefix(u.Path, "/")
	return id
}

func extractProductIDAndChapterFromURI(uri string) (string, string) {
	// Extract product_id and chapter_name from URIs like "oreilly://book-chapter/{product_id}/{chapter_name}"
	if uri == "" {
		return "", ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", ""
	}
	// Use RawPath to preserve %2F in chapter names; fall back to Path if RawPath is empty
	rawPath := u.RawPath
	if rawPath == "" {
		rawPath = u.Path
	}
	rawPath = strings.TrimPrefix(rawPath, "/")
	parts := strings.SplitN(rawPath, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	productID, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", ""
	}
	chapterName, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", ""
	}
	return productID, chapterName
}

func extractQuestionIDFromURI(uri string) string {
	// Extract question_id from URIs like "oreilly://answer/{question_id}"
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	id := strings.TrimPrefix(u.Path, "/")
	return id
}

// recordSearchHistory records a search to the research history.
// If entryID is provided, it is used as the history entry ID (to match the cache file).
func (s *Server) recordSearchHistory(query string, options map[string]any, results []map[string]any, filePath string, duration time.Duration, entryID string) {
	if s.historyManager == nil {
		return
	}

	// Build top results summary
	topResults := make([]history.TopResultSummary, 0, 5)
	for i, result := range results {
		if i >= 5 {
			break
		}
		summary := history.TopResultSummary{}
		if title, ok := result["title"].(string); ok {
			summary.Title = title
		}
		if authors, ok := result["authors"].([]any); ok && len(authors) > 0 {
			if author, ok := authors[0].(string); ok {
				summary.Author = author
			}
		}
		if productID, ok := result["product_id"].(string); ok {
			summary.ProductID = productID
		} else if isbn, ok := result["isbn"].(string); ok {
			summary.ProductID = isbn
		}
		topResults = append(topResults, summary)
	}

	if entryID == "" {
		entryID = history.GenerateRequestID()
	}

	entry := history.Entry{
		ID:         entryID,
		Type:       "search",
		Query:      query,
		ToolName:   "oreilly_search_content",
		Parameters: options,
		ResultSummary: history.ResultSummary{
			Count:      len(results),
			TopResults: topResults,
		},
		DurationMs: duration.Milliseconds(),
		FilePath:   filePath,
	}

	if err := s.historyManager.AddEntry(entry); err != nil {
		slog.Warn("調査履歴の追加に失敗しました", "error", err)
		return
	}

	if err := s.historyManager.Save(); err != nil {
		slog.Warn("調査履歴の保存に失敗しました", "error", err)
	}
}

func ptrBool(b bool) *bool { return &b }

// recordQuestionHistory records a question to the research history.
func (s *Server) recordQuestionHistory(question string, answer *browser.AnswerResponse, duration time.Duration) {
	if s.historyManager == nil {
		return
	}

	// Build answer preview (first 200 characters)
	answerPreview := answer.MisoResponse.Data.Answer
	if len(answerPreview) > 200 {
		answerPreview = answerPreview[:200] + "..."
	}

	entry := history.Entry{
		Type:     "question",
		Query:    question,
		ToolName: "oreilly_ask_question",
		ResultSummary: history.ResultSummary{
			AnswerPreview: answerPreview,
			SourcesCount:  len(answer.MisoResponse.Data.Sources),
			FollowupCount: len(answer.MisoResponse.Data.FollowupQuestions),
		},
		DurationMs: duration.Milliseconds(),
	}

	if err := s.historyManager.AddEntry(entry); err != nil {
		slog.Warn("調査履歴の追加に失敗しました", "error", err)
		return
	}

	if err := s.historyManager.Save(); err != nil {
		slog.Warn("調査履歴の保存に失敗しました", "error", err)
	}
}
