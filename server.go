package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	// MCPサーバーの設定
	return &Server{
		oreillyClient: oreillyClient,
		mcpServer: server.NewMCPServer("Demo 🚀",
			"1.0.0",
			server.WithToolCapabilities(false)),
	}
}

// CreateNewServer はMCPサーバを返します
func (s *Server) CreateNewServer() *server.StreamableHTTPServer {
	// ハンドラーの登録
	s.registerHandlers()

	// サーバーを起動
	return server.NewStreamableHTTPServer(s.mcpServer, server.WithStateLess(true))
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

	s.mcpServer.AddTool(tool, s.helloHandler)
}

// handleSearch は検索リクエストを処理します
func (s *Server) helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// リクエストパラメータの取得
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	fmt.Printf("%x", request)
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
	results, err := s.oreillyClient.Search(params.Query, params.Limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil
	}

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

// Shutdown はサーバーをシャットダウンします
func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}
