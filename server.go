package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server はMCPサーバーの実装です
type Server struct {
	oreillyClient *OreillyClient
	mcpServer     *server.MCPServer
}

// NewServer は新しいサーバーインスタンスを作成します
func NewServer(oreillyClient *OreillyClient) *Server {
	// MCPサーバーの設定とデバッグログの追加
	mcpServer := server.NewMCPServer(
		"Search O'Reilly Learning Platform",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	srv := &Server{
		oreillyClient: oreillyClient,
		mcpServer:     mcpServer,
	}
	// 初期化処理の成功を確認するためのログ
	log.Printf("サーバーを初期化しました")

	srv.registerHandlers()
	log.Printf("ハンドラーを登録しました")

	return srv
}

// StartStreamableHTTPServer はMCPサーバを返します
func (s *Server) StartStreamableHTTPServer(port string) error {
	// タイムアウト設定を調整したサーバーを作成
	httpServer := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)
	log.Printf("HTTPサーバーを作成しました")
	err := httpServer.Start(port)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) StartStdioServer() error {
	// MCPサーバーを標準入出力で起動
	log.Printf("MCPサーバーを標準入出力で起動します")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	return nil
}

// registerHandlers はハンドラーを登録します
func (s *Server) registerHandlers() {
	// Add tool
	searchTool := mcp.NewTool("search_content",
		mcp.WithDescription("Search content on O'Reilly Learning Platform. Returns URLs that can be used with get_book_details_by_url tool."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query to find content on O'Reilly Learning Platform"),
		),
		mcp.WithNumber("rows",
			mcp.Description("Number of results to return (default: 100)"),
		),
		mcp.WithArray("languages",
			mcp.Description("Languages to search in (default: ['en', 'ja'])"),
		),
		mcp.WithNumber("tzOffset",
			mcp.Description("Timezone offset (default: -9 for JST)"),
		),
		mcp.WithBoolean("aia_only",
			mcp.Description("Search only AI-assisted content (default: false)"),
		),
		mcp.WithString("feature_flags",
			mcp.Description("Feature flags (default: 'improveSearchFilters')"),
		),
		mcp.WithBoolean("report",
			mcp.Description("Include reporting data (default: true)"),
		),
		mcp.WithBoolean("isTopics",
			mcp.Description("Search topics only (default: false)"),
		),
	)
	s.mcpServer.AddTool(searchTool, s.SearchContentHandler)

	// 書籍詳細取得ツールの追加
	getBookDetailsTool := mcp.NewTool("get_book_details",
		mcp.WithDescription("Get detailed book information and table of contents from O'Reilly. Accepts a book product ID."),
		mcp.WithString("product_id",
			mcp.Description("Book product ID or ISBN (e.g., 9781098166298)"),
			mcp.Required(),
		),
	)
	s.mcpServer.AddTool(getBookDetailsTool, s.GetBookDetailsHandler)

	// 書籍目次取得ツールの追加
	getBookTOCTool := mcp.NewTool("get_book_toc",
		mcp.WithDescription("Get table of contents from O'Reilly book. Accepts a book product ID."),
		mcp.WithString("product_id",
			mcp.Description("Book product ID or ISBN (e.g., 9781098166298)"),
			mcp.Required(),
		),
	)
	s.mcpServer.AddTool(getBookTOCTool, s.GetBookTOCHandler)

	// チャプター本文取得ツールの追加
	getBookChapterContentTool := mcp.NewTool("get_book_chapter_content",
		mcp.WithDescription("Get structured chapter content from O'Reilly book. Accepts a book product ID and chapter name."),
		mcp.WithString("product_id",
			mcp.Description("Book product ID or ISBN (e.g., 9781098131814)"),
			mcp.Required(),
		),
		mcp.WithString("chapter_name",
			mcp.Description("Chapter name from flat-toc (e.g., preface01)"),
			mcp.Required(),
		),
	)
	s.mcpServer.AddTool(getBookChapterContentTool, s.GetBookChapterContentHandler)

	s.mcpServer.AddNotificationHandler("ping", s.handlePing)
}

// SearchContentHandler は検索リクエストを処理します
func (s *Server) SearchContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		Query        string   `json:"query"`
		Rows         int      `json:"rows,omitempty"`
		Languages    []string `json:"languages,omitempty"`
		TzOffset     int      `json:"tzOffset,omitempty"`
		AiaOnly      bool     `json:"aia_only,omitempty"`
		FeatureFlags string   `json:"feature_flags,omitempty"`
		Report       bool     `json:"report,omitempty"`
		IsTopics     bool     `json:"isTopics,omitempty"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// SearchParamsに変換
	searchParams := SearchParams{
		Query:        requestParams.Query,
		Rows:         requestParams.Rows,
		Languages:    requestParams.Languages,
		TzOffset:     requestParams.TzOffset,
		AiaOnly:      requestParams.AiaOnly,
		FeatureFlags: requestParams.FeatureFlags,
		Report:       requestParams.Report,
		IsTopics:     requestParams.IsTopics,
	}

	// O'Reilly APIで検索を実行
	log.Printf("O'Reillyクライアント呼び出し前")
	results, err := s.oreillyClient.Search(ctx, searchParams)
	if err != nil {
		log.Printf("O'Reillyクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil
	}
	log.Printf("O'Reillyクライアント呼び出し後: %v", results)

	// 結果をレスポンスに変換
	response := struct {
		Count   int           `json:"count"`
		Total   int           `json:"total"`
		Results []interface{} `json:"results"`
	}{
		Count:   len(results.Results),
		Total:   results.Total,
		Results: make([]interface{}, 0, len(results.Results)),
	}

	for _, result := range results.Results {
		response.Results = append(response.Results, map[string]interface{}{
			"id":          result.ID,
			"title":       result.Title,
			"description": result.Description,
			"url":         result.URL,
			"web_url":     result.WebURL,
			"type":        result.Type,
			"authors":     result.Authors,
			"publishers":  result.Publishers,
			"topics":      result.Topics,
			"language":    result.Language,
			"metadata":    result.Metadata,
		})
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	// レスポンスを返す
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// pingハンドラーの実装
func (s *Server) handlePing(ctx context.Context, notification mcp.JSONRPCNotification) {
	log.Printf("pingリクエスト受信: %+v", notification)
	// セッションを取得してpongを送信
	if session := server.ClientSessionFromContext(ctx); session != nil {
		select {
		case session.NotificationChannel() <- mcp.JSONRPCNotification{
			JSONRPC: "2.0"}:
		default:
			log.Printf("Failed to send pong notification")
		}
	}
}

// GetBookDetailsHandler handles book detail requests with URL or product ID
func (s *Server) GetBookDetailsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("書籍詳細取得リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		ProductID string `json:"product_id"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	// Either product_id or url must be provided
	if requestParams.ProductID == "" {
		return mcp.NewToolResultError("product_id parameter is required"), nil
	}

	// ブラウザクライアントで書籍詳細を取得
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	var result interface{}

	// 書籍詳細を取得
	log.Printf("プロダクトIDから書籍詳細を取得: %s", requestParams.ProductID)
	bookOverview, err := s.oreillyClient.browserClient.GetBookDetails(requestParams.ProductID)
	if err != nil {
		log.Printf("プロダクトID指定書籍詳細取得失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get book details by ProductID: %v", err)), nil
	}
	result = bookOverview

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetBookTOCHandler handles the book table of contents requests
func (s *Server) GetBookTOCHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("書籍目次取得リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		ProductID string `json:"product_id"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	// product_id must be provided
	if requestParams.ProductID == "" {
		return mcp.NewToolResultError("product_id parameter is required"), nil
	}

	// ブラウザクライアントの確認
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	// プロダクトIDから書籍目次を取得
	log.Printf("プロダクトIDから書籍目次を取得: %s", requestParams.ProductID)
	tocResponse, err := s.oreillyClient.browserClient.GetBookTOC(requestParams.ProductID)
	if err != nil {
		log.Printf("プロダクトID指定書籍目次取得失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get book TOC by ProductID: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(tocResponse)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetBookChapterContentHandler handles the book chapter content requests
func (s *Server) GetBookChapterContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("書籍チャプター本文取得リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		ProductID   string `json:"product_id"`
		ChapterName string `json:"chapter_name"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	// product_id and chapter_name must be provided
	if requestParams.ProductID == "" {
		return mcp.NewToolResultError("product_id parameter is required"), nil
	}
	if requestParams.ChapterName == "" {
		return mcp.NewToolResultError("chapter_name parameter is required"), nil
	}

	// ブラウザクライアントの確認
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	// プロダクトIDとチャプター名から書籍チャプター本文を取得
	log.Printf("プロダクトIDとチャプター名から書籍チャプター本文を取得: %s/%s", requestParams.ProductID, requestParams.ChapterName)
	chapterResponse, err := s.oreillyClient.browserClient.GetBookChapterContent(requestParams.ProductID, requestParams.ChapterName)
	if err != nil {
		log.Printf("書籍チャプター本文取得失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get book chapter content: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(chapterResponse)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}
