package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/cache"
)

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
		return newToolResultError(errH.ValidationMessage()), nil, nil
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
	if err != nil && errH.IsAuth(err) {
		// Attempt re-authentication
		slog.Info("認証エラー検出: 再認証を試みます")
		if reauthErr := s.getBrowserClient().Reauthenticate(); reauthErr != nil {
			return newToolResultError(errH.Sanitize(reauthErr, "operation", "reauthenticate")), nil, nil
		}

		// Retry
		results, totalResults, err = s.getBrowserClient().SearchContent(args.Query, options)
	}
	if err != nil {
		return newToolResultError(errH.Sanitize(err, "operation", "search", "query", args.Query)), nil, nil
	}
	slog.Info("検索完了", "query", args.Query, "result_count", len(results), "total_results", totalResults)
	sessionLog.InfoContext(ctx, "検索完了", "query", args.Query, "result_count", len(results), "total_results", totalResults)

	// Generate history ID upfront so cache file includes it (single-save pattern)
	historyID := generateRequestID()

	// Save full results to cache file (single save with history ID)
	cacheDir := s.config.XDGDirs.ResponseCachePath()
	filePath, cacheErr := cache.SaveResponseAsMarkdown(cache.SaveParams{
		Dir: cacheDir, Query: args.Query, Results: results, HistoryID: historyID, TotalResults: totalResults,
	})
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
		return newToolResultError(errH.ValidationMessage()), nil, nil
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
		return newToolResultError(errH.Sanitize(err, "operation", "ask_question", "question", args.Question)), nil, nil
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
			s.config.Debug.Enabled,
			s.config.XDGDirs.StateHome,
		)
		if err != nil {
			return newToolResultError(errH.Sanitize(err, "operation", "create_browser_client")), nil, nil
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
		return newToolResultError(errH.Sanitize(err, "operation", "reauthenticate")), nil, nil
	}

	return nil, &ReauthResult{
		Status:  "setup_completed",
		Message: "再認証が完了しました。O'Reilly セッションが更新されました。",
	}, nil
}
