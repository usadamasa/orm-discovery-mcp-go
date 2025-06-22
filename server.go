package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
	log.Printf("HTTPサーバーを起動します :%s/mcp", port)
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
		mcp.WithDescription(`
			Search content on O'Reilly Learning Platform.
			Returns a list of books, videos, and articles with their product IDs, titles, descriptions, authors, and topics.

			Use this as the first step to discover relevant content for specific technologies, programming concepts, or technical challenges.
			Each result includes a product_id that can be used to access detailed information through MCP resources:

			- Book details and TOC: Use resource URI "oreilly://book-details/{product_id}"
			- Table of contents only: Use resource URI "oreilly://book-toc/{product_id}" 
			- Chapter content: Use resource URI "oreilly://book-chapter/{product_id}/{chapter_name}"

			IMPORTANT: When referencing any content found through this search, always cite the source with title, author(s), and O'Reilly Media as the publisher.
		`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for specific technologies, frameworks, concepts, or technical challenges (e.g., 'Docker containers', 'React hooks', 'machine learning algorithms', 'microservices architecture')"),
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

	// Resources are now handled by the resource system instead of tools
	s.registerResources()
}

// registerResources はリソースを登録します
func (s *Server) registerResources() {
	// 書籍詳細リソースの登録
	bookDetailsResource := mcp.NewResource(
		"oreilly://book-details/{product_id}",
		"O'Reilly Book Details",
		mcp.WithResourceDescription("Get comprehensive book information including title, authors, publication date, description, topics, and table of contents. IMPORTANT: Always provide proper attribution when referencing this book information."),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(bookDetailsResource, s.GetBookDetailsResource)

	// 書籍目次リソースの登録
	bookTOCResource := mcp.NewResource(
		"oreilly://book-toc/{product_id}",
		"O'Reilly Book Table of Contents",
		mcp.WithResourceDescription("Get detailed table of contents with chapter names and structure. IMPORTANT: When discussing book structure, always cite the book title, author(s), and O'Reilly Media."),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(bookTOCResource, s.GetBookTOCResource)

	// チャプター本文リソースの登録
	bookChapterResource := mcp.NewResource(
		"oreilly://book-chapter/{product_id}/{chapter_name}",
		"O'Reilly Book Chapter Content",
		mcp.WithResourceDescription("Extract full text content of a specific chapter. CRITICAL: Any content extracted MUST be properly cited with book title, author(s), chapter title, and O'Reilly Media."),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(bookChapterResource, s.GetBookChapterContentResource)

	// Resource Templates for dynamic discovery
	bookDetailsTemplate := mcp.NewResourceTemplate(
		"oreilly://book-details/{product_id}",
		"O'Reilly Book Details Template",
		mcp.WithTemplateDescription("Template for accessing O'Reilly book details. Use product_id from search_content results to get comprehensive book information including title, authors, publication date, description, topics, and table of contents."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(bookDetailsTemplate, s.GetBookDetailsResourceTemplate)

	bookTOCTemplate := mcp.NewResourceTemplate(
		"oreilly://book-toc/{product_id}",
		"O'Reilly Book TOC Template",
		mcp.WithTemplateDescription("Template for accessing O'Reilly book table of contents. Use product_id from search_content results to get detailed chapter structure and navigation information."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(bookTOCTemplate, s.GetBookTOCResourceTemplate)

	bookChapterTemplate := mcp.NewResourceTemplate(
		"oreilly://book-chapter/{product_id}/{chapter_name}",
		"O'Reilly Book Chapter Template",
		mcp.WithTemplateDescription("Template for accessing O'Reilly book chapter content. Use product_id from search_content and chapter_name from table of contents to get full chapter text including headings, paragraphs, and code examples."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(bookChapterTemplate, s.GetBookChapterContentResourceTemplate)
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
	log.Printf("O'Reillyクライアント呼び出し後 取得件数: %d件", results.Count)

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

// GetBookDetailsResource handles book detail resource requests
func (s *Server) GetBookDetailsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	log.Printf("書籍詳細リソース取得リクエスト受信: %+v", request)

	// URIからproduct_idを抽出
	productID := extractProductIDFromURI(request.Params.URI)
	if productID == "" {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id not found in URI"}`,
			},
		}, nil
	}

	// ブラウザクライアントで書籍詳細を取得
	if s.oreillyClient.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	bookOverview, err := s.oreillyClient.browserClient.GetBookDetails(productID)
	if err != nil {
		log.Printf("書籍詳細取得失敗: %v", err)
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book details: %v"}`, err),
			},
		}, nil
	}

	jsonBytes, err := json.Marshal(bookOverview)
	if err != nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			},
		}, nil
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		},
	}, nil
}

// GetBookTOCResource handles book TOC resource requests
func (s *Server) GetBookTOCResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	log.Printf("書籍目次リソース取得リクエスト受信: %+v", request)

	// URIからproduct_idを抽出
	productID := extractProductIDFromURI(request.Params.URI)
	if productID == "" {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id not found in URI"}`,
			},
		}, nil
	}

	// ブラウザクライアントの確認
	if s.oreillyClient.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	tocResponse, err := s.oreillyClient.browserClient.GetBookTOC(productID)
	if err != nil {
		log.Printf("書籍目次取得失敗: %v", err)
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book TOC: %v"}`, err),
			},
		}, nil
	}

	jsonBytes, err := json.Marshal(tocResponse)
	if err != nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			},
		}, nil
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		},
	}, nil
}

// GetBookChapterContentResource handles book chapter content resource requests
func (s *Server) GetBookChapterContentResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	log.Printf("書籍チャプター本文リソース取得リクエスト受信: %+v", request)

	// URIからproduct_idとchapter_nameを抽出
	productID, chapterName := extractProductIDAndChapterFromURI(request.Params.URI)
	if productID == "" || chapterName == "" {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "product_id or chapter_name not found in URI"}`,
			},
		}, nil
	}

	// ブラウザクライアントの確認
	if s.oreillyClient.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	chapterResponse, err := s.oreillyClient.browserClient.GetBookChapterContent(productID, chapterName)
	if err != nil {
		log.Printf("書籍チャプター本文取得失敗: %v", err)
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get book chapter content: %v"}`, err),
			},
		}, nil
	}

	jsonBytes, err := json.Marshal(chapterResponse)
	if err != nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			},
		}, nil
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		},
	}, nil
}

// Resource Template Handlers
// GetBookDetailsResourceTemplate handles resource template requests for book details
func (s *Server) GetBookDetailsResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Template handlers should return the actual resource content when given a valid URI
	// For templates, we delegate to the actual resource handler
	return s.GetBookDetailsResource(ctx, request)
}

// GetBookTOCResourceTemplate handles resource template requests for book TOC
func (s *Server) GetBookTOCResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Template handlers should return the actual resource content when given a valid URI
	// For templates, we delegate to the actual resource handler
	return s.GetBookTOCResource(ctx, request)
}

// GetBookChapterContentResourceTemplate handles resource template requests for book chapter content
func (s *Server) GetBookChapterContentResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Template handlers should return the actual resource content when given a valid URI
	// For templates, we delegate to the actual resource handler
	return s.GetBookChapterContentResource(ctx, request)
}

// Helper functions to extract parameters from URIs
func extractProductIDFromURI(uri string) string {
	// Extract product_id from URIs like "oreilly://book-details/{product_id}" or "oreilly://book-toc/{product_id}"
	parts := strings.Split(uri, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractProductIDAndChapterFromURI(uri string) (string, string) {
	// Extract product_id and chapter_name from URIs like "oreilly://book-chapter/{product_id}/{chapter_name}"
	parts := strings.Split(uri, "/")
	if len(parts) >= 4 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", ""
}
