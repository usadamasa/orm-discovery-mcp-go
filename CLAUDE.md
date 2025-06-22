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

# Generate OpenAPI client code
task generate:api:oreilly

# Clean generated code
task clean:generated
```

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