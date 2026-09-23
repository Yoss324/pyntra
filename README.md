<div align="center">
  <img src="web/static/logo.svg" alt="Pyntra" width="96" height="96">

  <h1>Pyntra</h1>

  <p><strong>An AI-native security testing platform.</strong><br>
  Conversational commands in, vulnerabilities, attack chains, and reports out.</p>

  <p>
    <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white">
    <img alt="License" src="https://img.shields.io/badge/License-Apache_2.0-blue.svg">
    <img alt="Platform" src="https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey">
    <img alt="LLM" src="https://img.shields.io/badge/LLM-OpenAI%20%7C%20Ollama%20%7C%20Claude-8A2BE2">
    <img alt="PRs" src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg">
  </p>
</div>

---

Pyntra drives AI agents through the full offensive-security lifecycle. It orchestrates 100+ security tools over the Model Context Protocol (MCP), reasons about findings, builds attack chains, retrieves from a security knowledge base, and keeps every step auditable — all from a single web console or a chat message.

> ⚠️ **Authorized use only.** Pyntra is for penetration testing and security research on systems you **own or are explicitly authorized to test**. You are responsible for complying with all applicable laws and rules of engagement.

## ✨ Highlights

- 🧠 **AI orchestration** — single-agent (ReAct) and multi-agent modes (Deep, Plan-Execute, Supervisor) via the [CloudWeGo Eino](https://github.com/cloudwego/eino) framework.
- 🔌 **Native MCP** — a built-in MCP server plus first-class integration of external MCP tool servers.
- 🎭 **Role-based testing** — predefined security roles with scoped prompts and restricted tool access.
- 🧩 **Skills system** — modular skill packages for domains like SQLi, XSS, and API security.
- 📚 **Knowledge base (RAG)** — vector retrieval over your own security knowledge, with local-embedding support.
- 🕸️ **Attack-chain graphing** — visualize, score, and replay testing sequences.
- 🐚 **Vulnerability & WebShell management** — track findings and manage remote sessions from the console.
- 🌗 **Modern web console** — clean dashboard with light/dark themes.
- 💬 **Chat access** — optional Telegram, Slack, and Discord bots for testing on the go.
- 🏠 **Runs fully local** — point it at [Ollama](https://ollama.com) for an offline, self-hosted setup, or use any OpenAI-compatible / Anthropic Claude endpoint.

## 🚀 Quick start

**Prerequisites:** [Go 1.25+](https://go.dev/dl/), and an LLM endpoint (a local [Ollama](https://ollama.com) install, or an OpenAI-compatible / Claude API key).

```bash
# 1. Clone
git clone https://github.com/prnvv2/pyntra.git
cd pyntra

# 2. Configure — set your LLM endpoint and CHANGE the default password
#    (edit config.yaml, or use the Settings page after first launch)

# 3. Build & run
go build -o pyntra ./cmd/server
./pyntra
```

Open the console at **http://localhost:8080**.

> 🏠 For a fully offline setup with Ollama, see **[OLLAMA_QUICKSTART.md](OLLAMA_QUICKSTART.md)**.

## ⚙️ Configuration

All settings live in [`config.yaml`](config.yaml) and can also be edited from the **Settings** page in the web console.

| Section | What it controls |
|---------|------------------|
| `server` | Listen host and port |
| `auth` | Web login password and session length — **change `Root@1234` before exposing the app** |
| `openai` | LLM provider, base URL, model (OpenAI-compatible, Ollama, or Claude via `provider: claude`) |
| `knowledge` | Embedding model and RAG settings |
| `bots` | Telegram / Slack / Discord chat bots |
| `mcp` | Built-in MCP server and external MCP tool servers |

## 🧱 Architecture

```
┌──────────────────────────────────────────────────────────┐
│  Web Console (SPA)                                         │
├──────────────────────────────────────────────────────────┤
│  HTTP + WebSocket API (Gin)                                │
├──────────────────────────────────────────────────────────┤
│  Agent orchestration (Eino)  ·  single & multi-agent       │
│  Roles · Skills · Knowledge (RAG) · Attack-chain builder   │
├──────────────────────────────────────────────────────────┤
│  MCP layer  ── built-in server + external tool servers     │
├──────────────────────────────────────────────────────────┤
│  SQLite  ·  conversations, vulns, webshells, attack chains │
└──────────────────────────────────────────────────────────┘
```

**Stack:** Go · Gin · Gorilla WebSocket · CloudWeGo Eino · Model Context Protocol · modernc SQLite · Zap.

## 🗺️ Roadmap

- [ ] Redesigned dashboard and views on the new design system
- [ ] Scope / authorization guardrails (target allowlist, engagement scope)
- [ ] Multi-user accounts, roles, and audit log
- [ ] Exportable pentest report generator (PDF / HTML)
- [ ] Local model eval & benchmark panel

## 🤝 Contributing

Issues and pull requests are welcome. Please keep contributions focused and include a clear description of the change.

## 📄 License

Licensed under the **Apache License 2.0** — see [LICENSE](LICENSE).

Pyntra is a derivative work based on an upstream Apache-2.0 project; see [NOTICE](NOTICE) for attribution and a summary of modifications.
