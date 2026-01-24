package e2e

import (
	"encoding/json"
	"fmt"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolCallParams represents parameters for tools/call method.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ResourceReadParams represents parameters for resources/read method.
type ResourceReadParams struct {
	URI string `json:"uri"`
}

// ToolCallResult represents the result of a tools/call method.
type ToolCallResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents content in a tool result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ResourceReadResult represents the result of a resources/read method.
type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents content in a resource result.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// NewToolCallRequest creates a JSON-RPC request for tools/call.
func NewToolCallRequest(id int, toolName string, args map[string]interface{}) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "tools/call",
		Params: ToolCallParams{
			Name:      toolName,
			Arguments: args,
		},
	}
}

// NewResourceReadRequest creates a JSON-RPC request for resources/read.
func NewResourceReadRequest(id int, uri string) *JSONRPCRequest {
	return &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "resources/read",
		Params: ResourceReadParams{
			URI: uri,
		},
	}
}

// MarshalRequest marshals a JSON-RPC request to JSON bytes.
func MarshalRequest(req *JSONRPCRequest) ([]byte, error) {
	return json.Marshal(req)
}

// UnmarshalResponse unmarshals a JSON-RPC response from JSON bytes.
func UnmarshalResponse(data []byte) (*JSONRPCResponse, error) {
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &resp, nil
}

// ParseToolCallResult parses the result of a tools/call response.
func ParseToolCallResult(resp *JSONRPCResponse) (*ToolCallResult, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}
	return &result, nil
}

// ParseResourceReadResult parses the result of a resources/read response.
func ParseResourceReadResult(resp *JSONRPCResponse) (*ResourceReadResult, error) {
	if resp.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	var result ResourceReadResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse resource result: %w", err)
	}
	return &result, nil
}

// ValidateJSONRPC20Response validates that a response follows JSON-RPC 2.0 format.
func ValidateJSONRPC20Response(resp *JSONRPCResponse) error {
	if resp.JSONRPC != "2.0" {
		return fmt.Errorf("expected jsonrpc version 2.0, got %s", resp.JSONRPC)
	}
	if resp.Error == nil && resp.Result == nil {
		return fmt.Errorf("response must have either result or error")
	}
	return nil
}
