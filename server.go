package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

// Server is the MCP server implementation.
type Server struct {
	browserClient   *browser.BrowserClient
	server          *mcp.Server
	config          *Config
	historyManager  *ResearchHistoryManager
	samplingManager *SamplingManager
}

// NewServer creates a new server instance.
func NewServer(browserClient *browser.BrowserClient, config *Config) *Server {
	// Create MCP server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "Search O'Reilly Learning Platform",
			Version: "1.0.0",
		},
		nil,
	)

	// Initialize research history manager
	historyManager := NewResearchHistoryManager(
		config.XDGDirs.ResearchHistoryPath(),
		config.HistoryMaxEntries,
	)
	if err := historyManager.Load(); err != nil {
		slog.Warn("調査履歴の読み込みに失敗しました", "error", err)
	}

	// Initialize sampling manager
	samplingManager := NewSamplingManager(config)

	srv := &Server{
		browserClient:   browserClient,
		server:          mcpServer,
		config:          config,
		historyManager:  historyManager,
		samplingManager: samplingManager,
	}

	// Add middleware for logging
	mcpServer.AddReceivingMiddleware(
		createLoggingMiddleware(config.LogLevel),
		createToolLoggingMiddleware(config.LogLevel),
	)

	slog.Info("サーバーを初期化しました")

	srv.registerHandlers()
	slog.Info("ハンドラーを登録しました")

	return srv
}

// originValidationMiddleware は Origin ヘッダーを検証して DNS rebinding 攻撃を防ぐミドルウェア。
//
// 許可ルール:
//   - Origin ヘッダーなし (ブラウザ以外のクライアント): 許可
//   - Origin が http://localhost:* または http://127.0.0.1:*: 許可
//   - allowedOrigins に含まれる: 許可
//   - それ以外: 403 Forbidden
func originValidationMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if isLocalOrigin(origin) {
			next.ServeHTTP(w, r)
			return
		}

		if slices.Contains(allowedOrigins, origin) {
			next.ServeHTTP(w, r)
			return
		}

		slog.Warn("不正な Origin からのリクエストを拒否しました", "origin", origin)
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

// isLocalOrigin は Origin がローカルホストからのリクエストかどうかを判定する。
func isLocalOrigin(origin string) bool {
	// http://localhost or http://localhost:PORT
	if origin == "http://localhost" || strings.HasPrefix(origin, "http://localhost:") {
		return true
	}
	// http://127.0.0.1 or http://127.0.0.1:PORT
	if origin == "http://127.0.0.1" || strings.HasPrefix(origin, "http://127.0.0.1:") {
		return true
	}
	return false
}

// StartStreamableHTTPServer starts the HTTP server.
func (s *Server) StartStreamableHTTPServer(ctx context.Context, addr string) error {
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return s.server
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true,
		},
	)

	handler := originValidationMiddleware(s.config.AllowedOrigins, mcpHandler)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("HTTPサーバーを作成しました")

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

// isAuthError checks if the error is an authentication error.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "authentication error") ||
		strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "403")
}

// registerHandlers registers the tool and resource handlers.
func (s *Server) registerHandlers() {
	// Add search tool
	searchTool := &mcp.Tool{
		Name: "oreilly_search_content",
		Description: `Search O'Reilly content and return books/videos/articles with product_id for resource access.

Example: "Docker containers" (Good) / "How to use Docker" (Poor)

Results: Use product_id with oreilly://book-details/{id} or oreilly://book-chapter/{id}/{chapter}

IMPORTANT: Cite sources with title, author(s), and O'Reilly Media.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Search O'Reilly Content",
			ReadOnlyHint:    true,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(true),
		},
	}
	mcp.AddTool(s.server, searchTool, s.SearchContentHandler)

	// Add ask question tool
	askQuestionTool := &mcp.Tool{
		Name: "oreilly_ask_question",
		Description: `Ask technical questions to O'Reilly Answers AI and get sourced responses.

Example: "How to optimize React performance?" (Good) / "Explain everything about React" (Poor)

Response: Markdown answer, sources, related resources, question_id (use with oreilly://answer/{id})

IMPORTANT: Cite sources provided in the response.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Ask O'Reilly Answers AI",
			ReadOnlyHint:    true,
			DestructiveHint: ptrBool(false),
			IdempotentHint:  true,
			OpenWorldHint:   ptrBool(true),
		},
	}
	mcp.AddTool(s.server, askQuestionTool, s.AskQuestionHandler)

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
			Description: "Get book info (title, authors, date, description, topics, TOC). Cite sources when referencing.",
			MIMEType:    "application/json",
		},
		s.GetBookDetailsResource,
	)

	// Book TOC resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-toc/{product_id}",
			Name:        "O'Reilly Book Table of Contents",
			Description: "Get table of contents with chapter names and structure. Cite book title, author(s), O'Reilly Media.",
			MIMEType:    "application/json",
		},
		s.GetBookTOCResource,
	)

	// Book chapter content resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://book-chapter/{product_id}/{chapter_name}",
			Name:        "O'Reilly Book Chapter Content",
			Description: "Get full chapter text. CRITICAL: Cite book title, author(s), chapter title, O'Reilly Media.",
			MIMEType:    "application/json",
		},
		s.GetBookChapterContentResource,
	)

	// Answer resource
	s.server.AddResource(
		&mcp.Resource{
			URI:         "oreilly://answer/{question_id}",
			Name:        "O'Reilly Answers Response",
			Description: "Retrieve previously generated answer by question_id. Cite sources when referencing.",
			MIMEType:    "application/json",
		},
		s.GetAnswerResource,
	)

	// Resource Templates for dynamic discovery
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-details/{product_id}",
			Name:        "O'Reilly Book Details Template",
			Description: "Use product_id from oreilly_search_content to get book details.",
			MIMEType:    "application/json",
		},
		s.GetBookDetailsResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-toc/{product_id}",
			Name:        "O'Reilly Book TOC Template",
			Description: "Use product_id from oreilly_search_content to get table of contents.",
			MIMEType:    "application/json",
		},
		s.GetBookTOCResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://book-chapter/{product_id}/{chapter_name}",
			Name:        "O'Reilly Book Chapter Template",
			Description: "Use product_id and chapter_name to get chapter content.",
			MIMEType:    "application/json",
		},
		s.GetBookChapterContentResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "oreilly://answer/{question_id}",
			Name:        "O'Reilly Answers Template",
			Description: "Use question_id from oreilly_ask_question to retrieve the answer.",
			MIMEType:    "application/json",
		},
		s.GetAnswerResource,
	)
}

// SearchContentHandler handles search requests.
func (s *Server) SearchContentHandler(ctx context.Context, req *mcp.CallToolRequest, args SearchContentArgs) (*mcp.CallToolResult, *SearchContentResult, error) {
	slog.Debug("検索リクエスト受信")
	start := time.Now()

	if args.Query == "" {
		return newToolResultError("query parameter is required"), nil, nil
	}

	// Set default values
	if args.Rows <= 0 {
		args.Rows = 25
	}
	if args.Rows > 100 {
		args.Rows = 100
	}
	if args.Offset < 0 {
		args.Offset = 0
	}
	if len(args.Languages) == 0 {
		args.Languages = []string{"en", "ja"}
	}

	// Set default mode to BFS
	mode := args.Mode
	if mode == "" {
		mode = SearchModeBFS
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
	slog.Debug("BrowserClient検索開始", "query", args.Query, "mode", mode, "offset", args.Offset, "rows", args.Rows)
	results, totalResults, err := s.browserClient.SearchContent(args.Query, options)
	if err != nil && isAuthError(err) {
		// Attempt re-authentication
		slog.Info("認証エラー検出: 再認証を試みます")
		if reauthErr := s.browserClient.ReauthenticateIfNeeded(s.config.OReillyUserID, s.config.OReillyPassword); reauthErr != nil {
			slog.Error("再認証失敗", "error", reauthErr)
			return newToolResultError(fmt.Sprintf("再認証に失敗しました: %v", reauthErr)), nil, nil
		}

		// Retry
		results, totalResults, err = s.browserClient.SearchContent(args.Query, options)
	}

	if err != nil {
		slog.Error("BrowserClient検索失敗", "error", err, "query", args.Query)
		return newToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil, nil
	}
	slog.Info("検索完了", "query", args.Query, "result_count", len(results), "total_results", totalResults, "mode", mode)

	// Record to research history and get the history ID
	historyID := s.recordSearchHistoryWithFullResponse(args.Query, options, results, time.Since(start))

	// Build response based on mode
	switch mode {
	case SearchModeBFS:
		return s.buildBFSResponse(results, historyID, args.Offset, totalResults)
	case SearchModeDFS:
		return s.buildDFSResponse(ctx, req.Session, args.Query, results, historyID, args.Summarize, args.Offset, totalResults)
	default:
		// Default to BFS for unknown modes
		return s.buildBFSResponse(results, historyID, args.Offset, totalResults)
	}
}

// buildBFSResponse builds a lightweight response for BFS mode.
func (s *Server) buildBFSResponse(results []map[string]any, historyID string, offset, totalResults int) (*mcp.CallToolResult, *SearchContentResult, error) {
	// Extract only essential fields: id, title, authors
	lightweightResults := make([]map[string]any, 0, len(results))
	for _, result := range results {
		lightweight := make(map[string]any)

		// Extract product_id or ISBN
		if productID, ok := result["product_id"].(string); ok {
			lightweight["id"] = productID
		} else if isbn, ok := result["isbn"].(string); ok {
			lightweight["id"] = isbn
		} else if id, ok := result["id"].(string); ok {
			lightweight["id"] = id
		}

		// Extract title
		if title, ok := result["title"].(string); ok {
			lightweight["title"] = title
		}

		// Extract authors
		if authors, ok := result["authors"].([]any); ok && len(authors) > 0 {
			lightweight["authors"] = authors
		} else if authors, ok := result["authors"].([]string); ok && len(authors) > 0 {
			lightweight["authors"] = authors
		}

		lightweightResults = append(lightweightResults, lightweight)
	}

	hasMore, nextOffset := calcPagination(offset, len(results), totalResults)

	note := "Use oreilly://book-details/{id} for full details. Full data available at orm-mcp://history/" + historyID + "/full"
	if hasMore {
		note += fmt.Sprintf(". More results available: use offset=%d to get next page.", nextOffset)
	}

	return nil, &SearchContentResult{
		Count:        len(results),
		Total:        len(results),
		TotalResults: totalResults,
		HasMore:      hasMore,
		NextOffset:   nextOffset,
		Results:      lightweightResults,
		Mode:         SearchModeBFS,
		HistoryID:    historyID,
		Note:         note,
	}, nil
}

// buildDFSResponse builds a detailed response for DFS mode.
func (s *Server) buildDFSResponse(
	ctx context.Context,
	session *mcp.ServerSession,
	query string,
	results []map[string]any,
	historyID string,
	summarize bool,
	offset, totalResults int,
) (*mcp.CallToolResult, *SearchContentResult, error) {
	resultMaps := make([]map[string]any, len(results))
	copy(resultMaps, results)

	hasMore, nextOffset := calcPagination(offset, len(results), totalResults)

	response := &SearchContentResult{
		Count:        len(results),
		Total:        len(results),
		TotalResults: totalResults,
		HasMore:      hasMore,
		NextOffset:   nextOffset,
		Results:      resultMaps,
		Mode:         SearchModeDFS,
		HistoryID:    historyID,
	}

	if hasMore {
		response.Note = fmt.Sprintf("More results available: use offset=%d to get next page.", nextOffset)
	}

	// Generate summary using MCP Sampling if requested
	if summarize && s.samplingManager != nil {
		summary, err := s.samplingManager.SummarizeSearchResults(ctx, session, query, results)
		if err != nil {
			slog.Warn("Sampling summarization failed", "error", err)
			if response.Note != "" {
				response.Note += " "
			}
			response.Note += "Summarization failed. Full results are included."
		} else if summary != "" {
			response.Summary = summary
		} else {
			if response.Note != "" {
				response.Note += " "
			}
			response.Note += "Summarization not available. Full results are included."
		}
	}

	return nil, response, nil
}

// AskQuestionHandler processes question requests for O'Reilly Answers.
func (s *Server) AskQuestionHandler(ctx context.Context, req *mcp.CallToolRequest, args AskQuestionArgs) (*mcp.CallToolResult, *AskQuestionResult, error) {
	slog.Debug("質問リクエスト受信")
	start := time.Now()

	if args.Question == "" {
		return newToolResultError("question parameter is required"), nil, nil
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
	if s.browserClient == nil {
		return newToolResultError("browser client is not available"), nil, nil
	}

	slog.Info("質問処理開始", "question", args.Question, "max_wait_time", maxWaitTime)

	// Execute question (with polling)
	answer, err := s.browserClient.AskQuestion(args.Question, maxWaitTime)
	if err != nil {
		slog.Error("質問処理失敗", "error", err, "question", args.Question)
		return newToolResultError(fmt.Sprintf("failed to ask question: %v", err)), nil, nil
	}

	slog.Info("質問に対する回答を取得しました", "question", args.Question, "question_id", answer.QuestionID)

	// Record to research history
	s.recordQuestionHistory(args.Question, answer, time.Since(start))

	// Build StructuredContent response
	return nil, &AskQuestionResult{
		QuestionID:          answer.QuestionID,
		Question:            args.Question,
		Answer:              answer.MisoResponse.Data.Answer,
		IsFinished:          answer.IsFinished,
		Sources:             answer.MisoResponse.Data.Sources,
		RelatedResources:    answer.MisoResponse.Data.RelatedResources,
		AffiliationProducts: answer.MisoResponse.Data.AffiliationProducts,
		FollowupQuestions:   answer.MisoResponse.Data.FollowupQuestions,
		CitationNote:        "IMPORTANT: When referencing this information, always cite the sources listed above with proper attribution to O'Reilly Media.",
	}, nil
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
	if s.browserClient == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	bookOverview, err := s.browserClient.GetBookDetails(productID)
	if err != nil {
		slog.Error("書籍詳細取得失敗", "error", err, "product_id", productID)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book details: %v"}`, err),
			}},
		}, nil
	}

	jsonBytes, err := json.Marshal(bookOverview)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
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
	if s.browserClient == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	tocResponse, err := s.browserClient.GetBookTOC(productID)
	if err != nil {
		slog.Error("書籍目次取得失敗", "error", err, "product_id", productID)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book TOC: %v"}`, err),
			}},
		}, nil
	}

	jsonBytes, err := json.Marshal(tocResponse)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
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
	if s.browserClient == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	chapterResponse, err := s.browserClient.GetBookChapterContent(productID, chapterName)
	if err != nil {
		slog.Error("書籍チャプター本文取得失敗", "error", err, "product_id", productID, "chapter_name", chapterName)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book chapter content: %v"}`, err),
			}},
		}, nil
	}

	jsonBytes, err := json.Marshal(chapterResponse)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
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
	if s.browserClient == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			}},
		}, nil
	}

	// Get answer
	answer, err := s.browserClient.GetQuestionByID(questionID)
	if err != nil {
		slog.Error("回答取得失敗", "error", err, "question_id", questionID)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get answer: %v"}`, err),
			}},
		}, nil
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
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
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

// recordSearchHistoryWithFullResponse records a search to the research history with full response data.
// Returns the history entry ID.
func (s *Server) recordSearchHistoryWithFullResponse(query string, options map[string]any, results []map[string]any, duration time.Duration) string {
	if s.historyManager == nil {
		return ""
	}

	// Build top results summary
	topResults := make([]TopResultSummary, 0, 5)
	for i, result := range results {
		if i >= 5 {
			break
		}
		summary := TopResultSummary{}
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

	// Generate entry ID
	entryID := GenerateRequestID()

	entry := ResearchEntry{
		ID:         entryID,
		Type:       "search",
		Query:      query,
		ToolName:   "oreilly_search_content",
		Parameters: options,
		ResultSummary: ResultSummary{
			Count:      len(results),
			TopResults: topResults,
		},
		DurationMs:   duration.Milliseconds(),
		FullResponse: results, // Store full response for later access
	}

	if err := s.historyManager.AddEntry(entry); err != nil {
		slog.Warn("調査履歴の追加に失敗しました", "error", err)
		return ""
	}

	if err := s.historyManager.Save(); err != nil {
		slog.Warn("調査履歴の保存に失敗しました", "error", err)
	}

	return entryID
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

	entry := ResearchEntry{
		Type:     "question",
		Query:    question,
		ToolName: "oreilly_ask_question",
		ResultSummary: ResultSummary{
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
