package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"
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
		" Search O'Reilly Learning Platform",
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
		mcp.WithDescription("Search content on O'Reilly Learning Platform"),
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

	listCollectionsTool := mcp.NewTool("list_collections",
		mcp.WithDescription("List my collections on O'Reilly Learning Platform"),
	)
	s.mcpServer.AddTool(listCollectionsTool, s.ListCollectionsHandler)

	s.mcpServer.AddNotificationHandler("ping", s.handlePing)
}

// SearchContentHandler は検索リクエストを処理します
func (s *Server) SearchContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		Query        string      `json:"query"`
		Rows         int         `json:"rows,omitempty"`
		Languages    []string    `json:"languages,omitempty"`
		TzOffset     int         `json:"tzOffset,omitempty"`
		AiaOnly      bool        `json:"aia_only,omitempty"`
		FeatureFlags string      `json:"feature_flags,omitempty"`
		Report       bool        `json:"report,omitempty"`
		IsTopics     bool        `json:"isTopics,omitempty"`
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

func (s *Server) ListCollectionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("リクエスト受信: %+v", request)

	// O'Reilly APIでコレクション取得を実行
	log.Printf("O'Reillyクライアント呼び出し前")
	results, err := s.oreillyClient.ListCollections(ctx)
	if err != nil {
		log.Printf("O'Reillyクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil
	}
	log.Printf("O'Reillyクライアント呼び出し後: %v", results)

	// 結果をレスポンスに変換
	response := struct {
		Count   int           `json:"count"`
		Results []interface{} `json:"results"`
	}{
		Count:   len(results.Results),
		Results: make([]interface{}, 0, len(results.Results)),
	}

	for _, result := range results.Results {
		response.Results = append(response.Results, map[string]interface{}{
			"id":          result.ID,
			"title":       result.Name,
			"description": result.Description,
			"web_url":     result.WebURL,
			"type":        result.Type,
			"content":     result.Content,
		})
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	// レスポンスを返す
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
