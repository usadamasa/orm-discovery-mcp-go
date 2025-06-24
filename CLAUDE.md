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
go build .
```

### Run the MCP server (stdio mode)

```bash
go run .
```

### Run HTTP server mode

```bash
export TRANSPORT=http
export PORT=8080
go run .
```

### Run search tests

```bash
go run . test
go run . test "Docker"
go run . test playlists
```

### Update dependencies

```bash
go mod tidy
```

## High-Level Architecture

This is an **O'Reilly Learning Platform MCP (Model Context Protocol) Server** that provides programmatic access to
O'Reilly content through browser automation. The architecture consists of several key components:

### Core Components

1. **BrowserClient** (`browser_client.go`, ~2,700 lines) - The heart of the system
    - Uses ChromeDP for headless browser automation
    - Handles O'Reilly login with ACM IdP support
    - Scrapes web content using DOM selectors and JavaScript execution
    - Manages authentication cookies and session state

2. **OreillyClient** (`oreilly_client.go`) - High-level API wrapper
    - Abstracts browser operations into clean interfaces
    - Provides structured search functionality
    - Manages content extraction and normalization

3. **MCPServer** (`server.go`) - MCP protocol implementation
    - Exposes 1 MCP tool for content search and 3 MCP resources for content access
    - Handles JSON-RPC request/response mapping
    - Supports both stdio and HTTP transport modes

4. **Config** (`config.go`) - Configuration management
    - Loads settings from `.env` files and environment variables
    - Handles executable-relative .env file discovery

### Key Design Patterns

**Browser-First Approach**: This system doesn't use traditional REST APIs. Instead, it automates the actual O'Reilly
Learning Platform web interface using a headless browser. This is because O'Reilly doesn't provide public APIs.

**Cookie-Based Authentication**: The system extracts JWT tokens (`orm-jwt`), session IDs (`groot_sessionid`), and
refresh tokens (`orm-rt`) from browser sessions to maintain authentication state.

**DOM Scraping with JavaScript**: Content extraction relies on JavaScript execution within the browser context to query
DOM elements and extract structured data from the web interface.

### Available MCP Tools and Resources

The server exposes the following MCP capabilities:

#### MCP Tools
| Tool               | Description                                         |
|--------------------|-----------------------------------------------------|
| `search_content`   | Content discovery and search - returns book/video/article listings with product IDs for use with resources |

#### MCP Resources  
| Resource URI Pattern | Description | Example |
|---------------------|-------------|---------|
| `oreilly://book-details/{product_id}` | Get comprehensive book information including title, authors, publication date, description, topics, and complete table of contents | `oreilly://book-details/9781098166298` |
| `oreilly://book-toc/{product_id}` | Get detailed table of contents with chapter names, sections, and navigation structure | `oreilly://book-toc/9781098166298` |
| `oreilly://book-chapter/{product_id}/{chapter_name}` | Extract full text content of a specific book chapter including headings, paragraphs, code examples, and structured elements | `oreilly://book-chapter/9781098166298/ch01` |

#### MCP Resource Templates
The server provides resource templates for dynamic discovery, allowing MCP clients to understand available resource patterns:
- `oreilly://book-details/{product_id}` - Template for book details access
- `oreilly://book-toc/{product_id}` - Template for table of contents access  
- `oreilly://book-chapter/{product_id}/{chapter_name}` - Template for chapter content access

#### Usage Workflow
1. Use `search_content` tool to discover relevant books/content for specific technologies or concepts
2. Extract `product_id` from search results  
3. Access book details and structure via `oreilly://book-details/{product_id}` resource
4. Access specific chapter content via `oreilly://book-chapter/{product_id}/{chapter_name}` resource

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
```

### .env File Support

Place a `.env` file in the same directory as the executable. The system will automatically detect and load it, with
`.env` values taking precedence over environment variables.

## Dependencies

**Core Framework**: `github.com/mark3labs/mcp-go` - MCP protocol implementation
**Browser Automation**: `github.com/chromedp/chromedp` - Chrome DevTools Protocol

## Browser Requirements

This application requires Chrome or Chromium to be installed on the system for headless browser operations. The browser
is used to:

- Authenticate with O'Reilly Learning Platform
- Navigate and scrape content from web pages
- Execute JavaScript for DOM manipulation
- Handle complex authentication flows (including ACM IdP redirects)

## File Organization

- `main.go` (225 lines) - Entry point with test modes
- `server.go` (~420 lines) - MCP server with tool and resource handlers
- `browser_client.go` (2,728 lines) - Browser automation logic
- `oreilly_client.go` (243 lines) - High-level client wrapper
- `config.go` (72 lines) - Configuration management

## Testing

The application includes built-in test modes accessible via command line:

- General search testing: `go run . test`
- Custom query testing: `go run . test "your query"`
- Playlist functionality testing: `go run . test playlists`

## Important Notes

- This system works by automating the web interface, making it sensitive to O'Reilly's frontend changes
- Authentication requires valid O'Reilly Learning Platform credentials
- The browser automation may be slower than direct API calls but provides access to content not available through public
  APIs
- ACM (Association for Computing Machinery) institutional login is automatically detected and handled

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