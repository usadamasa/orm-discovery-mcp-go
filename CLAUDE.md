# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

This is an **O'Reilly Learning Platform MCP (Model Context Protocol) Server** that provides programmatic access to O'Reilly content through browser automation. The architecture consists of several key components:

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
   - Exposes 12 MCP tools for content operations
   - Handles JSON-RPC request/response mapping
   - Supports both stdio and HTTP transport modes

4. **Config** (`config.go`) - Configuration management
   - Loads settings from `.env` files and environment variables
   - Handles executable-relative .env file discovery

### Key Design Patterns

**Browser-First Approach**: This system doesn't use traditional REST APIs. Instead, it automates the actual O'Reilly Learning Platform web interface using a headless browser. This is because O'Reilly doesn't provide public APIs.

**Cookie-Based Authentication**: The system extracts JWT tokens (`orm-jwt`), session IDs (`groot_sessionid`), and refresh tokens (`orm-rt`) from browser sessions to maintain authentication state.

**DOM Scraping with JavaScript**: Content extraction relies on JavaScript execution within the browser context to query DOM elements and extract structured data from the web interface.

### Available MCP Tools

- `search_content` - Content discovery and search
- `list_collections` - Homepage collection enumeration
- `summarize_books` - Multi-book analysis with Japanese summaries
- `list_playlists`, `create_playlist`, `add_to_playlist`, `get_playlist_details` - Playlist management
- `extract_table_of_contents` - Book structure extraction
- `search_in_book` - In-book content search

## Environment Setup

### Required Environment Variables
```bash
OREILLY_USER_ID=your_email@example.com    # Your O'Reilly account email
OREILLY_PASSWORD=your_password             # Your O'Reilly password
PORT=8080                                  # HTTP server port (optional)
TRANSPORT=stdio                            # Transport mode: stdio or http
```

### .env File Support
Place a `.env` file in the same directory as the executable. The system will automatically detect and load it, with `.env` values taking precedence over environment variables.

## Dependencies

**Core Framework**: `github.com/mark3labs/mcp-go` - MCP protocol implementation
**Browser Automation**: `github.com/chromedp/chromedp` - Chrome DevTools Protocol
**Configuration**: `github.com/joho/godotenv` - Environment file loading

## Browser Requirements

This application requires Chrome or Chromium to be installed on the system for headless browser operations. The browser is used to:
- Authenticate with O'Reilly Learning Platform
- Navigate and scrape content from web pages
- Execute JavaScript for DOM manipulation
- Handle complex authentication flows (including ACM IdP redirects)

## File Organization

- `main.go` (225 lines) - Entry point with test modes
- `server.go` (815 lines) - MCP server and tool handlers
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
- The browser automation may be slower than direct API calls but provides access to content not available through public APIs
- ACM (Association for Computing Machinery) institutional login is automatically detected and handled