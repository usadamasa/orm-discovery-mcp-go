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

	summarizeBooksTool := mcp.NewTool("summarize_books",
		mcp.WithDescription("Search and summarize multiple books in Japanese from O'Reilly Learning Platform"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query to find books on O'Reilly Learning Platform"),
		),
		mcp.WithNumber("max_books",
			mcp.Description("Maximum number of books to summarize (default: 5)"),
		),
		mcp.WithArray("languages",
			mcp.Description("Languages to search in (default: ['en', 'ja'])"),
		),
	)
	s.mcpServer.AddTool(summarizeBooksTool, s.SummarizeBooksHandler)

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

// SummarizeBooksHandler は複数の書籍を検索して日本語でまとめます
func (s *Server) SummarizeBooksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("書籍まとめリクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		Query     string   `json:"query"`
		MaxBooks  int      `json:"max_books,omitempty"`
		Languages []string `json:"languages,omitempty"`
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

	// デフォルト値の設定
	maxBooks := requestParams.MaxBooks
	if maxBooks <= 0 {
		maxBooks = 5
	}

	languages := requestParams.Languages
	if len(languages) == 0 {
		languages = []string{"en", "ja"}
	}

	// SearchParamsに変換（書籍のみを検索）
	searchParams := SearchParams{
		Query:        requestParams.Query + " content_type:book",
		Rows:         maxBooks * 2, // 余裕を持って多めに取得
		Languages:    languages,
		TzOffset:     -9, // JST
		AiaOnly:      false,
		FeatureFlags: "improveSearchFilters",
		Report:       true,
		IsTopics:     false,
	}

	// O'Reilly APIで検索を実行
	log.Printf("O'Reillyクライアント呼び出し前")
	results, err := s.oreillyClient.Search(ctx, searchParams)
	if err != nil {
		log.Printf("O'Reillyクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil
	}
	log.Printf("O'Reillyクライアント呼び出し後: %d件の結果", len(results.Results))

	// 書籍のみをフィルタリングし、指定された数まで制限
	var books []SearchResult
	for _, result := range results.Results {
		if result.Type == "book" && len(books) < maxBooks {
			books = append(books, result)
		}
	}

	if len(books) == 0 {
		return mcp.NewToolResultText("指定されたクエリに該当する書籍が見つかりませんでした。"), nil
	}

	// 日本語でまとめを作成
	summary := s.createBooksSummary(requestParams.Query, books)

	return mcp.NewToolResultText(summary), nil
}

// createBooksSummary は複数の書籍情報を日本語でまとめます
func (s *Server) createBooksSummary(query string, books []SearchResult) string {
	summary := fmt.Sprintf("# 「%s」に関する書籍まとめ\n\n", query)
	summary += fmt.Sprintf("検索結果: %d冊の書籍が見つかりました。\n\n", len(books))

	// 各書籍の詳細情報
	summary += "## 📚 書籍一覧\n\n"
	for i, book := range books {
		summary += fmt.Sprintf("### %d. %s\n\n", i+1, book.Title)
		
		// 著者情報
		if len(book.Authors) > 0 {
			summary += fmt.Sprintf("**著者**: %s\n", joinStrings(book.Authors, ", "))
		}
		
		// 出版社情報
		if len(book.Publishers) > 0 {
			summary += fmt.Sprintf("**出版社**: %s\n", joinStrings(book.Publishers, ", "))
		}
		
		// 言語
		if book.Language != "" {
			summary += fmt.Sprintf("**言語**: %s\n", book.Language)
		}
		
		// 説明
		if book.Description != "" {
			summary += fmt.Sprintf("**概要**: %s\n", book.Description)
		}
		
		// トピック
		if len(book.Topics) > 0 {
			summary += fmt.Sprintf("**トピック**: %s\n", joinStrings(book.Topics, ", "))
		}
		
		// URL
		if book.WebURL != "" {
			summary += fmt.Sprintf("**リンク**: [O'Reilly Learning Platformで読む](%s)\n", book.WebURL)
		}
		
		summary += "\n---\n\n"
	}

	// 全体的な分析
	summary += "## 📊 分析結果\n\n"
	
	// 著者の統計
	authorCount := make(map[string]int)
	for _, book := range books {
		for _, author := range book.Authors {
			authorCount[author]++
		}
	}
	
	if len(authorCount) > 0 {
		summary += "**主要な著者**:\n"
		for author, count := range authorCount {
			if count > 1 {
				summary += fmt.Sprintf("- %s (%d冊)\n", author, count)
			}
		}
		summary += "\n"
	}
	
	// トピックの統計
	topicCount := make(map[string]int)
	for _, book := range books {
		for _, topic := range book.Topics {
			topicCount[topic]++
		}
	}
	
	if len(topicCount) > 0 {
		summary += "**関連トピック**:\n"
		for topic, count := range topicCount {
			if count > 1 {
				summary += fmt.Sprintf("- %s (%d冊)\n", topic, count)
			}
		}
		summary += "\n"
	}
	
	// 言語の統計
	langCount := make(map[string]int)
	for _, book := range books {
		if book.Language != "" {
			langCount[book.Language]++
		}
	}
	
	if len(langCount) > 0 {
		summary += "**言語別分布**:\n"
		for lang, count := range langCount {
			summary += fmt.Sprintf("- %s: %d冊\n", lang, count)
		}
		summary += "\n"
	}

	summary += "## 💡 学習の推奨事項\n\n"
	summary += "これらの書籍は「" + query + "」というテーマに関連しており、"
	summary += "体系的に学習することで理解を深めることができます。\n\n"
	summary += "初心者の方は基礎的な内容から始めて、徐々に応用的な書籍に進むことをお勧めします。\n"

	return summary
}

// joinStrings は文字列スライスを指定された区切り文字で結合します
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
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
