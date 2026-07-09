package aiconfig

// ProviderPreset is a named starting point for a profile editor.
type ProviderPreset struct {
	Name     string
	Provider Provider
	BaseURL  string
	Notes    string
}

var providerPresets = []ProviderPreset{
	{
		Name:     "Ollama (local)",
		Provider: ProviderOllama,
		BaseURL:  DefaultOllamaBaseURL,
		Notes:    "local-first; no API key required",
	},
	{
		Name:     "OpenRouter",
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://openrouter.ai/api/v1",
		Notes:    "OpenAI-compatible; bearer key from openrouter.ai/keys",
	},
	{
		Name:     "GitHub Models",
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://models.github.ai/inference",
		Notes:    "OpenAI-compatible; use a GitHub PAT as the bearer key",
	},
	{
		Name:     "Groq",
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://api.groq.com/openai/v1",
		Notes:    "OpenAI-compatible; bearer key from console.groq.com",
	},
	{
		Name:     "Together",
		Provider: ProviderOpenAICompatible,
		BaseURL:  "https://api.together.xyz/v1",
		Notes:    "OpenAI-compatible; bearer key from api.together.ai",
	},
	{
		Name:     "Anthropic API",
		Provider: ProviderAnthropic,
		BaseURL:  "https://api.anthropic.com",
		Notes:    "direct /v1/messages; x-api-key from console.anthropic.com",
	},
	{
		Name:     "Gemini",
		Provider: ProviderGemini,
		BaseURL:  "https://generativelanguage.googleapis.com",
		Notes:    "Google AI; API key from aistudio.google.com",
	},
}

// ProviderPresets returns a copy of the built-in provider presets.
func ProviderPresets() []ProviderPreset {
	out := make([]ProviderPreset, len(providerPresets))
	copy(out, providerPresets)
	return out
}
