# Browser Package - O'Reilly API Implementation Guide

This file provides guidance for working with the browser package, which implements O'Reilly Learning Platform integration using a hybrid approach.

## Architecture Overview

**Current Implementation (Post-Refactoring)**:
```
Browser(Auth Only) → Go HTTP Client → O'Reilly Internal API → Structured JSON → Auto-Normalization
```

**Key Improvements**:
- JavaScript reduced from 340+ lines to 1 line (99.7% reduction)
- Direct HTTP API calls instead of DOM manipulation
- Type-safe Go struct processing
- Multiple endpoint fallback strategy

## Package Structure

```
browser/
├── types.go      # Type definitions and constants
├── auth.go       # Authentication and session management  
├── search.go     # Search API implementation
└── operations.go # Other operations (collections, playlists, TOC)
```

## Core Components

### 1. Authentication (auth.go)

**Primary Functions**:
- `NewBrowserClient()` - Creates and initializes browser client with login
- `login()` - Handles O'Reilly login with ACM IdP support
- `RefreshSession()` - Updates authentication cookies
- Cookie/JWT management functions

**Login Flow**:
1. Navigate to `https://www.oreilly.com/member/login/`
2. Input email and handle Continue button
3. Detect ACM IdP redirect if applicable
4. Establish session at `https://learning.oreilly.com/`
5. Extract authentication cookies

### 2. Search Implementation (search.go)

**Internal API Endpoints** (with fallback):
```go
const (
    APIEndpointV2       = "/api/v2/search/"           // Primary
    APIEndpointSearch   = "/search/api/search/"       // Fallback 1
    APIEndpointLegacy   = "/api/search/"              // Fallback 2
    APIEndpointLearning = "/learningapi/v1/search/"   // Fallback 3
)
```

**Key Functions**:
- `SearchContent()` - Main search implementation
- `makeHTTPSearchRequest()` - Direct HTTP API calls
- `normalizeSearchResult()` - Go-based result processing

**API Parameters**:
- `q` (required): Search query
- `rows` (optional): Result count (default: 100)
- `tzOffset` (optional): Timezone offset (default: -9 for JST)
- `aia_only` (optional): AI-assisted content only (default: false)
- `feature_flags` (optional): Feature flags (default: "improveSearchFilters")
- `report` (optional): Include report data (default: true)
- `isTopics` (optional): Topics search only (default: false)

### 3. Type Definitions (types.go)

**Core Structures**:
```go
type SearchAPIResponse struct {
    Data    *SearchDataContainer `json:"data,omitempty"`
    Results []RawSearchResult   `json:"results,omitempty"`
    Items   []RawSearchResult   `json:"items,omitempty"`
    Hits    []RawSearchResult   `json:"hits,omitempty"`
}

type RawSearchResult struct {
    ID                     string   `json:"id,omitempty"`
    ProductID              string   `json:"product_id,omitempty"`
    Title                  string   `json:"title,omitempty"`
    WebURL                 string   `json:"web_url,omitempty"`
    Authors                []string `json:"authors,omitempty"`
    ContentType            string   `json:"content_type,omitempty"`
    Description            string   `json:"description,omitempty"`
    Publisher              string   `json:"publisher,omitempty"`
    PublishedDate          string   `json:"published_date,omitempty"`
    // ... 15+ additional fields
}
```

### 4. Operations (operations.go)

**Implemented**:
- `GetCollectionsFromHomePage()` - Extract homepage collections

**Placeholder (for future implementation)**:
- `GetPlaylistsFromPlaylistsPage()` - List user playlists
- `CreatePlaylist()` - Create new playlists
- `AddContentToPlaylist()` - Add content to playlists
- `GetPlaylistDetails()` - Get playlist details
- `ExtractTableOfContents()` - Extract book TOC
- `SearchInBook()` - In-book content search

## Implementation Patterns

### HTTP API Client Pattern
```go
// 1. Get authentication cookies from browser
err := chromedp.Run(bc.ctx,
    chromedp.Navigate("https://learning.oreilly.com/search/"),
    chromedp.ActionFunc(func(ctx context.Context) error {
        // Update cookies from browser context
        cookiesResp, _ := network.GetCookies().Do(ctx)
        bc.cookies = convertCookies(cookiesResp)
        return nil
    }),
)

// 2. Make direct HTTP API calls with Go client
for _, endpoint := range endpoints {
    apiResponse, err := bc.makeHTTPSearchRequest(endpoint, query, ...)
    if err == nil {
        // Success - normalize and return results
        break
    }
}
```

### Result Normalization Pattern
```go
func normalizeSearchResult(raw RawSearchResult, index int) map[string]interface{} {
    // URL normalization
    itemURL := raw.WebURL
    if itemURL == "" && raw.ProductID != "" {
        itemURL = "https://learning.oreilly.com/library/view/-/" + raw.ProductID + "/"
    }
    
    // Author normalization
    var authors []string
    if len(raw.Authors) > 0 {
        authors = raw.Authors
    } else if raw.Author != "" {
        authors = []string{raw.Author}
    }
    
    return map[string]interface{}{
        "id":             raw.ProductID,
        "title":          raw.Title,
        "authors":        authors,
        "content_type":   raw.ContentType,
        "url":            itemURL,
        "source":         "api_search_oreilly",
    }
}
```

### Minimal JavaScript Pattern
```javascript
// Only used for browser context - domain name extraction
window.location.hostname
```

## Error Handling & Fallbacks

### 1. Multiple API Endpoint Strategy
- Try primary endpoint first (`/api/v2/search/`)
- Automatic fallback to secondary endpoints
- Detailed logging for each attempt
- Fail fast with appropriate error messages

### 2. Authentication Resilience
- Browser session re-establishment
- Cookie validation and refresh
- ACM IdP flow handling
- Multiple URL access attempts

### 3. HTTP Client Error Handling
- Status code specific responses
- Timeout configuration (30 seconds)
- JSON parsing error recovery
- Comprehensive error logging

## Environment Requirements

### Required Environment Variables
```bash
OREILLY_USER_ID=your_email@example.com
OREILLY_PASSWORD=your_password
PORT=8080
TRANSPORT=stdio
```

### Browser Dependencies
- Chrome or Chromium installation
- Headless mode support
- JavaScript execution environment

## Performance Optimizations

### 1. Go HTTP Client Benefits
- **JavaScript execution reduction**: 340 lines → 1 line
- **Direct API access**: No DOM parsing overhead
- **Type-safe processing**: Native Go struct handling
- **Parallel processing**: Multiple endpoint attempts

### 2. Resource Optimization
- **Minimal browser usage**: Authentication and cookie extraction only
- **Context efficiency**: Reduced page navigation
- **Memory reduction**: No large DOM element loading

### 3. API Strategy Benefits
- **Primary/fallback configuration**: High availability
- **Response time improvement**: Direct internal API access
- **Fast error recovery**: Quick fallback execution

## Security Considerations

### 1. Authentication Management
- Environment variable credential storage
- Session timeout handling
- Cookie security validation

### 2. Rate Limiting
- Request interval management
- Concurrent request limits
- Platform compliance

### 3. Data Privacy
- Sensitive information masking in logs
- Secure network communication
- Temporary file cleanup

## Development Guidelines

### 1. Adding New Functionality
- Follow the established pattern of minimal browser usage
- Implement Go HTTP clients for new API endpoints
- Use structured types for response handling
- Add appropriate fallback mechanisms

### 2. Debugging
- Use comprehensive logging throughout
- Browser network monitoring for API discovery
- HTTP response inspection
- Cookie/session state validation

### 3. Testing
- Mock HTTP responses for unit tests
- Browser automation integration tests
- Error scenario validation
- Performance benchmarking

## Future Considerations

### 1. API Endpoint Evolution
- Monitor O'Reilly internal API changes
- Maintain multiple endpoint fallbacks
- Update Go structs for new response formats

### 2. Authentication Flow Changes
- ACM IdP integration updates
- New institutional login support
- OAuth/SSO integration possibilities

### 3. Feature Extensions
- Additional content type support
- Advanced search filters
- User-specific content access
- Real-time content updates

## Key Benefits of Current Implementation

| Aspect | Improvement |
|--------|-------------|
| **JavaScript Dependency** | 99.7% reduction (340+ lines → 1 line) |
| **Maintainability** | Single large file → 4 focused modules |
| **Performance** | DOM parsing → Direct API calls |
| **Reliability** | Single point of failure → Multi-endpoint fallback |
| **Type Safety** | Dynamic extraction → Static Go structs |

This implementation represents a significant evolution from DOM-based scraping to API-first integration, providing better performance, reliability, and maintainability while minimizing JavaScript dependencies.