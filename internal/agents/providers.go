package agents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/anthropic"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/compat_oai/openrouter"
	"github.com/firebase/genkit/go/plugins/ollama"
	"github.com/openai/openai-go/option"
)

// TODO: при пополнении провайдеров разместить их по модулями
func GetProviderPlugin(ctx context.Context, provider string) (genkit.GenkitOption, error) {
	var plugin api.Plugin

	switch domain.ProviderKey(provider) {
	case domain.OpenAI:
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, errors.New("OPENAI_API_KEY is required for openai models")
		}
		client := &openai.OpenAI{
			APIKey: key,
		}
		if endpoint := os.Getenv("OPENAI_BASE_URL"); endpoint != "" {
			client.Opts = append(client.Opts, option.WithBaseURL(endpoint))
		}
		plugin = client
	case domain.Anthropic:
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" && os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" {
			return nil, errors.New("ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN is required for anthropic models")
		}
		plugin = &anthropic.Anthropic{
			APIKey: key,
		}
	case domain.Ollama:
		endpoint := os.Getenv("OLLAMA_HOST")
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		} else if !strings.Contains(endpoint, "://") {
			endpoint = "http://" + endpoint
		}
		plugin = &ollama.Ollama{
			ServerAddress: strings.TrimRight(endpoint, "/"),
		}
	case domain.OpenRouter:
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return nil, errors.New("OPENROUTER_API_KEY is required for openrouter models")
		}
		plugin = &openrouter.OpenRouter{
			APIKey: key,
		}
	default:
		return nil, fmt.Errorf("unsupported model provider %q", provider)
	}

	return genkit.WithPlugins(plugin), nil
}
