package domain

type ProviderKey string

const (
	OpenAI     ProviderKey = "openai"
	Anthropic  ProviderKey = "anthropic"
	Ollama     ProviderKey = "ollama"
	OpenRouter ProviderKey = "openrouter"
)

var Providers []ProviderKey = []ProviderKey{
	OpenAI, Anthropic, Ollama, OpenRouter,
}
