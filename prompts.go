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
			Description: "Generate a structured learning path for a specific technology.\n\nExample: learn-technology(technology=\"Docker\", experience_level=\"beginner\")\n\nIMPORTANT: Uses search_content and book-details resources for learning.",
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

	// research-topic prompt
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "research-topic",
			Title:       "Research a Topic",
			Description: "Conduct multi-perspective research on a technical topic.\n\nExample: research-topic(topic=\"microservices\", depth=\"detailed\")\n\nIMPORTANT: Combines ask_question and search_content for comprehensive research.",
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
1. Use search_content to find foundational materials
   - Recommended query: "%s fundamentals"
2. Access book details via oreilly://book-details/{product_id}
3. Read key chapters via oreilly://book-chapter/{product_id}/{chapter}
4. For questions, use ask_question tool

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
1. Start with ask_question for overview: "%s overview and key concepts"
2. Search for related content: search_content("%s")
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
1. Use ask_question to understand the error: "What causes %s in %s?"
2. Search for related solutions: search_content("%s error handling")
3. Check relevant documentation chapters for proper patterns
4. Provide step-by-step resolution with explanations

Troubleshooting Guidelines:
- Identify the root cause, not just symptoms
- Consider common causes and edge cases
- Suggest preventive measures for the future
- Reference official documentation and best practices

IMPORTANT: Verify solutions against official documentation. Always provide tested, reliable fixes.`, technology, errorMessage, contextInfo, errorMessage, technology, technology)
}
