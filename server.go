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
	mcpServer := server.NewMCPServer("Demo 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithLogging(), // ログ機能を有効化
	)

	srv := &Server{
		oreillyClient: oreillyClient,
		mcpServer:     mcpServer,
	}
	// 初期化処理の成功を確認するためのログ
	log.Printf("サーバーを初期化しました")

	srv.registerHandlers()

	return srv
}

// CreateNewServer はMCPサーバを返します
func (s *Server) CreateNewServer() *server.StreamableHTTPServer {
	// タイムアウト設定を調整したサーバーを作成
	httpServer := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)
	log.Printf("HTTPサーバーを作成しました")
	return httpServer
}

// registerHandlers はハンドラーを登録します
func (s *Server) registerHandlers() {
	// Add tool
	tool := mcp.NewTool("search_content",
		mcp.WithDescription("Search content on O'Reilly Learning Platform"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query to find content on O'Reilly Learning Platform"),
		),
	)

	s.mcpServer.AddTool(tool, s.SearchContentHandler)
	s.mcpServer.AddNotificationHandler("ping", s.handlePing)
	log.Printf("ハンドラーを登録しました")
}

// SearchContentHandler は検索リクエストを処理します
func (s *Server) SearchContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	fmt.Printf("request: %x", request)
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &params); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if params.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// デフォルト値の設定
	if params.Limit <= 0 {
		params.Limit = 10
	}

	// O'Reilly APIで検索を実行
	log.Printf("O'Reillyクライアント呼び出し前")
	results, err := s.oreillyClient.Search(ctx, params.Query, params.Limit)
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
		Count:   results.Count,
		Results: make([]interface{}, 0, len(results.Results)),
	}

	for _, result := range results.Results {
		response.Results = append(response.Results, map[string]interface{}{
			"id":          result.ID,
			"title":       result.Title,
			"description": result.Description,
			"url":         result.URL,
			"type":        result.Type,
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

// Shutdown はサーバーをシャットダウンします
func (s *Server) Shutdown(_ context.Context) error {
	return nil
}
