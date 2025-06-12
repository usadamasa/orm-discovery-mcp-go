package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"log"

	"github.com/usadamasa/orm-discovery-mcp-go/browser"
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


	// プレイリスト管理ツールの追加
	listPlaylistsTool := mcp.NewTool("list_playlists",
		mcp.WithDescription("List playlists from O'Reilly Learning Platform"),
	)
	s.mcpServer.AddTool(listPlaylistsTool, s.ListPlaylistsHandler)

	createPlaylistTool := mcp.NewTool("create_playlist",
		mcp.WithDescription("Create a new playlist on O'Reilly Learning Platform"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the playlist"),
		),
		mcp.WithString("description",
			mcp.Description("Description of the playlist"),
		),
		mcp.WithBoolean("is_public",
			mcp.Description("Whether the playlist should be public (default: false)"),
		),
	)
	s.mcpServer.AddTool(createPlaylistTool, s.CreatePlaylistHandler)

	addToPlaylistTool := mcp.NewTool("add_to_playlist",
		mcp.WithDescription("Add content to a playlist"),
		mcp.WithString("playlist_id",
			mcp.Required(),
			mcp.Description("ID of the playlist to add content to"),
		),
		mcp.WithString("content_id",
			mcp.Required(),
			mcp.Description("ID or OURN of the content to add"),
		),
	)
	s.mcpServer.AddTool(addToPlaylistTool, s.AddToPlaylistHandler)

	getPlaylistDetailsTool := mcp.NewTool("get_playlist_details",
		mcp.WithDescription("Get detailed information about a specific playlist"),
		mcp.WithString("playlist_id",
			mcp.Required(),
			mcp.Description("ID of the playlist to get details for"),
		),
	)
	s.mcpServer.AddTool(getPlaylistDetailsTool, s.GetPlaylistDetailsHandler)

	// 目次抽出ツールの追加
	extractTocTool := mcp.NewTool("extract_table_of_contents",
		mcp.WithDescription("Extract table of contents from O'Reilly book URL"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("O'Reilly book URL (e.g., https://learning.oreilly.com/library/view/docker-deep-dive/9781806024032/chap04.xhtml)"),
		),
	)
	s.mcpServer.AddTool(extractTocTool, s.ExtractTableOfContentsHandler)

	// 書籍内検索ツールの追加
	searchInBookTool := mcp.NewTool("search_in_book",
		mcp.WithDescription("Search for terms within a specific O'Reilly book"),
		mcp.WithString("book_id",
			mcp.Required(),
			mcp.Description("Book ID or ISBN (e.g., 9784814400607)"),
		),
		mcp.WithString("search_term",
			mcp.Required(),
			mcp.Description("Term or phrase to search for within the book"),
		),
	)
	s.mcpServer.AddTool(searchInBookTool, s.SearchInBookHandler)

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

// ListPlaylistsHandler はプレイリスト一覧取得リクエストを処理します
func (s *Server) ListPlaylistsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("プレイリスト一覧取得リクエスト受信: %+v", request)

	// ブラウザクライアントからプレイリストページのプレイリストを取得
	var playlists []map[string]interface{}
	if s.oreillyClient.browserClient != nil {
		log.Printf("ブラウザクライアントからプレイリストページのプレイリストを取得します")
		playlistResults, err := s.oreillyClient.browserClient.GetPlaylistsFromPlaylistsPage()
		if err != nil {
			log.Printf("プレイリストページからの取得に失敗: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to get playlists: %v", err)), nil
		} else {
			playlists = playlistResults
			log.Printf("プレイリストページから%d個のプレイリストを取得しました", len(playlistResults))
		}
	} else {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	// 結果をレスポンスに変換
	response := struct {
		Count   int           `json:"count"`
		Results []interface{} `json:"results"`
		Source  string        `json:"source"`
	}{
		Count:   len(playlists),
		Results: make([]interface{}, 0, len(playlists)),
		Source:  "playlists_page",
	}

	for _, playlist := range playlists {
		response.Results = append(response.Results, playlist)
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// CreatePlaylistHandler はプレイリスト作成リクエストを処理します
func (s *Server) CreatePlaylistHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("プレイリスト作成リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		IsPublic    bool   `json:"is_public,omitempty"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.Name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	// ブラウザクライアントでプレイリスト作成を実行
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	log.Printf("ブラウザクライアント呼び出し前")
	result, err := s.oreillyClient.browserClient.CreatePlaylist(requestParams.Name, requestParams.Description, requestParams.IsPublic)
	if err != nil {
		log.Printf("ブラウザクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to create playlist: %v", err)), nil
	}
	log.Printf("ブラウザクライアント呼び出し後: %v", result)

	// 結果をレスポンスに変換
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("プレイリスト「%s」を正常に作成しました", requestParams.Name),
		"playlist": result,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// AddToPlaylistHandler はプレイリストへのコンテンツ追加リクエストを処理します
func (s *Server) AddToPlaylistHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("プレイリストへのコンテンツ追加リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		PlaylistID string `json:"playlist_id"`
		ContentID  string `json:"content_id"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.PlaylistID == "" {
		return mcp.NewToolResultError("playlist_id parameter is required"), nil
	}
	if requestParams.ContentID == "" {
		return mcp.NewToolResultError("content_id parameter is required"), nil
	}

	// ブラウザクライアントでコンテンツ追加を実行
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	log.Printf("ブラウザクライアント呼び出し前")
	err = s.oreillyClient.browserClient.AddContentToPlaylist(requestParams.PlaylistID, requestParams.ContentID)
	if err != nil {
		log.Printf("ブラウザクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to add content to playlist: %v", err)), nil
	}
	log.Printf("ブラウザクライアント呼び出し後: 成功")

	// 結果をレスポンスに変換
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("コンテンツ「%s」をプレイリスト「%s」に正常に追加しました", requestParams.ContentID, requestParams.PlaylistID),
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetPlaylistDetailsHandler はプレイリスト詳細取得リクエストを処理します
func (s *Server) GetPlaylistDetailsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("プレイリスト詳細取得リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		PlaylistID string `json:"playlist_id"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.PlaylistID == "" {
		return mcp.NewToolResultError("playlist_id parameter is required"), nil
	}

	// ブラウザクライアントでプレイリスト詳細取得を実行
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	log.Printf("ブラウザクライアント呼び出し前")
	result, err := s.oreillyClient.browserClient.GetPlaylistDetails(requestParams.PlaylistID)
	if err != nil {
		log.Printf("ブラウザクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get playlist details: %v", err)), nil
	}
	log.Printf("ブラウザクライアント呼び出し後: %v", result)

	// 結果をレスポンスに変換
	response := map[string]interface{}{
		"playlist": result,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
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

// ExtractTableOfContentsHandler は目次抽出リクエストを処理します
func (s *Server) ExtractTableOfContentsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("目次抽出リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		URL string `json:"url"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.URL == "" {
		return mcp.NewToolResultError("url parameter is required"), nil
	}

	// ExtractTableOfContentsParamsに変換
	extractParams := browser.ExtractTableOfContentsParams{
		URL: requestParams.URL,
	}

	// O'Reilly クライアントで目次抽出を実行
	log.Printf("O'Reillyクライアント呼び出し前")
	result, err := s.oreillyClient.ExtractTableOfContents(ctx, extractParams)
	if err != nil {
		log.Printf("O'Reillyクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to extract table of contents: %v", err)), nil
	}
	log.Printf("O'Reillyクライアント呼び出し後: %s (%d項目)", result.BookTitle, len(result.Items))

	// 結果をレスポンスに変換
	response := map[string]interface{}{
		"success":           true,
		"book_title":        result.BookTitle,
		"book_id":           result.BookID,
		"book_url":          requestParams.URL, // Use original URL since BookURL is not in new struct
		"authors":           []string{},        // Not available in new struct, use empty array
		"publisher":         "",                // Not available in new struct
		"table_of_contents": result.Items,
		"extracted_at":      "",                // Not available in new struct
		"total_items":       result.TotalItems,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// SearchInBookHandler は書籍内検索リクエストを処理します
func (s *Server) SearchInBookHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("書籍内検索リクエスト受信: %+v", request)

	// リクエストパラメータの取得
	var requestParams struct {
		BookID     string `json:"book_id"`
		SearchTerm string `json:"search_term"`
	}
	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.BookID == "" {
		return mcp.NewToolResultError("book_id parameter is required"), nil
	}
	if requestParams.SearchTerm == "" {
		return mcp.NewToolResultError("search_term parameter is required"), nil
	}

	// ブラウザクライアントで書籍内検索を実行
	if s.oreillyClient.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	log.Printf("ブラウザクライアント呼び出し前")
	results, err := s.oreillyClient.browserClient.SearchInBook(requestParams.BookID, requestParams.SearchTerm)
	if err != nil {
		log.Printf("ブラウザクライアント失敗: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to search in book: %v", err)), nil
	}
	log.Printf("ブラウザクライアント呼び出し後: %d件の結果", len(results))

	// 結果をレスポンスに変換
	response := map[string]interface{}{
		"success":     true,
		"book_id":     requestParams.BookID,
		"search_term": requestParams.SearchTerm,
		"results":     results,
		"total_matches": len(results),
		"message":     fmt.Sprintf("書籍「%s」で「%s」を検索し、%d件の結果が見つかりました", requestParams.BookID, requestParams.SearchTerm, len(results)),
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) ListCollectionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("リクエスト受信: %+v", request)

	// ブラウザクライアントからホームページのコレクションを取得
	var homepageCollections []map[string]interface{}
	if s.oreillyClient.browserClient != nil {
		log.Printf("ブラウザクライアントからホームページのコレクションを取得します")
		collections, err := s.oreillyClient.browserClient.GetCollectionsFromHomePage()
		if err != nil {
			log.Printf("ホームページからのコレクション取得に失敗: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to get collections: %v", err)), nil
		} else {
			homepageCollections = collections
			log.Printf("ホームページから%d個のコレクションを取得しました", len(collections))
		}
	} else {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	// 結果をレスポンスに変換
	response := struct {
		Count   int           `json:"count"`
		Results []interface{} `json:"results"`
		Source  string        `json:"source"`
	}{
		Count:   len(homepageCollections),
		Results: make([]interface{}, 0, len(homepageCollections)),
		Source:  "homepage_only",
	}
	
	for _, collection := range homepageCollections {
		response.Results = append(response.Results, collection)
	}
	
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
