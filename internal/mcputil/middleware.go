package mcputil

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MiddlewareFactory creates MCP middleware with a configured log level.
type MiddlewareFactory struct {
	LogLevel slog.Level
}

// Logging creates middleware for logging MCP requests and responses.
func (mf MiddlewareFactory) Logging() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()

			// Log request
			if mf.LogLevel <= slog.LevelDebug {
				reqJSON, _ := json.Marshal(req)
				slog.Debug("MCP受信",
					"method", method,
					"payload", string(reqJSON))
			} else {
				slog.Info("MCPリクエスト開始",
					"method", method)
			}

			// Call the next handler
			result, err := next(ctx, method, req)

			duration := time.Since(start)

			// Log response
			if err != nil {
				slog.Error("MCPリクエスト失敗",
					"method", method,
					"duration", duration,
					"error", err.Error())
			} else {
				if mf.LogLevel <= slog.LevelDebug {
					resultJSON, _ := json.Marshal(result)
					resultSize := len(resultJSON)
					if resultSize > 1000 {
						slog.Debug("MCP成功",
							"method", method,
							"duration", duration,
							"result_size", resultSize,
							"result_preview", string(resultJSON[:500])+"...")
					} else {
						slog.Debug("MCP成功",
							"method", method,
							"duration", duration,
							"result", string(resultJSON))
					}
				} else {
					slog.Info("MCPリクエスト成功",
						"method", method,
						"duration", duration)
				}
			}

			return result, err
		}
	}
}

// MCP method constants used in middleware routing.
const (
	mcpMethodToolsCall     = "tools/call"
	mcpMethodResourcesRead = "resources/read"
)

// ToolLogging creates middleware for logging tool calls and resource reads.
func (mf MiddlewareFactory) ToolLogging() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// Only log for tool calls
			if method == mcpMethodToolsCall {
				if mf.LogLevel <= slog.LevelDebug {
					reqJSON, _ := json.Marshal(req)
					slog.Debug("ツール呼び出し開始",
						"method", method,
						"request", string(reqJSON))
				} else {
					slog.Info("ツール呼び出し",
						"method", method)
				}
			}

			// Only log for resource reads
			if method == mcpMethodResourcesRead {
				slog.Debug("リソース読み込み開始",
					"method", method)
			}

			return next(ctx, method, req)
		}
	}
}
