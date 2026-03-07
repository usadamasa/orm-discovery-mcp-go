package main

// MCP description constants.
// Extracted for testability and progressive-disclosure optimization.
// See .claude/skills/mcp-tool-progressive-disclosure/SKILL.md for guidelines.

// Tool descriptions.

const descSearchContent = `Search O'Reilly content and return books/videos/articles with product_id for resource access.

Example: "Docker containers" (Good) / "How to use Docker" (Poor)

Results: Use product_id with oreilly://book-details/{id} or oreilly://book-chapter/{id}/{chapter}

IMPORTANT: Cite sources with title, author(s), and O'Reilly Media.`

const descAskQuestion = `Ask technical questions to O'Reilly Answers AI and get sourced responses.

Example: "How to optimize React performance?" (Good) / "Explain everything about React" (Poor)

Response: Markdown answer, sources, related resources, question_id (use with oreilly://answer/{id})

IMPORTANT: Cite sources provided in the response.`

// Resource descriptions.

const (
	descResBookDetails = "Get book info (title, ISBN, description, publication date). Cite sources when referencing."
	descResBookTOC     = "Get table of contents with chapter names and structure. Cite book title, author(s), O'Reilly Media."
	descResBookChapter = "Get full chapter text. CRITICAL: Cite book title, author(s), chapter title, O'Reilly Media."
	descResAnswer      = "Retrieve previously generated answer by question_id. Cite sources when referencing."
	descResHistRecent  = "Get recent 20 research entries. Use to review past searches and questions."
)

// Resource template descriptions.

const (
	descTmplBookDetails = "Use product_id from oreilly_search_content to get book details."
	descTmplBookTOC     = "Use product_id from oreilly_search_content to get table of contents."
	descTmplBookChapter = "Use product_id and chapter_name to get chapter content."
	descTmplAnswer      = "Use question_id from oreilly_ask_question to retrieve the answer."
	descTmplHistSearch  = "Search past research by keyword or type (search/question)."
	descTmplHistDetail  = "Get details of a specific research entry by ID."
	descTmplHistFull    = "Get the full API response data for a research entry. Use with BFS mode to access complete data later."
)

// Prompt descriptions.

const (
	descPromptLearnTech  = "Generate a structured learning path for a specific technology.\n\nExample: learn-technology(technology=\"Docker\", experience_level=\"beginner\")\n\nIMPORTANT: Uses oreilly_search_content and book-details resources for learning."
	descPromptReviewHist = "Review past research history and find related information.\n\nExample: review-history(keyword=\"docker\")\n\nWorkflow: Access orm-mcp://history/recent or search by keyword."
	descPromptContRes    = "Continue and deepen a previous research.\n\nExample: continue-research(research_id=\"req_abc123\")\n\nWorkflow: Retrieve past research and conduct additional searches."
	descPromptResTopic   = "Conduct multi-perspective research on a technical topic.\n\nExample: research-topic(topic=\"microservices\", depth=\"detailed\")\n\nIMPORTANT: Combines oreilly_ask_question and oreilly_search_content for comprehensive research."
	descPromptDebugErr   = "Guide for troubleshooting and debugging errors.\n\nExample: debug-error(error_message=\"NullPointerException\", technology=\"Java\")\n\nIMPORTANT: Uses O'Reilly Answers and documentation for solutions."
	descPromptSumHist    = "Summarize a specific research entry with full response data.\n\nExample: summarize-history(history_id=\"req_abc123\")\n\nWorkflow: Access full data via orm-mcp://history/{id}/full and generate a concise summary."
)
