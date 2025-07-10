# Network Monitoring

## network-monitor Usage

- `network-monitor` is a tool for observing and analyzing network-related activities in the project
- Can be used to debug API calls, track network requests, and diagnose connectivity issues
- Potential use cases include monitoring O'Reilly API interactions, browser authentication flows, and HTTP/HTTPS traffic

```shell
# playwright-min-network-mcp
claude mcp add -s user network-monitor \
    -- npx -y playwright-min-network-mcp

# playwright
claude mcp add -s user playwright \
    -- npx -y @playwright/mcp --cdp-endpoint http://localhost:9222

# install playwright cmd
npx -g -y install playwright
```

## Ref
- [bun913/playwright-min-network-mcp](https://github.com/bun913/playwright-min-network-mcp)
