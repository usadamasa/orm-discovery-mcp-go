# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Quick Start

### Build and Run

```bash
# Build the project
task build

# Run MCP server (stdio mode)
bin/orm-discovery-mcp-go

# Run with debug mode
ORM_MCP_GO_DEBUG=true go run .
```

### Environment Setup

```bash
# Required credentials
export OREILLY_USER_ID="your_email@example.com"
export OREILLY_PASSWORD="your_password"

# Optional
export TRANSPORT="stdio"  # or "http"
export PORT="8080"
```

### Development Workflow

```bash
task dev    # Format + Lint + Test
task ci     # Complete CI workflow (MUST pass before completion)
task check  # Quick quality check
```

## Architecture Overview

**O'Reilly Learning Platform MCP Server** - Provides programmatic access to O'Reilly content through modern browser automation and API integration.

### Core Components

| Component | Location | Role |
|-----------|----------|------|
| Browser Package | `browser/` | Authentication, API client, content retrieval |
| MCP Server | `server.go` | JSON-RPC handling, tools, resources |
| Config | `config.go` | Environment and .env loading |
| History | `research_history.go` | Research history management |

### MCP Capabilities

| Type | Name | Description |
|------|------|-------------|
| Tool | `oreilly_search_content` | Content discovery (BFS/DFS modes) |
| Tool | `oreilly_ask_question` | O'Reilly Answers AI Q&A |
| Resource | `oreilly://book-*` | Book details, TOC, chapters |
| Resource | `orm-mcp://history/*` | Research history access |

## Rules and Guides

Context-specific guidance is provided through rule files:

| Rule | Applies To | Description |
|------|------------|-------------|
| `core-architecture.md` | Always | Design, MCP capabilities, workflows |
| `development-tools.md` | `Taskfile.yml`, `aqua.yaml` | Tools, build, tasks |
| `mcp-testing-practices.md` | `*_test.go`, `server.go`, `main.go` | Testing, debugging |
| `code-quality.md` | `**.go` | QA, verification checklist |
| `browser-package-guide.md` | `browser/**` | Browser package patterns |
| `deployment-security.md` | `config.go`, `.env*`, `.github/workflows/*` | Security, environment |
| `plugin-marketplace.md` | `.claude-plugin/**`, `plugins/**` | Plugin distribution |

## Critical Requirements

1. **`task ci` must pass** before any task completion
2. **MCP testing only** - No standalone CLI commands
3. **API-first** - Browser automation for auth only
4. **Citation required** - All O'Reilly content must be properly cited
5. **Security** - Credentials in env vars only, never hardcoded

## Project Structure

```
orm-discovery-mcp-go/
├── main.go              # Entry point
├── server.go            # MCP server implementation
├── config.go            # Configuration management
├── research_history.go  # History management
├── browser/             # Browser automation & API client
│   ├── auth.go          # Authentication (ChromeDP, ACM IdP)
│   ├── search.go        # Search API
│   ├── book.go          # Book content retrieval
│   ├── types.go         # Type definitions
│   └── cookie/          # Cookie management
├── .claude/rules/       # Context-specific rules
├── .claude-plugin/      # Plugin configuration
└── plugins/agents/      # Agent definitions
```

## Cross-Package References

- **Browser Details**: See `browser/CLAUDE.md` for implementation patterns
- **ChromeDP Lifecycle**: See `.claude/skills/chromedp-lifecycle.md`
- **Plugin Structure**: See `.claude/rules/plugin-marketplace.md`

## Future Development

### API Expansion Opportunities

- Enhanced search filters and pagination
- User-specific content access (playlists, bookmarks)
- In-book search functionality
- Content summarization features
