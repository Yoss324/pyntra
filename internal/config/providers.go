package config

import "strings"

// ProviderPreset describes a known inference provider that Pyntra can talk to.
//
// Pyntra speaks the OpenAI Chat Completions protocol for every provider except
// Anthropic, which is bridged to the Messages API automatically (see
// internal/openai/claude_bridge.go). Any OpenAI-compatible endpoint therefore
// works out of the box: set openai.base_url / openai.api_key / openai.model and,
// optionally, openai.provider to one of the ids below. Unknown providers fall
// back to the generic OpenAI-compatible path, so "custom" is always valid.
type ProviderPreset struct {
	ID             string   `json:"id"`               // canonical provider id (value for openai.provider)
	Label          string   `json:"label"`            // human friendly name
	BaseURL        string   `json:"base_url"`         // suggested base_url (empty => user must supply)
	APIKeyRequired bool     `json:"api_key_required"` // whether a real API key is needed
	Local          bool     `json:"local"`            // runs on the user's own machine / network
	Anthropic      bool     `json:"anthropic"`        // uses the Anthropic Messages API bridge
	Notes          string   `json:"notes"`
	ExampleModels  []string `json:"example_models"`
	Aliases        []string `json:"aliases,omitempty"`
}

// providerPresets is the built-in catalogue. Order is display order.
var providerPresets = []ProviderPreset{
	{
		ID:             "ollama",
		Label:          "Ollama (local)",
		BaseURL:        "http://localhost:11434/v1",
		APIKeyRequired: false,
		Local:          true,
		Notes:          "Any local Ollama model. api_key can be any placeholder (e.g. \"ollama\"). Run `ollama pull <model>` first.",
		ExampleModels:  []string{"llama3.1", "qwen2.5:7b", "mistral", "deepseek-r1", "gpt-oss:20b"},
		Aliases:        []string{"local"},
	},
	{
		ID:             "openai",
		Label:          "OpenAI",
		BaseURL:        "https://api.openai.com/v1",
		APIKeyRequired: true,
		Notes:          "Official OpenAI API.",
		ExampleModels:  []string{"gpt-4o", "gpt-4o-mini", "o3-mini"},
		Aliases:        []string{"openai-compatible", "compatible", ""},
	},
	{
		ID:             "anthropic",
		Label:          "Anthropic Claude (API)",
		BaseURL:        "https://api.anthropic.com/v1",
		APIKeyRequired: true,
		Anthropic:      true,
		Notes:          "Bridged to the Anthropic Messages API automatically. Use a real Claude model id.",
		ExampleModels:  []string{"claude-opus-4-5", "claude-sonnet-4-5", "claude-haiku-4-5"},
		Aliases:        []string{"claude"},
	},
	{
		ID:             "huggingface",
		Label:          "Hugging Face Inference",
		BaseURL:        "https://router.huggingface.co/v1",
		APIKeyRequired: true,
		Notes:          "Hugging Face Inference Providers router (OpenAI-compatible). Use an HF access token as api_key.",
		ExampleModels:  []string{"meta-llama/Llama-3.3-70B-Instruct", "Qwen/Qwen2.5-72B-Instruct"},
		Aliases:        []string{"hf", "hf-inference"},
	},
	{
		ID:             "deepseek",
		Label:          "DeepSeek",
		BaseURL:        "https://api.deepseek.com/v1",
		APIKeyRequired: true,
		Notes:          "DeepSeek OpenAI-compatible API.",
		ExampleModels:  []string{"deepseek-chat", "deepseek-reasoner"},
	},
	{
		ID:             "openrouter",
		Label:          "OpenRouter",
		BaseURL:        "https://openrouter.ai/api/v1",
		APIKeyRequired: true,
		Notes:          "Aggregator that fronts many models behind one OpenAI-compatible endpoint.",
		ExampleModels:  []string{"anthropic/claude-sonnet-4.5", "meta-llama/llama-3.3-70b-instruct"},
	},
	{
		ID:             "groq",
		Label:          "Groq",
		BaseURL:        "https://api.groq.com/openai/v1",
		APIKeyRequired: true,
		Notes:          "Fast hosted inference, OpenAI-compatible.",
		ExampleModels:  []string{"llama-3.3-70b-versatile", "qwen-2.5-32b"},
	},
	{
		ID:             "together",
		Label:          "Together AI",
		BaseURL:        "https://api.together.xyz/v1",
		APIKeyRequired: true,
		Notes:          "Hosted open models, OpenAI-compatible.",
		ExampleModels:  []string{"meta-llama/Llama-3.3-70B-Instruct-Turbo"},
	},
	{
		ID:             "mistral",
		Label:          "Mistral AI",
		BaseURL:        "https://api.mistral.ai/v1",
		APIKeyRequired: true,
		Notes:          "Mistral hosted API, OpenAI-compatible.",
		ExampleModels:  []string{"mistral-large-latest", "codestral-latest"},
	},
	{
		ID:             "lmstudio",
		Label:          "LM Studio (local)",
		BaseURL:        "http://localhost:1234/v1",
		APIKeyRequired: false,
		Local:          true,
		Notes:          "Local LM Studio server. Start the built-in server and load a model.",
		ExampleModels:  []string{"<loaded-model-id>"},
		Aliases:        []string{"lm-studio"},
	},
	{
		ID:             "vllm",
		Label:          "vLLM (self-hosted)",
		BaseURL:        "http://localhost:8000/v1",
		APIKeyRequired: false,
		Local:          true,
		Notes:          "Self-hosted vLLM OpenAI-compatible server.",
		ExampleModels:  []string{"<served-model-name>"},
	},
	{
		ID:             "localai",
		Label:          "LocalAI (self-hosted)",
		BaseURL:        "http://localhost:8080/v1",
		APIKeyRequired: false,
		Local:          true,
		Notes:          "Self-hosted LocalAI OpenAI-compatible server.",
		ExampleModels:  []string{"<configured-model>"},
		Aliases:        []string{"local-ai"},
	},
	{
		ID:             "custom",
		Label:          "Custom (OpenAI-compatible)",
		BaseURL:        "",
		APIKeyRequired: false,
		Notes:          "Any other OpenAI-compatible endpoint. Set base_url/api_key/model yourself.",
		ExampleModels:  nil,
		Aliases:        []string{"openai_compatible"},
	},
}

// ProviderPresets returns a copy of the built-in provider catalogue (display order).
func ProviderPresets() []ProviderPreset {
	out := make([]ProviderPreset, len(providerPresets))
	copy(out, providerPresets)
	return out
}

// NormalizeProvider maps a user-supplied provider string (with common aliases and
// casing/whitespace variance) to a canonical provider id. Empty or unknown values
// resolve to "openai" (the generic OpenAI-compatible path), except that a value
// explicitly meaning "custom" is preserved.
func NormalizeProvider(s string) string {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" {
		return "openai"
	}
	for _, p := range providerPresets {
		if v == p.ID {
			return p.ID
		}
		for _, a := range p.Aliases {
			if a != "" && v == a {
				return p.ID
			}
		}
	}
	return "custom"
}

// ProviderIsAnthropic reports whether the given provider string resolves to the
// Anthropic Messages API bridge.
func ProviderIsAnthropic(s string) bool {
	return NormalizeProvider(s) == "anthropic"
}

// LookupProviderPreset returns the preset for a provider id or alias.
func LookupProviderPreset(s string) (ProviderPreset, bool) {
	id := NormalizeProvider(s)
	for _, p := range providerPresets {
		if p.ID == id {
			return p, true
		}
	}
	return ProviderPreset{}, false
}
