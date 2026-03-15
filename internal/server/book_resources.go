package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/browser"
	"github.com/usadamasa/orm-discovery-mcp-go/internal/mcputil"
)

// resourceDef describes a resource and its optional template registration.
type resourceDef struct {
	uri, name, desc, mimeType string
	handler                   func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
	tmplDesc                  string // if non-empty, also register as ResourceTemplate
}

// registerResources registers the resource handlers using a data-driven table.
func (s *Server) registerResources() {
	resources := []resourceDef{
		{uri: "oreilly://book-details/{product_id}", name: "O'Reilly Book Details", desc: descResBookDetails, mimeType: "application/json", handler: s.GetBookDetailsResource, tmplDesc: descTmplBookDetails},
		{uri: "oreilly://book-toc/{product_id}", name: "O'Reilly Book Table of Contents", desc: descResBookTOC, mimeType: "application/json", handler: s.GetBookTOCResource, tmplDesc: descTmplBookTOC},
		{uri: "oreilly://book-chapter/{product_id}/{chapter_name}", name: "O'Reilly Book Chapter Content", desc: descResBookChapter, mimeType: "application/json", handler: s.GetBookChapterContentResource, tmplDesc: descTmplBookChapter},
		{uri: "oreilly://answer/{question_id}", name: "O'Reilly Answers Response", desc: descResAnswer, mimeType: "application/json", handler: s.GetAnswerResource, tmplDesc: descTmplAnswer},
		{uri: "orm-mcp://server/status", name: "MCP Server Status", desc: "Server startup time and version for restart verification", mimeType: "application/json", handler: s.GetServerStatusResource},
	}

	for _, r := range resources {
		s.server.AddResource(&mcp.Resource{URI: r.uri, Name: r.name, Description: r.desc, MIMEType: r.mimeType}, r.handler)
		if r.tmplDesc != "" {
			s.server.AddResourceTemplate(&mcp.ResourceTemplate{URITemplate: r.uri, Name: r.name + " Template", Description: r.tmplDesc, MIMEType: r.mimeType}, r.handler)
		}
	}
}

// readResourceJSON is a generic helper for resource handlers that fetch data and return JSON.
func (s *Server) readResourceJSON(uri string, fetch func() (any, error), opName string, kvs ...any) (*mcp.ReadResourceResult, error) {
	if s.getBrowserClient() == nil {
		return clientUnavailableResult(uri), nil
	}
	data, err := fetch()
	if err != nil {
		return errH.ResourceContents(uri, err, append([]any{"operation", opName}, kvs...)...), nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return errH.ResourceContents(uri, err, "operation", "marshal_"+opName), nil
	}
	return jsonResourceResult(uri, jsonBytes), nil
}

func clientUnavailableResult(uri string) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     `{"error": "browser client is not available"}`,
		}},
	}
}

func jsonResourceResult(uri string, jsonBytes []byte) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}
}

func paramErrorResult(uri, msg string) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     fmt.Sprintf(`{"error": %q}`, msg),
		}},
	}
}

// GetBookDetailsResource handles book detail resource requests.
func (s *Server) GetBookDetailsResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	productID := mcputil.ExtractProductIDFromURI(req.Params.URI)
	if productID == "" {
		return paramErrorResult(req.Params.URI, "product_id not found in URI"), nil
	}
	return s.readResourceJSON(req.Params.URI, func() (any, error) {
		return s.getBrowserClient().GetBookDetails(productID)
	}, "get_book_details", "product_id", productID)
}

// GetBookTOCResource handles book TOC resource requests.
func (s *Server) GetBookTOCResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	productID := mcputil.ExtractProductIDFromURI(req.Params.URI)
	if productID == "" {
		return paramErrorResult(req.Params.URI, "product_id not found in URI"), nil
	}
	return s.readResourceJSON(req.Params.URI, func() (any, error) {
		return s.getBrowserClient().GetBookTOC(productID)
	}, "get_book_toc", "product_id", productID)
}

// GetBookChapterContentResource handles book chapter content resource requests.
func (s *Server) GetBookChapterContentResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	productID, chapterName := mcputil.ExtractProductIDAndChapterFromURI(req.Params.URI)
	if productID == "" || chapterName == "" {
		return paramErrorResult(req.Params.URI, "product_id or chapter_name not found in URI"), nil
	}
	return s.readResourceJSON(req.Params.URI, func() (any, error) {
		return s.getBrowserClient().GetBookChapterContent(productID, chapterName)
	}, "get_chapter", "product_id", productID, "chapter_name", chapterName)
}

// GetAnswerResource handles answer resource requests.
func (s *Server) GetAnswerResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	questionID := mcputil.ExtractQuestionIDFromURI(req.Params.URI)
	if questionID == "" {
		return paramErrorResult(req.Params.URI, "question_id not found in URI"), nil
	}
	return s.readResourceJSON(req.Params.URI, func() (any, error) {
		answer, err := s.getBrowserClient().GetQuestionByID(questionID)
		if err != nil {
			return nil, err
		}
		return struct {
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
		}, nil
	}, "get_answer", "question_id", questionID)
}

// GetServerStatusResource returns server startup time and version for restart verification.
func (s *Server) GetServerStatusResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	status := map[string]string{
		"started_at": s.startedAt.UTC().Format(time.RFC3339),
		"version":    s.serverVersion,
	}
	jsonBytes, _ := json.Marshal(status)
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}
