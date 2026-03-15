package mcputil

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

func TestMiddlewareFactory_Logging_PassesThrough(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := mf.Logging()(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_Logging_LogsOnError(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedErr := errors.New("test error")
	next, callCount := mockNextHandler(nil, expectedErr)

	wrapped := mf.Logging()(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_Logging_DebugLevel(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelDebug}
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := mf.Logging()(next)
	result, err := wrapped(context.Background(), "tools/call", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_ToolLogging_ToolsCall(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := mf.ToolLogging()(next)
	result, err := wrapped(context.Background(), mcpMethodToolsCall, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_ToolLogging_ResourcesRead(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedResult := &mcp.ReadResourceResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := mf.ToolLogging()(next)
	result, err := wrapped(context.Background(), mcpMethodResourcesRead, nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_ToolLogging_OtherMethod(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedResult := &mcp.CallToolResult{}
	next, callCount := mockNextHandler(expectedResult, nil)

	wrapped := mf.ToolLogging()(next)
	result, err := wrapped(context.Background(), "prompts/get", nil)

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
	assert.Equal(t, 1, *callCount)
}

func TestMiddlewareFactory_ToolLogging_PropagatesError(t *testing.T) {
	mf := MiddlewareFactory{LogLevel: slog.LevelInfo}
	expectedErr := errors.New("tool error")
	next, _ := mockNextHandler(nil, expectedErr)

	wrapped := mf.ToolLogging()(next)
	_, err := wrapped(context.Background(), mcpMethodToolsCall, nil)

	assert.ErrorIs(t, err, expectedErr)
}
