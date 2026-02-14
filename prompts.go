package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts registers MCP prompt handlers.
func (s *Server) registerPrompts() {
	// learn-technology prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "learn-technology",
			Title:       "Learn a Technology",
			Description: "Generate a structured learning path for a specific technology.\n\nExample: learn-technology(technology=\"Docker\", experience_level=\"beginner\")\n\nIMPORTANT: Uses oreilly_search_content and book-details resources for learning.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "technology",
					Description: "The technology name to learn (e.g., Docker, Kubernetes, React)",
					Required:    true,
				},
				{
					Name:        "experience_level",
					Description: "Experience level: beginner, intermediate, or advanced (default: beginner)",
					Required:    false,
				},
			},
		},
		s.LearnTechnologyPromptHandler,
	)

	// review-history prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "review-history",
			Title:       "Review Research History",
			Description: "Review past research history and find related information.\n\nExample: review-history(keyword=\"docker\")\n\nWorkflow: Access orm-mcp://history/recent or search by keyword.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "keyword",
					Description: "Optional keyword to filter research history",
					Required:    false,
				},
			},
		},
		s.ReviewHistoryPromptHandler,
	)

	// continue-research prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "continue-research",
			Title:       "Continue Previous Research",
			Description: "Continue and deepen a previous research.\n\nExample: continue-research(research_id=\"req_abc123\")\n\nWorkflow: Retrieve past research and conduct additional searches.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "research_id",
					Description: "The ID of the research entry to continue (e.g., req_abc123)",
					Required:    true,
				},
			},
		},
		s.ContinueResearchPromptHandler,
	)

	// research-topic prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "research-topic",
			Title:       "Research a Topic",
			Description: "Conduct multi-perspective research on a technical topic.\n\nExample: research-topic(topic=\"microservices\", depth=\"detailed\")\n\nIMPORTANT: Combines oreilly_ask_question and oreilly_search_content for comprehensive research.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "topic",
					Description: "The technical topic to research",
					Required:    true,
				},
				{
					Name:        "depth",
					Description: "Research depth: overview, detailed, or comprehensive (default: overview)",
					Required:    false,
				},
			},
		},
		s.ResearchTopicPromptHandler,
	)

	// debug-error prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "debug-error",
			Title:       "Debug an Error",
			Description: "Guide for troubleshooting and debugging errors.\n\nExample: debug-error(error_message=\"NullPointerException\", technology=\"Java\")\n\nIMPORTANT: Uses O'Reilly Answers and documentation for solutions.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "error_message",
					Description: "The error message to debug",
					Required:    true,
				},
				{
					Name:        "technology",
					Description: "The technology/language where the error occurred",
					Required:    true,
				},
				{
					Name:        "context",
					Description: "Additional context about when/where the error occurs",
					Required:    false,
				},
			},
		},
		s.DebugErrorPromptHandler,
	)

	// summarize-history prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "summarize-history",
			Title:       "Summarize Research History",
			Description: "Summarize a specific research entry with full response data.\n\nExample: summarize-history(history_id=\"req_abc123\")\n\nWorkflow: Access full data via orm-mcp://history/{id}/full and generate a concise summary.",
			Arguments: []*mcp.PromptArgument{
				{
					Name:        "history_id",
					Description: "The ID of the research entry to summarize (e.g., req_abc123)",
					Required:    true,
				},
				{
					Name:        "focus",
					Description: "Optional focus area for the summary (e.g., 'key concepts', 'practical examples')",
					Required:    false,
				},
			},
		},
		s.SummarizeHistoryPromptHandler,
	)
}

// LearnTechnologyPromptHandler handles the learn-technology prompt.
func (s *Server) LearnTechnologyPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("learn-technologyプロンプトリクエスト受信")

	technology := getStringArg(req.Params.Arguments, "technology", "")
	if technology == "" {
		return nil, fmt.Errorf("technology argument is required")
	}

	experienceLevel := getStringArg(req.Params.Arguments, "experience_level", "beginner")

	slog.Info("learn-technologyプロンプト生成完了", "technology", technology, "level", experienceLevel)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Learning path for %s (%s level)", technology, experienceLevel),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildLearnTechnologyUserMessage(technology, experienceLevel),
				},
			},
		},
	}, nil
}

// ResearchTopicPromptHandler handles the research-topic prompt.
func (s *Server) ResearchTopicPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("research-topicプロンプトリクエスト受信")

	topic := getStringArg(req.Params.Arguments, "topic", "")
	if topic == "" {
		return nil, fmt.Errorf("topic argument is required")
	}

	depth := getStringArg(req.Params.Arguments, "depth", "overview")

	slog.Info("research-topicプロンプト生成完了", "topic", topic, "depth", depth)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Research on %s (%s depth)", topic, depth),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildResearchTopicUserMessage(topic, depth),
				},
			},
		},
	}, nil
}

// DebugErrorPromptHandler handles the debug-error prompt.
func (s *Server) DebugErrorPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("debug-errorプロンプトリクエスト受信")

	errorMessage := getStringArg(req.Params.Arguments, "error_message", "")
	if errorMessage == "" {
		return nil, fmt.Errorf("error_message argument is required")
	}

	technology := getStringArg(req.Params.Arguments, "technology", "")
	if technology == "" {
		return nil, fmt.Errorf("technology argument is required")
	}

	errorContext := getStringArg(req.Params.Arguments, "context", "")

	slog.Info("debug-errorプロンプト生成完了", "error", errorMessage, "technology", technology)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Debug %s error in %s", errorMessage, technology),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildDebugErrorUserMessage(errorMessage, technology, errorContext),
				},
			},
		},
	}, nil
}

// getStringArg retrieves a string argument from the arguments map.
func getStringArg(args map[string]string, key, defaultValue string) string {
	if args == nil {
		return defaultValue
	}
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

// buildLearnTechnologyUserMessage builds the user message for learn-technology prompt.
func buildLearnTechnologyUserMessage(technology, experienceLevel string) string {
	return fmt.Sprintf(`You are an O'Reilly learning assistant helping with %s.

User's experience level: %s

Learning Strategy:
1. Use oreilly_search_content to find foundational materials
   - Recommended query: "%s fundamentals"
2. Access book details via oreilly://book-details/{product_id}
3. Read key chapters via oreilly://book-chapter/{product_id}/{chapter}
4. For questions, use oreilly_ask_question tool

Workflow:
- Start by searching for beginner-friendly content
- Review book table of contents to identify key learning chapters
- Focus on practical examples and hands-on exercises
- Build complexity gradually based on the experience level

IMPORTANT: Always cite sources (title, author, O'Reilly Media) when referencing content.`, technology, experienceLevel, technology)
}

// buildResearchTopicUserMessage builds the user message for research-topic prompt.
func buildResearchTopicUserMessage(topic, depth string) string {
	depthGuidance := ""
	switch depth {
	case "overview":
		depthGuidance = "Provide a high-level overview with key concepts and terminology."
	case "detailed":
		depthGuidance = "Include detailed explanations, best practices, and common patterns."
	case "comprehensive":
		depthGuidance = "Perform exhaustive research covering all aspects, edge cases, and advanced topics."
	default:
		depthGuidance = "Provide a high-level overview with key concepts and terminology."
	}

	return fmt.Sprintf(`You are conducting research on %s using O'Reilly resources.

Research depth: %s
%s

Workflow:
1. Start with oreilly_ask_question for overview: "%s overview and key concepts"
2. Search for related content: oreilly_search_content("%s")
3. Deep dive into relevant books and chapters
4. Synthesize findings with proper citations

Research Guidelines:
- Cross-reference multiple sources for accuracy
- Identify authoritative books and authors on the topic
- Note any conflicting information or evolving practices
- Include practical examples and real-world applications

IMPORTANT: Cross-reference multiple sources for accuracy. Always cite sources with proper attribution.`, topic, depth, depthGuidance, topic, topic)
}

// buildDebugErrorUserMessage builds the user message for debug-error prompt.
func buildDebugErrorUserMessage(errorMessage, technology, errorContext string) string {
	contextInfo := ""
	if errorContext != "" {
		contextInfo = fmt.Sprintf("\nContext: %s", errorContext)
	}

	return fmt.Sprintf(`You are debugging an error in %s.

Error message: %s%s

Debug Strategy:
1. Use oreilly_ask_question to understand the error: "What causes %s in %s?"
2. Search for related solutions: oreilly_search_content("%s error handling")
3. Check relevant documentation chapters for proper patterns
4. Provide step-by-step resolution with explanations

Troubleshooting Guidelines:
- Identify the root cause, not just symptoms
- Consider common causes and edge cases
- Suggest preventive measures for the future
- Reference official documentation and best practices

IMPORTANT: Verify solutions against official documentation. Always provide tested, reliable fixes.`, technology, errorMessage, contextInfo, errorMessage, technology, technology)
}

// ReviewHistoryPromptHandler handles the review-history prompt.
func (s *Server) ReviewHistoryPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("review-historyプロンプトリクエスト受信")

	keyword := getStringArg(req.Params.Arguments, "keyword", "")

	slog.Info("review-historyプロンプト生成完了", "keyword", keyword)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Review research history%s", func() string {
			if keyword != "" {
				return fmt.Sprintf(" (keyword: %s)", keyword)
			}
			return ""
		}()),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildReviewHistoryUserMessage(keyword),
				},
			},
		},
	}, nil
}

// ContinueResearchPromptHandler handles the continue-research prompt.
func (s *Server) ContinueResearchPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("continue-researchプロンプトリクエスト受信")

	researchID := getStringArg(req.Params.Arguments, "research_id", "")
	if researchID == "" {
		return nil, fmt.Errorf("research_id argument is required")
	}

	slog.Info("continue-researchプロンプト生成完了", "research_id", researchID)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Continue research %s", researchID),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildContinueResearchUserMessage(researchID),
				},
			},
		},
	}, nil
}

// buildReviewHistoryUserMessage builds the user message for review-history prompt.
func buildReviewHistoryUserMessage(keyword string) string {
	keywordGuidance := ""
	resourceURI := "orm-mcp://history/recent"
	if keyword != "" {
		keywordGuidance = fmt.Sprintf("\nFilter by keyword: %s", keyword)
		resourceURI = fmt.Sprintf("orm-mcp://history/search?keyword=%s", keyword)
	}

	return fmt.Sprintf(`You are reviewing past research history from O'Reilly resources.%s

Workflow:
1. Access research history: %s
2. Review past searches and questions
3. Identify patterns and frequently researched topics
4. Suggest related areas for further exploration

Review Guidelines:
- Summarize key findings from past research
- Identify knowledge gaps or areas needing deeper investigation
- Cross-reference related research entries
- Propose next steps based on research patterns

IMPORTANT: Use past research to avoid redundant queries and build upon existing knowledge.`, keywordGuidance, resourceURI)
}

// buildContinueResearchUserMessage builds the user message for continue-research prompt.
func buildContinueResearchUserMessage(researchID string) string {
	return fmt.Sprintf(`You are continuing a previous research session.

Research to continue: %s

Workflow:
1. Retrieve original research: orm-mcp://history/%s
2. Review the original query and results
3. Identify areas for deeper exploration
4. Execute additional searches or questions on the same topic
5. Synthesize new findings with previous results

Continuation Guidelines:
- Build upon the original research, don't repeat it
- Focus on unanswered questions or emerging topics
- Cross-reference with new sources
- Provide an updated summary combining old and new insights

IMPORTANT: Reference the original research when presenting continued findings.`, researchID, researchID)
}

// SummarizeHistoryPromptHandler handles the summarize-history prompt.
func (s *Server) SummarizeHistoryPromptHandler(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	slog.Debug("summarize-historyプロンプトリクエスト受信")

	historyID := getStringArg(req.Params.Arguments, "history_id", "")
	if historyID == "" {
		return nil, fmt.Errorf("history_id argument is required")
	}

	focus := getStringArg(req.Params.Arguments, "focus", "")

	slog.Info("summarize-historyプロンプト生成完了", "history_id", historyID, "focus", focus)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Summarize research entry %s", historyID),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: buildSummarizeHistoryUserMessage(historyID, focus),
				},
			},
		},
	}, nil
}

// buildSummarizeHistoryUserMessage builds the user message for summarize-history prompt.
func buildSummarizeHistoryUserMessage(historyID, focus string) string {
	focusGuidance := ""
	if focus != "" {
		focusGuidance = fmt.Sprintf("\nFocus area: %s", focus)
	}

	return fmt.Sprintf(`You are summarizing a research entry from O'Reilly resources.

Research entry ID: %s%s

Workflow:
1. Access full research data: orm-mcp://history/%s/full
2. Review the complete response data
3. Generate a concise summary highlighting key findings
4. Identify actionable insights and recommendations

Summary Guidelines:
- Keep the summary to 3-5 key points
- Focus on the most relevant and practical information
- Include any notable books, authors, or resources
- Suggest next steps or related topics to explore

Output Format:
- Brief overview of the research topic
- Key findings (bullet points)
- Notable resources with citations
- Recommended next steps

IMPORTANT: Use the full data from the resource to create an accurate summary. Cite sources properly.`, historyID, focusGuidance, historyID)
}
