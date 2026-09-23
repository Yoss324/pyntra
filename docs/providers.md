# Inference providers

Pyntra's LLM backend speaks the **OpenAI Chat Completions** protocol, so it
works with virtually any inference provider or local model. The only special
case is **Anthropic**, which is transparently bridged to the Messages API when
`provider: claude` (or `anthropic`) is set.

Configure it under `openai:` in `config.yaml`, or from **Settings** in the web
UI. It is fully custom-configurable — set `base_url`, `api_key`, and `model` to
whatever endpoint you want.

```yaml
openai:
  provider: ollama          # preset id (or "custom"); see list below
  base_url: http://localhost:11434/v1
  api_key: ollama           # placeholder is fine for keyless local servers
  model: llama3.1           # any model the endpoint serves
  max_total_tokens: 8192
```

## Presets

`GET /api/config/providers` returns this catalogue at runtime (with the current
selection), so the Settings UI can offer a dropdown while keeping every field
editable.

| provider id  | base_url                          | notes |
|--------------|-----------------------------------|-------|
| `ollama`     | `http://localhost:11434/v1`       | Any **local Ollama** model. `api_key` can be any placeholder. `ollama pull <model>` first. |
| `openai`     | `https://api.openai.com/v1`       | Official OpenAI. |
| `anthropic`  | `https://api.anthropic.com/v1`    | Alias `claude`. Auto-bridged to the Messages API. |
| `huggingface`| `https://router.huggingface.co/v1`| Alias `hf`. Use an HF access token as `api_key`. |
| `deepseek`   | `https://api.deepseek.com/v1`     | |
| `openrouter` | `https://openrouter.ai/api/v1`    | Many models behind one endpoint. |
| `groq`       | `https://api.groq.com/openai/v1`  | |
| `together`   | `https://api.together.xyz/v1`     | |
| `mistral`    | `https://api.mistral.ai/v1`       | |
| `lmstudio`   | `http://localhost:1234/v1`        | Local LM Studio server. |
| `vllm`       | `http://localhost:8000/v1`        | Self-hosted vLLM. |
| `localai`    | `http://localhost:8080/v1`        | Self-hosted LocalAI. |
| `custom`     | *(you set it)*                    | Any other OpenAI-compatible endpoint. |

Unknown provider strings resolve to the generic OpenAI-compatible path, so a new
endpoint never needs a code change — just set the three fields.

## Local models

Any local runtime that exposes an OpenAI-compatible `/v1` endpoint works:
Ollama, LM Studio, vLLM, LocalAI, llama.cpp's server, text-generation-webui, etc.
Point `base_url` at it and set `model` to the served model id. Keyless servers
accept any `api_key` placeholder.

## Note: coding agents are clients, not providers

Claude Code, Cursor, Cline, opencode, Codex and Windsurf are **MCP clients** —
they consume Pyntra's MCP server rather than serving models to it. See
[mcp-clients.md](mcp-clients.md).
