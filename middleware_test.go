package main

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNextHandler creates a MethodHandler that records calls and returns the given result/error.
func mockNextHandler(result mcp.Result, err error) (mcp.MethodHandler, *int) {
	callCount := 0
	handler := func(_ context.Context, _ string, req mcp.Request) (mcp.Result, error) {
		callCount++
		return result, err
	}
	return handler, &callCount
}

func TestLoggingMiddleware_PassesThrough(t *testing.T) {
	middleware := createLoggingMiddleware(slog.LevelInfo)
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestLoggingMiddleware_LogsOnError(t *testing.T) {
	middleware := createLoggingMiddleware(slog.LevelInfo)
	expectedErr := errors.New("test error")
	next, callCount := mockNextHandler(nil, expectedErr)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, *callCount)
}

func TestLoggingMiddleware_DebugLevel(t *testing.T) {
	middleware := createLoggingMiddleware(slog.LevelDebug)
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestToolLoggingMiddleware_ToolsCall(t *testing.T) {
	middleware := createToolLoggingMiddleware(slog.LevelInfo)
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), mcpMethodToolsCall, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestToolLoggingMiddleware_ResourcesRead(t *testing.T) {
	middleware := createToolLoggingMiddleware(slog.LevelInfo)
	expectedResult := &mcp.ReadResourceResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), mcpMethodResourcesRead, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestToolLoggingMiddleware_OtherMethod(t *testing.T) {
	middleware := createToolLoggingMiddleware(slog.LevelInfo)
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := middleware(next)
	result, err := wrapped(context.Background(), "prompts/get", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestToolLoggingMiddleware_PropagatesError(t *testing.T) {
	middleware := createToolLoggingMiddleware(slog.LevelInfo)
	expectedErr := errors.New("tool error")
	next, _ := mockNextHandler(nil, expectedErr)

	wrapped := middleware(next)
	_, err := wrapped(context.Background(), mcpMethodToolsCall, nil)

	assert.ErrorIs(t, err, expectedErr)
}
