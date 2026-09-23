# Quick Start: Pyntra with Ollama & Qwen 3.5 9B

## ⚡ 5-Minute Setup

### Step 1: Install & Start Ollama
```bash
# Download from https://ollama.ai
ollama serve
```

### Step 2: Pull Models (in another terminal)
```bash
# Pull Qwen 3.5 9B (6.6 GB)
ollama pull qwen3.5:9b

# Optional: Pull embedding model for knowledge base
ollama pull nomic-embed-text
```

### Step 3: Verify Configuration
The `config.yaml` is already configured for Ollama:
```yaml
openai:
  base_url: http://localhost:11434/v1
  api_key: ollama
  model: qwen3.5:9b
  max_total_tokens: 32000
```

### Step 4: Run Pyntra
```bash
# From project directory
./pyntra.exe
# or on Linux/Mac
./pyntra
```

### Step 5: Open Browser
Navigate to: `http://localhost:8080`
- Password: `Root@1234` (change in config.yaml)

✅ **Done!** You're now running Pyntra with local Ollama!

---

## 🔄 Switching Models

Edit `config.yaml` and change the model name:

```yaml
openai:
  model: mistral:latest  # Or any other Ollama model
```

Then restart the application.

---

## 📊 What Was Changed

| Component | OpenAI | Ollama |
|-----------|--------|--------|
| **Base URL** | `https://api.openai.com/v1` | `http://localhost:11434/v1` |
| **Model** | `gpt-4o` | `qwen3.5:9b` |
| **API Key** | Real key | Any value (ignored) |
| **Context** | 128K tokens | 32K tokens |
| **Cost** | Per API call | Free (GPU cost) |
| **Code Changes** | ❌ None needed | ✅ Configuration only |

---

## 🚀 Performance

- **Response Speed:** 2-5 seconds per message (with GPU)
- **Tool Calling:** ✅ Full support
- **Multi-Agent:** ✅ Full support
- **Streaming:** ✅ Real-time responses
- **GPU VRAM Required:** 10-12 GB

---

## 📝 Logs to Verify

Look for these messages in console:
```
Loaded AI model from Ollama: qwen3.5:9b
Connected to http://localhost:11434/v1
Using Ollama API for chat completions
Knowledge base using nomic-embed-text
```

---

## ❌ Troubleshooting

| Problem | Solution |
|---------|----------|
| Connection refused | Ensure `ollama serve` is running |
| Model not found | Run `ollama pull qwen3.5:9b` |
| Slow responses | Check GPU memory/usage |
| Out of memory | Use smaller model or more GPU RAM |

---

## 📚 Learn More

- Full migration guide: See `OLLAMA_MIGRATION.md`
- Ollama documentation: https://ollama.ai
- Pyntra repo: https://github.com/

---

## 💡 Pro Tips

1. **Test the connection:**
   ```bash
   curl http://localhost:11434/api/tags
   ```

2. **Try different models:**
   - Fast: `mistral:7b`, `neural-chat:7b`
   - Capable: `mixtral:latest`, `yi:34b`
   - Lightweight: `mistral:7b`, `all-minilm`

3. **Monitor Ollama:**
   ```bash
   # See running models
   ollama list
   
   # Show model info
   ollama show qwen3.5:9b
   ```

4. **Use environment variables:**
   ```bash
   # Change Ollama host
   export OLLAMA_HOST=192.168.1.100:11434
   ollama serve
   ```

---

✨ **Enjoy local, private, cost-free penetration testing with Pyntra!**
