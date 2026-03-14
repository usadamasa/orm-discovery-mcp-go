package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
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
			Description: descResHistRecent,
			MIMEType:    "application/json",
		},
		s.GetRecentHistoryResource,
	)

	// 履歴検索リソーステンプレート
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/search{?keyword,type}",
			Name:        "Search Research History",
			Description: descTmplHistSearch,
			MIMEType:    "application/json",
		},
		s.SearchHistoryResource,
	)

	// 特定の履歴詳細リソーステンプレート
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/{id}",
			Name:        "Research History Detail",
			Description: descTmplHistDetail,
			MIMEType:    "application/json",
		},
		s.GetHistoryDetailResource,
	)

	// フルレスポンスリソーステンプレート
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "orm-mcp://history/{id}/full",
			Name:        "Research History Full Response",
			Description: descTmplHistFull,
			MIMEType:    "application/json",
		},
		s.GetHistoryCachedFileResource,
	)
}

// GetRecentHistoryResource は直近の調査履歴を取得する
func (s *Server) GetRecentHistoryResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Info("直近の調査履歴リソース取得リクエスト受信", "uri", req.Params.URI)

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
	slog.Info("直近の調査履歴取得完了", "count", len(entries))

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
	slog.Info("調査履歴検索リソース取得リクエスト受信", "uri", req.Params.URI)

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
		slog.Info("パラメータなしで直近履歴取得", "count", len(entries))
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
	slog.Info("調査履歴詳細リソース取得リクエスト受信", "uri", req.Params.URI)

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
		slog.Info("調査履歴詳細取得完了", "id", id, "found", false)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "entry not found: %s"}`, id),
			}},
		}, nil
	}
	slog.Info("調査履歴詳細取得完了", "id", id, "found", true)

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

// GetHistoryCachedFileResource は特定の履歴のフルレスポンスを取得する
func (s *Server) GetHistoryCachedFileResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	slog.Info("調査履歴フルレスポンスリソース取得リクエスト受信", "uri", req.Params.URI)

	if s.historyManager == nil {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     `{"error": "research history manager is not available"}`,
			}},
		}, nil
	}

	// URIからIDを抽出（/full サフィックスを考慮）
	id := extractHistoryIDFromFullURI(req.Params.URI)
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
		slog.Info("調査履歴フルレスポンス取得完了", "id", id, "found", false)
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     fmt.Sprintf(`{"error": "entry not found: %s"}`, id),
			}},
		}, nil
	}
	slog.Info("調査履歴フルレスポンス取得完了", "id", id, "found", true, "has_file", entry.FilePath != "")

	// FilePath が設定されている場合、ファイルの内容を読んで返す
	if entry.FilePath != "" {
		data, err := os.ReadFile(entry.FilePath)
		if err != nil {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      req.Params.URI,
					MIMEType: "text/markdown",
					Text:     fmt.Sprintf("Error reading cached file: %v\nFile path: %s", err, entry.FilePath),
				}},
			}, nil
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     string(data),
			}},
		}, nil
	}

	// FilePath が未設定（旧エントリ）の場合
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     `{"error": "Full response data is no longer stored inline. Re-run the search to generate a cached file."}`,
		}},
	}, nil
}

// extractHistoryIDFromFullURI は orm-mcp://history/{id}/full 形式のURIからIDを抽出する
func extractHistoryIDFromFullURI(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	// Path: "/{id}/full"
	p := strings.TrimPrefix(u.Path, "/")
	p = strings.TrimSuffix(p, "/full")
	if p == "" || p == "search" || p == "recent" {
		return ""
	}
	return p
}

// extractHistorySearchParams はURIから検索パラメータを抽出する
func extractHistorySearchParams(uri string) (keyword, entryType string) {
	// orm-mcp://history/search?keyword=xxx&type=yyy の形式
	if uri == "" {
		return
	}
	u, err := url.Parse(uri)
	if err != nil {
		return
	}
	values := u.Query()
	keyword = values.Get("keyword")
	entryType = values.Get("type")
	return
}

// extractHistoryIDFromURI はURIからIDを抽出する
func extractHistoryIDFromURI(uri string) string {
	// orm-mcp://history/{id} の形式
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	id := strings.TrimPrefix(u.Path, "/")
	if id == "" || id == "search" || id == "recent" {
		return ""
	}
	return id
}
