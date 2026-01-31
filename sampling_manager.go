package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP Role constants
const (
	RoleUser      mcp.Role = "user"
	RoleAssistant mcp.Role = "assistant"
)

// SamplingManager handles MCP Sampling requests to generate summaries.
type SamplingManager struct {
	config *Config
}

// NewSamplingManager creates a new SamplingManager.
func NewSamplingManager(config *Config) *SamplingManager {
	return &SamplingManager{
		config: config,
	}
}

// CanSample checks if the client supports sampling capability.
func (sm *SamplingManager) CanSample(session *mcp.ServerSession) bool {
	if session == nil {
		return false
	}
	initParams := session.InitializeParams()
	if initParams == nil || initParams.Capabilities == nil {
		slog.Debug("Client capabilities not available")
		return false
	}
	if initParams.Capabilities.Sampling == nil {
		slog.Debug("Client does not support sampling capability")
		return false
	}
	return true
}

// SummarizeSearchResults generates a summary of search results using MCP Sampling.
// It sends a request to the client's LLM to summarize the results.
func (sm *SamplingManager) SummarizeSearchResults(
	ctx context.Context,
	session *mcp.ServerSession,
	query string,
	results []map[string]any,
) (string, error) {
	if !sm.config.EnableSampling {
		slog.Debug("Sampling is disabled")
		return "", nil
	}

	if session == nil {
		slog.Debug("ServerSession is nil, cannot perform sampling")
		return "", nil
	}

	if !sm.CanSample(session) {
		return "", nil
	}

	// Build the prompt for summarization
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		slog.Warn("Failed to marshal results for sampling", "error", err)
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	userPrompt := fmt.Sprintf(`Please summarize the following O'Reilly search results for the query "%s".

Provide a concise summary (3-5 sentences) highlighting:
1. The main topics covered
2. Key books/resources found
3. Recommended starting points for learning

Search Results:
%s`, query, string(resultsJSON))

	// Create the sampling request
	params := &mcp.CreateMessageParams{
		Messages: []*mcp.SamplingMessage{
			{
				Role: RoleUser,
				Content: &mcp.TextContent{
					Text: userPrompt,
				},
			},
		},
		MaxTokens: int64(sm.config.SamplingMaxTokens),
		SystemPrompt: `You are a helpful assistant that summarizes O'Reilly Learning Platform search results.
Be concise and focus on helping the user understand what resources are available.
Respond in the same language as the user's query.`,
		Temperature: 0.3, // Lower temperature for more focused summaries
	}

	slog.Debug("Sending sampling request", "query", query, "result_count", len(results))

	// Send the sampling request to the client
	result, err := session.CreateMessage(ctx, params)
	if err != nil {
		slog.Warn("Sampling request failed", "error", err)
		return "", fmt.Errorf("sampling request failed: %w", err)
	}

	// Extract the text from the response
	if textContent, ok := result.Content.(*mcp.TextContent); ok {
		slog.Debug("Sampling completed", "summary_length", len(textContent.Text))
		return textContent.Text, nil
	}

	slog.Warn("Unexpected content type in sampling response", "type", fmt.Sprintf("%T", result.Content))
	return "", nil
}

// SummarizeQuestionAnswer generates a summary of a question and answer.
func (sm *SamplingManager) SummarizeQuestionAnswer(
	ctx context.Context,
	session *mcp.ServerSession,
	question string,
	answer string,
	sources []any,
) (string, error) {
	if !sm.config.EnableSampling {
		slog.Debug("Sampling is disabled")
		return "", nil
	}

	if session == nil {
		slog.Debug("ServerSession is nil, cannot perform sampling")
		return "", nil
	}

	if !sm.CanSample(session) {
		return "", nil
	}

	userPrompt := fmt.Sprintf(`Please create a brief summary of the following Q&A:

Question: %s

Answer: %s

Provide a 2-3 sentence summary of the key points.`, question, answer)

	params := &mcp.CreateMessageParams{
		Messages: []*mcp.SamplingMessage{
			{
				Role: RoleUser,
				Content: &mcp.TextContent{
					Text: userPrompt,
				},
			},
		},
		MaxTokens:    int64(sm.config.SamplingMaxTokens),
		SystemPrompt: "You are a helpful assistant that summarizes technical Q&A content concisely.",
		Temperature:  0.3,
	}

	result, err := session.CreateMessage(ctx, params)
	if err != nil {
		slog.Warn("Sampling request failed", "error", err)
		return "", fmt.Errorf("sampling request failed: %w", err)
	}

	if textContent, ok := result.Content.(*mcp.TextContent); ok {
		return textContent.Text, nil
	}

	return "", nil
}
