# orm-discovery-mcp-go
Inspired by [odewahn/orm-discovery-mcp](https://github.com/odewahn/orm-discovery-mcp),
this project provides an example of how to build a MCP server using the mcp-go package.

# Disclaimer
The developers and contributors of this tool shall not be liable for
any damages, losses, or disadvantages arising from the use of this tool.
This includes but is not limited to:

- Data loss or corruption
- System downtime or interruption
- Third-party rights infringement
- Financial losses
- Any other direct or indirect damages

This tool is provided "AS IS" without a warranty of any kind, either express or implied.
Users shall use this tool at their own risk.

# Usage
Set Environment Variables
```bash
export PORT=8080
export OREILLY_API_KEY=YOUR_API_KEY
export TRANSPORT=http
```

Run Server.
```bash
go run .
```

In another terminal, you can see `tools/list` of the server.
```bash
$ curl -X POST -H "Content-Type: application/json" http://localhost:8080/mcp -d '
{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
}' | jq .
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "annotations": {
          "readOnlyHint": false,
          "destructiveHint": true,
          "idempotentHint": false,
          "openWorldHint": true
        },
        "description": "Search content on O'Reilly Learning Platform",
        "inputSchema": {
          "properties": {
            "query": {
              "description": "The search query to find content on O'Reilly Learning Platform",
              "type": "string"
            }
          },
          "required": [
            "query"
          ],
          "type": "object"
        },
        "name": "search_content"
      }
    ]
  }
}
```

And you can search content using the `search_content` tool.
```bash
$ curl -X POST "http://localhost:8080/mcp" -H "Content-Type: application/json" -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "search_content",
        "arguments": {
          "query": "golang",
          "limit": 10
        }
    },
    "id": 2
  }' | jq .
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "SEARCH RESPONSE"
      }
    ]
  }
}
```

# On Claude Desktop

```json
{
  "mcpServers": {
    "orm-discovery-mcp-go": {
      "command": "/your/path/to/orm-discovery-mcp-go",
      "args": [],
      "env": {
        "ORM_JWT": "YOUR_API_KEY"
      }
    }
  }
}
```