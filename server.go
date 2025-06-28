package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/usadamasa/orm-discovery-mcp-go/browser"
)

// Server はMCPサーバーの実装です
type Server struct {
	browserClient *browser.BrowserClient
	mcpServer     *server.MCPServer
}

// NewServer は新しいサーバーインスタンスを作成します
func NewServer(browserClient *browser.BrowserClient) *Server {
	// MCPサーバーの設定とデバッグログの追加
	mcpServer := server.NewMCPServer(
		"Search O'Reilly Learning Platform",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	srv := &Server{
		browserClient: browserClient,
		mcpServer:     mcpServer,
	}
	// 初期化処理の成功を確認するためのログ
	slog.Info("サーバーを初期化しました")

	srv.registerHandlers()
	slog.Info("ハンドラーを登録しました")

	return srv
}

// StartStreamableHTTPServer はMCPサーバを返します
func (s *Server) StartStreamableHTTPServer(port string) error {
	slog.Info("HTTPサーバーを起動します", "endpoint", port+"/mcp")
	// タイムアウト設定を調整したサーバーを作成
	httpServer := server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)
	slog.Info("HTTPサーバーを作成しました")
	err := httpServer.Start(port)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) StartStdioServer() error {
	// MCPサーバーを標準入出力で起動
	slog.Info("MCPサーバーを標準入出力で起動します")
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	return nil
}

// registerHandlers はハンドラーを登録します
func (s *Server) registerHandlers() {
	// Add search tool
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

	// Add ask question tool
	askQuestionTool := mcp.NewTool("ask_question",
		mcp.WithDescription(`
			Ask technical questions to O'Reilly Answers AI and receive comprehensive, well-sourced responses.
			
			This tool leverages O'Reilly's AI-powered question answering service, which draws from O'Reilly's extensive 
			library of technical books, videos, and articles to provide detailed, accurate answers. 
			
			Response includes:
			- Comprehensive markdown-formatted answer
			- Source citations with specific book/article references
			- Related resources for deeper learning
			- Suggested follow-up questions for exploration
			- Question ID for future reference
			
			The AI searches across programming, data science, cloud computing, DevOps, machine learning, 
			and other technical domains covered in O'Reilly's content library.

			IMPORTANT: Always cite the sources provided in the response when referencing the information.
		`),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("Natural language question about technical topics, programming, data science, cloud computing, etc. (e.g., 'How do I build a data lake on S3?', 'What are the best practices for React performance optimization?')"),
		),
		mcp.WithNumber("max_wait_time_seconds",
			mcp.Description("Maximum time to wait for answer generation in seconds (default: 300, max: 600)"),
		),
	)
	s.mcpServer.AddTool(askQuestionTool, s.AskQuestionHandler)

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

	// 回答リソースの登録
	answerResource := mcp.NewResource(
		"oreilly://answer/{question_id}",
		"O'Reilly Answers Response",
		mcp.WithResourceDescription("Access a previously generated answer by question ID. This retrieves the full AI-generated response including answer content, sources, and related resources. IMPORTANT: Always cite the sources provided when referencing the answer content."),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(answerResource, s.GetAnswerResource)

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

	answerTemplate := mcp.NewResourceTemplate(
		"oreilly://answer/{question_id}",
		"O'Reilly Answers Template",
		mcp.WithTemplateDescription("Template for accessing O'Reilly Answers responses. Use question_id returned from ask_question tool to retrieve the complete AI-generated answer with sources and related resources."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(answerTemplate, s.GetAnswerResourceTemplate)
}

// SearchContentHandler は検索リクエストを処理します
func (s *Server) SearchContentHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slog.Debug("検索リクエスト受信")

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

	// デフォルト値の設定
	if requestParams.Rows <= 0 {
		requestParams.Rows = 100
	}
	if len(requestParams.Languages) == 0 {
		requestParams.Languages = []string{"en", "ja"}
	}

	// BrowserClient用のオプションを準備
	options := map[string]interface{}{
		"rows":          requestParams.Rows,
		"languages":     requestParams.Languages,
		"tzOffset":      requestParams.TzOffset,
		"aia_only":      requestParams.AiaOnly,
		"feature_flags": requestParams.FeatureFlags,
		"report":        requestParams.Report,
		"isTopics":      requestParams.IsTopics,
	}

	// BrowserClientで直接検索を実行
	slog.Debug("BrowserClient検索開始", "query", requestParams.Query)
	results, err := s.browserClient.SearchContent(requestParams.Query, options)
	if err != nil {
		slog.Error("BrowserClient検索失敗", "error", err, "query", requestParams.Query)
		return mcp.NewToolResultError(fmt.Sprintf("failed to search O'Reilly: %v", err)), nil
	}
	slog.Info("検索完了", "query", requestParams.Query, "result_count", len(results))

	// 結果をレスポンスに変換
	response := struct {
		Count   int           `json:"count"`
		Total   int           `json:"total"`
		Results []interface{} `json:"results"`
	}{
		Count:   len(results),
		Total:   len(results),
		Results: make([]interface{}, len(results)),
	}

	// resultsを[]interface{}に変換
	for i, result := range results {
		response.Results[i] = result
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}
	// レスポンスを返す
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// AskQuestionHandler processes question requests for O'Reilly Answers
func (s *Server) AskQuestionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slog.Debug("質問リクエスト受信")

	// リクエストパラメータの取得
	var requestParams struct {
		Question           string `json:"question"`
		MaxWaitTimeSeconds int    `json:"max_wait_time_seconds,omitempty"`
	}

	argumentsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments"), nil
	}
	if err := json.Unmarshal(argumentsBytes, &requestParams); err != nil {
		return mcp.NewToolResultError("invalid parameters"), nil
	}

	if requestParams.Question == "" {
		return mcp.NewToolResultError("question parameter is required"), nil
	}

	// デフォルトのタイムアウト設定（5分）
	maxWaitTime := 300 * time.Second
	if requestParams.MaxWaitTimeSeconds > 0 {
		if requestParams.MaxWaitTimeSeconds > 600 { // 最大10分
			requestParams.MaxWaitTimeSeconds = 600
		}
		maxWaitTime = time.Duration(requestParams.MaxWaitTimeSeconds) * time.Second
	}

	// ブラウザクライアントの確認
	if s.browserClient == nil {
		return mcp.NewToolResultError("browser client is not available"), nil
	}

	slog.Info("質問処理開始", "question", requestParams.Question, "max_wait_time", maxWaitTime)

	// 質問を実行（ポーリング付き）
	answer, err := s.browserClient.AskQuestion(requestParams.Question, maxWaitTime)
	if err != nil {
		slog.Error("質問処理失敗", "error", err, "question", requestParams.Question)
		return mcp.NewToolResultError(fmt.Sprintf("failed to ask question: %v", err)), nil
	}

	slog.Info("質問に対する回答を取得しました", "question", requestParams.Question, "question_id", answer.QuestionID)

	// レスポンスの構築
	response := struct {
		QuestionID          string                       `json:"question_id"`
		Question            string                       `json:"question"`
		Answer              string                       `json:"answer"`
		IsFinished          bool                         `json:"is_finished"`
		Sources             []browser.AnswerSource       `json:"sources"`
		RelatedResources    []browser.RelatedResource    `json:"related_resources"`
		AffiliationProducts []browser.AffiliationProduct `json:"affiliation_products"`
		FollowupQuestions   []string                     `json:"followup_questions"`
		CitationNote        string                       `json:"citation_note"`
	}{
		QuestionID:          answer.QuestionID,
		Question:            requestParams.Question,
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
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// GetBookDetailsResource handles book detail resource requests
func (s *Server) GetBookDetailsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	slog.Debug("書籍詳細リソース取得リクエスト受信", "uri", request.Params.URI)

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
	if s.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	bookOverview, err := s.browserClient.GetBookDetails(productID)
	if err != nil {
		slog.Error("書籍詳細取得失敗", "error", err, "product_id", productID)
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
	slog.Debug("書籍目次リソース取得リクエスト受信", "uri", request.Params.URI)

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
	if s.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	tocResponse, err := s.browserClient.GetBookTOC(productID)
	if err != nil {
		slog.Error("書籍目次取得失敗", "error", err, "product_id", productID)
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
	slog.Debug("書籍チャプター本文リソース取得リクエスト受信", "uri", request.Params.URI)

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
	if s.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	chapterResponse, err := s.browserClient.GetBookChapterContent(productID, chapterName)
	if err != nil {
		slog.Error("書籍チャプター本文取得失敗", "error", err, "product_id", productID, "chapter_name", chapterName)
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

// GetAnswerResource handles answer resource requests
func (s *Server) GetAnswerResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	slog.Debug("回答リソース取得リクエスト受信", "uri", request.Params.URI)

	// URIからquestion_idを抽出
	questionID := extractQuestionIDFromURI(request.Params.URI)
	if questionID == "" {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "question_id not found in URI"}`,
			},
		}, nil
	}

	// ブラウザクライアントの確認
	if s.browserClient == nil {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "browser client is not available"}`,
			},
		}, nil
	}

	// 回答を取得
	answer, err := s.browserClient.GetQuestionByID(questionID)
	if err != nil {
		slog.Error("回答取得失敗", "error", err, "question_id", questionID)
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to get answer: %v"}`, err),
			},
		}, nil
	}

	// レスポンスの構築
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

// GetAnswerResourceTemplate handles resource template requests for answers
func (s *Server) GetAnswerResourceTemplate(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Template handlers should return the actual resource content when given a valid URI
	// For templates, we delegate to the actual resource handler
	return s.GetAnswerResource(ctx, request)
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

func extractQuestionIDFromURI(uri string) string {
	// Extract question_id from URIs like "oreilly://answer/{question_id}"
	parts := strings.Split(uri, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return ""
}
