# Changelog

## [v0.0.10](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.9...v0.0.10) - 2026-02-08
- feat: HistoryResource系のログレベルをDebugからInfoに変更 by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/58
- feat: oreilly-researcherエージェントにagent memory (userスコープ) を追加 by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/62
- feat: go install時にもReadBuildInfoでバージョン情報を表示する by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/63
- build(deps): bump github.com/stretchr/testify from 1.10.0 to 1.11.1 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/60

## [v0.0.9](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.8...v0.0.9) - 2026-01-31
- feat: add MCP prompts support with three prompts by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/45
- feat: add research history feature with MCP resources and prompts by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/49
- feat: add MCP Sampling support with BFS/DFS search modes by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/50
- feat: add marketplace.json for Claude Code plugin marketplace distribution by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/51
- fix: relocate agent file to plugins/ directory for correct path resolution by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/52
- refactor: split CLAUDE.md into 7 context-specific rule files by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/53
- refactor: shorten MCP server name from orm-discovery-mcp-go to orm by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/54
- docs: add version update trigger rules for plugin marketplace by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/55
- fix: remove invalid plugin.json fields causing validation error by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/56
- feat: add sampling capability check before MCP sampling calls by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/57

## [v0.0.8](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.7...v0.0.8) - 2026-01-25
- feat: add XDG Base Directory support by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/32
- fix: avoid context canceled error when saving cookies after login by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/34
- feat: migrate to modelcontextprotocol/go-sdk with StructuredContent support by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/35
- feat: implement E2E tests with TestMain shared client optimization by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/36
- chore: improve Taskfile.yml with install task and cleanup by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/37
- build(deps): bump golang.org/x/net from 0.48.0 to 0.49.0 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/28
- feat: apply progressive disclosure pattern to MCP tool descriptions by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/38
- fix: initialize empty slices in convertAnswerData to prevent MCP validation errors by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/39
- feat: embed version info in task install with clean build by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/40
- Release for v0.0.8 by @usadamasa-tagpr[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/33

## [v0.0.8](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.7...v0.0.8) - 2026-01-25
- feat: add XDG Base Directory support by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/32
- fix: avoid context canceled error when saving cookies after login by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/34
- feat: migrate to modelcontextprotocol/go-sdk with StructuredContent support by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/35
- feat: implement E2E tests with TestMain shared client optimization by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/36
- chore: improve Taskfile.yml with install task and cleanup by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/37
- build(deps): bump golang.org/x/net from 0.48.0 to 0.49.0 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/28
- feat: apply progressive disclosure pattern to MCP tool descriptions by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/38
- fix: initialize empty slices in convertAnswerData to prevent MCP validation errors by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/39
- feat: embed version info in task install with clean build by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/40

## [v0.0.7](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.6...v0.0.7) - 2026-01-24
- feat: add comprehensive timeouts to browser and API operations by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/29
- fix: resolve Chrome SingletonLock error by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/31

## [v0.0.6](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.5...v0.0.6) - 2026-01-11
- Fix/disposable chrome by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/26

## [v0.0.5](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.4...v0.0.5) - 2025-12-30
- fix goreleaser config by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/24

## [v0.0.4](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.3...v0.0.4) - 2025-12-30
- build(deps): bump github.com/mark3labs/mcp-go from 0.32.0 to 0.43.2 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/9
- build(deps): bump golang.org/x/net from 0.41.0 to 0.48.0 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/10
- build(deps): bump github.com/chromedp/chromedp from 0.13.7 to 0.14.2 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/5
- build(deps): bump github.com/oapi-codegen/runtime from 1.1.1 to 1.1.2 by @dependabot[bot] in https://github.com/usadamasa/orm-discovery-mcp-go/pull/4

## [Unreleased]

### Changed
- Upgrade mcp-go from v0.32.0 to v0.43.2
  - Enhanced session management capabilities
  - Improved HTTP client features with custom header support
  - Fixed notification issues affecting client tool calls
  - Improved JSON schema unmarshaling (supports both $defs and definitions)
  - No breaking changes - full backward compatibility maintained

## [v0.0.3](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.2...v0.0.3) - 2025-12-30
- handle terminate headless chrome by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/20
- add unit test to auth.go by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/22

## [v0.0.2](https://github.com/usadamasa/orm-discovery-mcp-go/compare/v0.0.1...v0.0.2) - 2025-12-30
- fix by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/16
- fix by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/18

## [v0.0.1](https://github.com/usadamasa/orm-discovery-mcp-go/commits/v0.0.1) - 2025-12-30
- Fix/login flow by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/1
- Setup ci by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/2
- add dependabot and other by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/3
- fix by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/8
- introduce tagpr and goreleaser by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/12
- fix permission by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/13
- fix variable by @usadamasa in https://github.com/usadamasa/orm-discovery-mcp-go/pull/14
