# Burp Suite MCP integration

Pyntra can drive **Burp Suite** through the Model Context Protocol (MCP). Once
connected, Pyntra's agents can list and call Burp's tools — send requests via
Repeater, read Proxy history, inspect the site map, launch Scanner/Intruder,
and more — as part of an authorized engagement.

This uses PortSwigger's official **MCP Server** Burp extension plus its
`mcp-proxy.jar` stdio bridge. Burp is registered as an *external MCP server* in
Pyntra (see `external_mcp.servers.burp` in `config.yaml`).

> Only use this against systems you are explicitly authorized to test.

## 1. Enable the MCP Server in Burp

1. In Burp: **Extensions → BApp Store → MCP Server → Install**.
2. Open the new **MCP Server** tab and **enable** the server. It listens on
   `http://127.0.0.1:9876` by default (SSE endpoint `http://127.0.0.1:9876/sse`).

## 2. Choose how Pyntra connects

### Option A — stdio via `mcp-proxy.jar` (default preset)

Requires **Java** on PATH (`java -version`).

1. In Burp's **MCP Server** tab, use *"Install proxy for AI clients that only
   support STDIO"* to obtain `mcp-proxy.jar`.
2. Place it at `plugins/burp-suite/mcp-proxy/mcp-proxy.jar` (relative to the
   Pyntra working directory). The jar is a third-party PortSwigger binary and is
   **git-ignored** — it is not redistributed with Pyntra.
3. The `burp` preset in `config.yaml` already points at it:

   ```yaml
   external_mcp:
     servers:
       burp:
         transport: stdio
         command: java
         args: ["-jar", "plugins/burp-suite/mcp-proxy/mcp-proxy.jar", "--sse-url", "http://127.0.0.1:9876/sse"]
         external_mcp_enable: false
   ```

### Option B — direct SSE (no Java, no jar)

If your Pyntra build's MCP client can speak SSE directly, skip the jar:

```yaml
external_mcp:
  servers:
    burp:
      transport: sse
      url: http://127.0.0.1:9876/sse
      external_mcp_enable: true
```

## 3. Turn it on

- **UI:** Settings → External MCP → `burp` → **Start**. The tool count should
  become non-zero once connected.
- **Config:** set `external_mcp_enable: true` and restart, or call
  `POST /api/external-mcp/burp/start`.

Burp's tools then appear to the agent namespaced as `burp::<tool>` and can be
allow-listed per role via `roles/*.yaml` `tools:` entries (e.g. `burp::send_http_request`).

## Troubleshooting

- **`connection refused` / 0 tools:** the Burp MCP Server isn't enabled, or the
  SSE URL/port doesn't match Burp's tab.
- **`java: command not found`:** install a JRE/JDK (Java 17+), or use Option B.
- **Wrong jar path:** paths are relative to Pyntra's working directory; use an
  absolute path in `args` if unsure.
