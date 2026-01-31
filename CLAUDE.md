# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Tools

### aqua (Package Manager)

This project uses [aqua](https://aquaproj.github.io/) for tool management:

```bash
# Install tools defined in aqua.yaml
aqua install

# List available tools
aqua list
```

**Managed Tools:**

- `go-task/task@v3.44.0` - Task runner
- `golang/tools/goimports@v0.34.0` - Go imports formatter
- `golangci/golangci-lint@v2.1.6` - Go linter
- `deepmap/oapi-codegen@v2.4.1` - OpenAPI code generator

### Task (Task Runner)

This project uses [Task](https://taskfile.dev/) for build automation:

```bash
# List available tasks
task --list

# Development workflow
task dev              # Format + Lint + Test
task ci               # Complete CI workflow
task check            # Quick code quality check

# Individual tasks
task format           # Format code
task lint             # Run linter
task test             # Run tests
task build            # Build binary
task generate:api:oreilly  # Generate OpenAPI client code

# Cleaning
task clean:all        # Clean everything
task clean:generated  # Clean generated code only
```

#### Task Categories and Dependencies

**Code Generation:**
- `generate:api:oreilly` - Generates client from OpenAPI spec

**Code Quality:**
- `format` - Formats Go code with goimports and go fmt
- `lint` - Runs golangci-lint (depends on format)

**Testing:**
- `test` - Runs Go tests (depends on generate:api:oreilly)
- `test:coverage` - Runs tests with coverage report

**Building:**
- `build` - Standard build (depends on generate:api:oreilly, lint)
- `build:release` - Optimized release build (depends on generate:api:oreilly, lint, test)

**Composite Workflows:**
- `dev` - Development workflow (format + lint + test)
- `ci` - Complete CI workflow (generate + format + lint + test:coverage + build)
- `check` - Quick quality check (format + lint)

### OpenAPI Code Generation

This project includes OpenAPI specifications and code generation:

- **Spec file**: `browser/openapi.yaml`
- **Config file**: `browser/oapi-codegen.yaml`
- **Output directory**: `browser/generated/api/`
- **Tool**: [oapi-codegen](https://github.com/deepmap/oapi-codegen)

## Build and Development Commands

### Build the project

```bash
task build
```

### Run the MCP server (stdio mode)

```bash
bin/orm-discovery-mcp-go
```

### Run HTTP server mode

```bash
source .env
bin/orm-discovery-mcp-go
```

### Update dependencies

```bash
task format
```

## High-Level Architecture

This is an **O'Reilly Learning Platform MCP (Model Context Protocol) Server** that provides programmatic access to
O'Reilly content through modern browser automation and API integration. The architecture consists of several key components:

### Core Components

1. **Browser Package** (`browser/`) - Modular browser automation and API client
    - `auth.go` - ChromeDP-based authentication with ACM IdP support and cookie caching
    - `search.go` - Direct HTTP API calls to O'Reilly internal search endpoints
    - `book.go` - Book metadata and chapter content retrieval via OpenAPI client
    - `types.go` - Comprehensive type definitions for API responses
    - `cookie/cookie.go` - Sophisticated cookie management and persistence
    - `generated/api/` - Type-safe OpenAPI client generation

2. **MCPServer** (@server.go) - MCP protocol implementation
    - Exposes 1 MCP tool for content search and 3 MCP resources for content access
    - Handles JSON-RPC request/response mapping
    - Supports both stdio and HTTP transport modes

3. **Config** (@config.go) - Configuration management
    - Loads settings from `.env` files and environment variables
    - Handles executable-relative .env file discovery

### Key Design Patterns

**API-First Architecture**: Modern approach using O'Reilly's internal APIs rather than DOM scraping:
- Generated OpenAPI clients for type safety and consistency
- Direct HTTP API calls for content retrieval
- Browser automation limited to authentication only

**Cookie-Based Session Management**: Sophisticated authentication state management:
- JWT tokens (`orm-jwt`), session IDs (`groot_sessionid`), refresh tokens (`orm-rt`)
- Local cookie caching with validation and expiration handling
- Automatic fallback to password login when cookies expire

**Structured Content Processing**: Native Go processing instead of JavaScript execution:
- HTML parsing with `golang.org/x/net/html` for chapter content
- Comprehensive field normalization for API response variations
- Rich content modeling with separate types for different elements

### Available MCP Tools and Resources

The server exposes the following MCP capabilities:

#### MCP Tools
| Tool               | Description                                         |
|--------------------|-----------------------------------------------------|
| `search_content`   | Content discovery and search - returns book/video/article listings with product IDs for use with resources. **Supports BFS/DFS modes** for context optimization. |
| `ask_question`     | Natural language Q&A using O'Reilly Answers AI - submit technical questions and receive comprehensive AI-generated answers with citations, sources, related resources, and follow-up suggestions |

##### search_content Exploration Modes

The `search_content` tool supports two exploration modes to optimize context consumption:

| Mode | Description | Response Size | Use Case |
|------|-------------|---------------|----------|
| `bfs` (default) | Breadth-First Search - Returns lightweight results (id, title, authors only) | ~2-5KB | Quick discovery, minimal context usage |
| `dfs` | Depth-First Search - Returns full detailed results | ~50-100KB | Comprehensive research, detailed analysis |

**BFS Mode (Recommended for most uses)**:
```json
{
  "query": "Docker",
  "mode": "bfs"
}
```
Returns: `{ "count": 100, "results": [{"id": "123", "title": "...", "authors": [...]}], "history_id": "req_abc123", "note": "Use oreilly://book-details/{id} for full details..." }`

**DFS Mode with Sampling**:
```json
{
  "query": "Docker",
  "mode": "dfs",
  "summarize": true
}
```
Returns full results with AI-generated summary via MCP Sampling.

#### MCP Resources
| Resource URI Pattern | Description | Example |
|---------------------|-------------|---------|
| `oreilly://book-details/{product_id}` | Get comprehensive book information including title, authors, publication date, description, topics, and complete table of contents | `oreilly://book-details/9781098166298` |
| `oreilly://book-toc/{product_id}` | Get detailed table of contents with chapter names, sections, and navigation structure | `oreilly://book-toc/9781098166298` |
| `oreilly://book-chapter/{product_id}/{chapter_name}` | Extract full text content of a specific book chapter including headings, paragraphs, code examples, and structured elements | `oreilly://book-chapter/9781098166298/ch01` |
| `oreilly://answer/{question_id}` | Retrieve answers from previously submitted questions to O'Reilly Answers service | `oreilly://answer/abc123-def456` |
| `orm-mcp://history/recent` | Get recent 20 research entries (searches and questions) | `orm-mcp://history/recent` |
| `orm-mcp://history/search{?keyword,type}` | Search past research by keyword or type (search/question) | `orm-mcp://history/search?keyword=docker` |
| `orm-mcp://history/{id}` | Get details of a specific research entry by ID | `orm-mcp://history/req_abc12345` |
| `orm-mcp://history/{id}/full` | Get full API response data for a research entry (used with BFS mode to access complete data later) | `orm-mcp://history/req_abc12345/full` |

#### MCP Resource Templates
The server provides resource templates for dynamic discovery, allowing MCP clients to understand available resource patterns:
- `oreilly://book-details/{product_id}` - Template for book details access
- `oreilly://book-toc/{product_id}` - Template for table of contents access
- `oreilly://book-chapter/{product_id}/{chapter_name}` - Template for chapter content access
- `oreilly://answer/{question_id}` - Template for answer retrieval access

#### MCP Prompts
The server provides reusable prompt templates for common learning and research workflows:

| Prompt | Title | Description | Arguments |
|--------|-------|-------------|-----------|
| `learn-technology` | Learn a Technology | Generate a structured learning path for a specific technology | `technology` (required): Technology name to learn (e.g., "Kubernetes", "React") |
| `research-topic` | Research a Topic | Conduct multi-perspective research on a technical topic | `topic` (required): Topic to research (e.g., "microservices architecture") |
| `debug-error` | Debug an Error | Generate a debugging guide for a specific error message | `error_message` (required): The error message to debug |
| `review-history` | Review Research History | Review past research activities and identify patterns, trends, and knowledge gaps | None |
| `continue-research` | Continue Research | Continue and deepen a previous research topic | `topic` (required): Topic to continue researching |
| `summarize-history` | Summarize Research History | Summarize a specific research entry with full response data | `history_id` (required): Research entry ID, `focus` (optional): Focus area for summary |

#### Usage Workflow

**Content Discovery and Access:**
1. Use `search_content` tool to discover relevant books/content for specific technologies or concepts
2. Extract `product_id` from search results  
3. Access book details and structure via `oreilly://book-details/{product_id}` resource
4. Access specific chapter content via `oreilly://book-chapter/{product_id}/{chapter_name}` resource

**Natural Language Q&A:**
1. Use `ask_question` tool to submit technical questions to O'Reilly Answers AI service
2. Receive comprehensive response including:
   - AI-generated answer with markdown formatting
   - Source citations and references
   - Related resources for further reading
   - Suggested follow-up questions
   - `question_id` for future reference
3. Optionally access stored answers via `oreilly://answer/{question_id}` resource

**Prompt-Based Workflows:**
1. **Learning Path Generation**: Use `learn-technology` prompt with a technology name to generate a structured learning path with recommended O'Reilly resources
2. **Technical Research**: Use `research-topic` prompt to conduct comprehensive research on a topic, leveraging O'Reilly's content library
3. **Error Debugging**: Use `debug-error` prompt with an error message to get a systematic debugging guide with relevant O'Reilly resources
4. **History Review**: Use `review-history` prompt to analyze past research patterns and identify knowledge gaps
5. **Research Continuation**: Use `continue-research` prompt with a topic to deepen previous research

**Research History Workflow:**
1. All `search_content` and `ask_question` tool calls are automatically recorded
2. Access recent history via `orm-mcp://history/recent` resource
3. Search past research by keyword via `orm-mcp://history/search?keyword=xxx`
4. Filter by type (search/question) via `orm-mcp://history/search?type=search`
5. Get specific entry details via `orm-mcp://history/{id}`

#### Citation Requirements
**IMPORTANT**: All content accessed through these resources must be properly cited with:
- Book title and author(s)
- Chapter title (when applicable)  
- Publisher: O'Reilly Media
- Proper attribution as required by O'Reilly's terms of service

## Environment Setup

### Required Environment Variables

```bash
OREILLY_USER_ID=your_email@example.com    # Your O'Reilly account email
OREILLY_PASSWORD=your_password             # Your O'Reilly password
PORT=8080                                  # HTTP server port (optional)
TRANSPORT=stdio                            # Transport mode: stdio or http

# Search Mode 設定
ORM_MCP_GO_DEFAULT_MODE=bfs               # Default search mode: bfs | dfs

# Sampling 設定
ORM_MCP_GO_ENABLE_SAMPLING=true           # Enable MCP Sampling for summarization (default: true)
ORM_MCP_GO_SAMPLING_MAX_TOKENS=500        # Max tokens for sampling responses (default: 500)
```

### .env File Support

Place a `.env` file in the same directory as the executable. The system will automatically detect and load it, with
`.env` values taking precedence over environment variables.

## Dependencies

**Core Framework**: `github.com/mark3labs/mcp-go v0.43.2` - MCP protocol implementation
**Browser Automation**: `github.com/chromedp/chromedp` - Chrome DevTools Protocol

## Browser Requirements

This application requires Chrome or Chromium to be installed on the system for headless browser operations. The browser
is used to:

- Authenticate with O'Reilly Learning Platform
- Navigate and scrape content from web pages
- Execute JavaScript for DOM manipulation
- Handle complex authentication flows (including ACM IdP redirects)

## File Organization

**Root Level:**
- `main.go` - Entry point with CLI interface
- `server.go` - MCP server with tool and resource handlers
- `config.go` - Configuration management and environment variable handling
- `research_history.go` - Research history management (ResearchHistoryManager)
- `history_resources.go` - MCP resources for history access (orm-mcp://history/*)

**Browser Package** (`browser/`):
- `auth.go` - Authentication logic with cookie caching and ACM IdP support
- `search.go` - Search API implementation using OpenAPI client
- `book.go` - Book operations (details, TOC, chapter content)
- `types.go` - Type definitions and response structures
- `debug.go` - Debug utilities and screenshot capture
- `cookie/cookie.go` - Cookie management interface and JSON persistence
- `generated/api/` - OpenAPI-generated client code

**Configuration Files:**
- `browser/openapi.yaml` - OpenAPI specification for O'Reilly APIs
- `browser/oapi-codegen.yaml` - Code generation configuration
- `aqua.yaml` - Tool dependency management
- `Taskfile.yml` - Build automation and workflow definitions

## Important Notes

**Modern Implementation Approach:**
- Uses O'Reilly's internal APIs directly for content retrieval (faster and more reliable)
- Browser automation limited to authentication only (not content scraping)
- Generated OpenAPI clients provide type safety and consistency

**Authentication Requirements:**
- Valid O'Reilly Learning Platform credentials required
- ACM (Association for Computing Machinery) institutional login automatically detected and handled
- Cookie caching improves performance by avoiding repeated logins

**System Dependencies:**
- Chrome or Chromium installation required for authentication browser automation
- Aqua package manager for tool dependency management
- Task runner for standardized build and development workflows

**Browser Package Memory Reference:**
For detailed implementation patterns and module-specific guidance, see @browser/CLAUDE

## Task Completion Quality Assurance

### CRITICAL REQUIREMENT: Test and Build Verification

**MANDATORY**: Before completing any development task, you MUST ensure that both tests and builds succeed. This is a non-negotiable requirement for maintaining code quality and project stability.

#### Required Verification Steps

**For ANY code changes, you MUST run:**

```bash
task ci    # Complete CI workflow including tests and build
```

**If `task ci` fails, the task is NOT complete until all issues are resolved.**

#### Alternative Verification Commands

If you need to run individual steps:

```bash
# Step 1: Ensure code quality
task check              # Format + Lint

# Step 2: Ensure functionality  
task test              # Run all tests

# Step 3: Ensure buildability
task build             # Build the project
```

#### What Must Pass

1. **Code Quality Checks**:
   - `task format` - Code formatting must be consistent
   - `task lint` - All linting rules must pass (0 issues)

2. **Functionality Tests**:
   - `task test` - All tests must pass without errors
   - No test failures or panics allowed

3. **Build Verification**:
   - `task build` - Project must compile successfully
   - No compilation errors allowed

#### When to Run Verification

**ALWAYS run verification after:**
- Adding new code or features
- Modifying existing code
- Refactoring
- Updating dependencies
- Making configuration changes
- Before committing changes

#### Failure Resolution

**If any verification step fails:**

1. **Fix the issue immediately** - Do not proceed with other tasks
2. **Re-run the failed step** to confirm the fix
3. **Run `task ci`** to ensure overall project health
4. **Only then consider the task complete**

#### Exception Policy

**There are NO exceptions to this requirement.** Even for:
- Documentation-only changes (may affect build/generate tasks)
- Configuration updates (may affect functionality)  
- "Minor" code changes (may have unexpected side effects)

#### CI Integration

The GitHub Actions CI pipeline enforces these same requirements:
- All tasks in `task ci` must pass for PRs to be mergeable
- Local verification prevents CI failures and speeds up development

#### Summary

**Task completion checklist:**
- [ ] Code changes implemented
- [ ] `task ci` executed successfully  
- [ ] All tests pass
- [ ] Build succeeds
- [ ] No linting errors
- [ ] Task is now complete

**Remember: A task is only complete when `task ci` passes without errors.**

## Testing and Development

### MCP Standard I/O Mode Testing

**CRITICAL**: All functionality testing must be performed using MCP standard input/output mode, not standalone CLI commands.

#### Starting the MCP Server

```bash
# Start MCP server in stdio mode (default)
go run .

# The server will output initialization logs and then wait for MCP JSON-RPC requests over stdin/stdout
# Example output:
# 2025/06/28 13:10:51 設定を読み込みました
# 2025/06/28 13:10:53 ブラウザクライアントの初期化が完了しました
# 2025/06/28 13:10:54 MCPサーバーを標準入出力で起動します
```

#### MCP Protocol Testing

Use MCP-compatible clients to test functionality:

**1. Search Content Testing:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "search_content",
    "arguments": {
      "query": "Docker containers"
    }
  }
}
```

**2. Ask Question Testing:**
```json
{
  "jsonrpc": "2.0", 
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "ask_question",
    "arguments": {
      "question": "What career paths are available for software engineers in their late 30s?",
      "max_wait_minutes": 5
    }
  }
}
```

**3. Resource Access Testing:**
```json
{
  "jsonrpc": "2.0",
  "id": 3, 
  "method": "resources/read",
  "params": {
    "uri": "oreilly://book-details/9781098166298"
  }
}
```

#### Testing with Claude Code

The easiest way to test is using Claude Code as an MCP client:

1. Start the MCP server: `go run .`
2. Use Claude Code to interact with the tools and resources
3. Test various scenarios:
   - Content search: "Search for books about machine learning"
   - Natural language Q&A: "Ask about Python best practices for beginners"
   - Resource access: Access book details and chapter content

#### Header Verification Testing

To verify 401 authentication issues are resolved:

1. Enable debug mode: `ORM_MCP_GO_DEBUG=true go run .`
2. Monitor debug logs for header transmission:
   ```
   API呼び出し先URL: https://learning.oreilly.com/api/v1/miso-answers-relay-service/questions/
   送信予定Cookie数: 20
   送信Cookie: groot_sessionid=... (Domain: .oreilly.com, Path: /)
   ```
3. Verify all required headers are sent:
   - Accept: */*
   - Referer: https://learning.oreilly.com/answers2/
   - Origin: https://learning.oreilly.com
   - Sec-Fetch-* security headers

#### Important Testing Notes

- **Do NOT implement standalone CLI commands** - All testing goes through MCP protocol
- **Cookie authentication** is handled automatically through the browser client
- **Debug mode** provides detailed logs for troubleshooting authentication issues
- **All API calls** use the comprehensive header set matching real browser requests

## Architecture Overview

This is an **O'Reilly Learning Platform MCP Server** providing programmatic access to O'Reilly content through modern browser automation and API integration.

### Modular Browser Package Architecture

The `browser/` package implements a clean, modular design:

1. **Authentication Layer** (`browser/auth.go`)
   - Cookie-first authentication strategy with validation
   - ChromeDP-based browser automation for login flows (see `.claude/skills/chromedp-lifecycle.md` for lifecycle management)
   - ACM IdP automatic detection and handling

2. **API Integration Layer** (`browser/search.go`, `browser/book.go`)
   - Generated OpenAPI clients for type-safe API calls
   - Direct HTTP communication with O'Reilly internal endpoints
   - Comprehensive response normalization and error handling

3. **Data Management Layer** (`browser/types.go`, `browser/cookie/`)
   - Rich type definitions for all API responses
   - Interface-based cookie management with JSON persistence
   - Structured content modeling for chapters and TOC

4. **Development Support** (`browser/debug.go`, `browser/generated/`)
   - Environment-controlled debugging with screenshot capture
   - Automated OpenAPI client generation for API consistency

### MCP Protocol Implementation

**Server Layer** (`server.go`):
- 1 MCP tool: `search_content` for content discovery
- 3 MCP resources: `book-details`, `book-toc`, `book-chapter` for content access
- Resource templates for dynamic discovery
- JSON-RPC request/response handling with error propagation

### Key Design Patterns

- **API-First Content Access**: Direct HTTP calls instead of DOM scraping
- **Cookie-Based Session Management**: Persistent authentication with validation
- **Type-Safe Processing**: OpenAPI-generated Go structs for consistency
- **Modular Package Design**: Clear separation of concerns across modules
- **Interface-Based Development**: Testable and flexible component design

### Environment Configuration

```bash
# Core credentials
OREILLY_USER_ID=your_email@acm.org    # O'Reilly account email
OREILLY_PASSWORD=your_password         # O'Reilly password

# Server configuration
PORT=8080                              # HTTP server port (optional)
TRANSPORT=stdio                        # Transport mode: stdio or http

# Development and debugging
ORM_MCP_GO_DEBUG=true                 # Enable debug logging and screenshots
ORM_MCP_GO_TMP_DIR=/path/to/tmp       # Custom temporary directory for cookies
```

### System Dependencies

**Runtime Requirements:**
- Chrome or Chromium for authentication browser automation
- Go 1.24.3+ for modern language features

**Development Tools** (managed via Aqua):
- Task runner for workflow automation
- golangci-lint for code quality
- oapi-codegen for OpenAPI client generation

## Development Workflow

### Standard Development Cycle
```bash
# Development workflow
task dev              # Format + Lint + Test

# Individual tasks
task format           # Format code
task lint             # Run linter
task test             # Run tests
task build            # Build binary

# Complete CI workflow
task ci               # All checks + build
```

### Code Quality Requirements
- All code must pass `golangci-lint` with 0 issues
- Code formatting with `goimports` and `go fmt`
- All tests must pass before task completion

## Cross-Package Integration

### Memory Import Strategy

This project uses a hierarchical memory system:

1. **Main Memory** (`CLAUDE.md`) - High-level architecture and development workflows
2. **Package Memory** (`browser/CLAUDE.md`) - Detailed implementation patterns and module-specific guidance
3. **Cross-Reference Integration** - Main memory imports and references package-specific knowledge

**When working on browser package issues:**
1. Consult `browser/CLAUDE.md` for detailed implementation patterns
2. Reference main `CLAUDE.md` for overall architecture context
3. Follow established development workflows and quality requirements

### Package Memory References

- **Browser Implementation Details**: See `browser/CLAUDE.md` for:
  - Authentication flow patterns and cookie management
  - OpenAPI client integration examples
  - HTML content parsing strategies
  - Error handling and debugging approaches

## Future Development Considerations

### Implemented Features
- **Cookie Caching**: ✅ Implemented in `browser/cookie/cookie.go`
  - JSON format storage with configurable temp directory
  - Cookie validation with automatic fallback to password login
  - Secure file permissions (0600) and expiration handling

### API Expansion Opportunities
- Enhanced search filters and pagination
- User-specific content access (playlists, bookmarks)
- In-book search functionality
- Content summarization features

## Claude Code Plugin Distribution

### Plugin Installation

This project is distributed as a Claude Code Plugin Marketplace:

```bash
# Add the marketplace
/plugin marketplace add usadamasa/orm-discovery-mcp-go

# Install the plugin
/plugin install orm-discovery-mcp-go

# Required environment variables
export OREILLY_USER_ID="your_email@acm.org"
export OREILLY_PASSWORD="your_password"
```

**Note:** The MCP Server binary must be installed separately via [Releases](https://github.com/usadamasa/orm-discovery-mcp-go/releases) or built from source.

### Marketplace Structure

```
.claude-plugin/
├── marketplace.json     # Marketplace definition
├── plugin.json          # Plugin definition
└── agents/
    └── oreilly-researcher.md  # Agent definition
```

### Validation

```bash
# Validate marketplace and plugin configuration
task plugin:validate:marketplace

# Full CI validation (includes marketplace validation)
task ci
```

### What the Plugin Provides

- **Agent**: `oreilly-researcher` - Research technical topics using O'Reilly Learning Platform
- **MCP Server Configuration**: Pre-configured `orm-discovery-mcp-go` MCP server settings

## Claude Code Agent Integration

### O'Reilly Researcher Agent

This project provides a specialized Claude Code agent for efficient O'Reilly content research:

**Agent Location (Plugin):** `agents/oreilly-researcher.md`

**When to Use:**
- Researching technical topics using O'Reilly resources
- Finding learning materials for specific technologies
- Technical Q&A leveraging O'Reilly Answers AI

**Usage Example:**
```
User: "Dockerについて調べてほしい"
Claude Code: Uses oreilly-researcher agent for comprehensive O'Reilly search
```

**Agent Workflow:**
1. Uses `search_content` with BFS mode for lightweight discovery
2. Leverages `ask_question` for technical Q&A
3. Accesses `book-details` and `book-chapter` for deep dives
4. Returns structured findings with citations

**Benefits:**
- **Context Efficiency**: Agent runs in independent context (subagent)
- **BFS/DFS Support**: Optimized for minimal parent context consumption
- **Research History**: Integrates with `orm-mcp://history/*` resources

## Security Considerations

**Authentication Security:**
- Environment variable credential storage (never hardcoded)
- Cookie file permissions (0600) for cached authentication
- Session timeout and cookie expiration handling
- Rate limiting compliance with O'Reilly platform

**Development Security:**
- Debug mode controls for sensitive screenshot capture
- Secure temporary directory configuration
- Proper error handling without credential leakage