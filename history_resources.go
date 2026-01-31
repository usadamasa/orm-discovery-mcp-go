package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerHistoryResources は履歴リソースを登録する
func (s *Server) registerHistoryResources() {
	// 直近の調査履歴リソース
	s.server.AddResource(
		&mcp.Resource{
			URI:         "orm-mcp://history/recent",
			Name:        "Recent Research History",
			Description: "Get recent 20 research entries. Use to review past searches and questions.",
			MIMEType:    "application/json",
		},
		s.GetRecentHistoryResource,
	)

	// 履歴検索リソーステンプレート
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/search{?keyword,type}",
			Name:        "Search Research History",
			Description: "Search past research by keyword or type (search/question).",
			MIMEType:    "application/json",
		},
		s.SearchHistoryResource,
	)

	// 特定の履歴詳細リソーステンプレート
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/{id}",
			Name:        "Research History Detail",
			Description: "Get details of a specific research entry by ID.",
			MIMEType:    "application/json",
		},
		s.GetHistoryDetailResource,
	)
}

// GetRecentHistoryResource は直近の調査履歴を取得する
func (s *Server) GetRecentHistoryResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("直近の調査履歴リソース取得リクエスト受信", "uri", req.Params.URI)

	if s.historyManager == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "research history manager is not available"}`,
			}},
		}, nil
	}

	entries := s.historyManager.GetRecent(20)

	response := struct {
		Count   int             `json:"count"`
		Entries []ResearchEntry `json:"entries"`
	}{
		Count:   len(entries),
		Entries: entries,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// SearchHistoryResource は履歴を検索する
func (s *Server) SearchHistoryResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("調査履歴検索リソース取得リクエスト受信", "uri", req.Params.URI)

	if s.historyManager == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "research history manager is not available"}`,
			}},
		}, nil
	}

	// URIからクエリパラメータを抽出
	keyword, entryType := extractHistorySearchParams(req.Params.URI)

	var entries []ResearchEntry

	if keyword != "" {
		entries = s.historyManager.SearchByKeyword(keyword)
		slog.Info("キーワードで履歴検索", "keyword", keyword, "results", len(entries))
	} else if entryType != "" {
		entries = s.historyManager.SearchByType(entryType)
		slog.Info("タイプで履歴検索", "type", entryType, "results", len(entries))
	} else {
		// パラメータがない場合は直近20件を返す
		entries = s.historyManager.GetRecent(20)
	}

	response := struct {
		Keyword string          `json:"keyword,omitempty"`
		Type    string          `json:"type,omitempty"`
		Count   int             `json:"count"`
		Entries []ResearchEntry `json:"entries"`
	}{
		Keyword: keyword,
		Type:    entryType,
		Count:   len(entries),
		Entries: entries,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// GetHistoryDetailResource は特定の履歴詳細を取得する
func (s *Server) GetHistoryDetailResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Debug("調査履歴詳細リソース取得リクエスト受信", "uri", req.Params.URI)

	if s.historyManager == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "research history manager is not available"}`,
			}},
		}, nil
	}

	// URIからIDを抽出
	id := extractHistoryIDFromURI(req.Params.URI)
	if id == "" {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "id not found in URI"}`,
			}},
		}, nil
	}

	entry := s.historyManager.GetByID(id)
	if entry == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "entry not found: %s"}`, id),
			}},
		}, nil
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "failed to marshal response: %v"}`, err),
			}},
		}, nil
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonBytes),
		}},
	}, nil
}

// extractHistorySearchParams はURIから検索パラメータを抽出する
func extractHistorySearchParams(uri string) (keyword, entryType string) {
	// orm-mcp://history/search?keyword=xxx&type=yyy の形式
	if idx := strings.Index(uri, "?"); idx != -1 {
		queryStr := uri[idx+1:]
		values, err := url.ParseQuery(queryStr)
		if err == nil {
			keyword = values.Get("keyword")
			entryType = values.Get("type")
		}
	}
	return
}

// extractHistoryIDFromURI はURIからIDを抽出する
func extractHistoryIDFromURI(uri string) string {
	// orm-mcp://history/{id} の形式
	// クエリパラメータがある場合は除去
	if idx := strings.Index(uri, "?"); idx != -1 {
		uri = uri[:idx]
	}

	parts := strings.Split(uri, "/")
	if len(parts) >= 4 {
		lastPart := parts[len(parts)-1]
		// "search" の場合はIDではない
		if lastPart != "search" && lastPart != "recent" {
			return lastPart
		}
	}
	return ""
}
