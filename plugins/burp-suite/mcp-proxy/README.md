# mcp-proxy.jar (Burp Suite MCP bridge)

This directory is where Pyntra's `burp` external-MCP preset expects PortSwigger's
`mcp-proxy.jar` — a stdio↔SSE bridge for the Burp **MCP Server** extension.

The jar is a **third-party PortSwigger binary** and is intentionally **not
committed** to this repository (it is git-ignored). Obtain it yourself:

1. In Burp: **Extensions → BApp Store → MCP Server → Install**, then open the
   **MCP Server** tab and enable it.
2. Use *"Install proxy for AI clients that only support STDIO"* to download
   `mcp-proxy.jar`, and drop it in this folder as `mcp-proxy.jar`.

Requires Java 17+ on PATH. Full setup: see [`docs/burp-mcp.md`](../../../docs/burp-mcp.md).

If you'd rather not use the jar, connect Pyntra to Burp over SSE directly —
set `transport: sse` / `url: http://127.0.0.1:9876/sse` in the `burp` preset.
