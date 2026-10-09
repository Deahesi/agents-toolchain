package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/anthropics/anthropic-sdk-go"
)

func GetProviderConfig(ctx context.Context, config *domain.Agent) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("agent config is required")
	}
	if config.MaxOutputTokens != nil && *config.MaxOutputTokens <= 0 {
		return nil, errors.New("max_output_tokens must be greater than zero")
	}
	if config.Think != nil && domain.ProviderKey(config.Provider) != domain.Ollama {
		return nil, errors.New("think is only supported for the ollama provider")
	}
	if config.Reasoning != nil && config.Think != nil && *config.Reasoning != *config.Think {
		return nil, errors.New("reasoning and think must agree when both are set")
	}

	result := map[string]any{"temperature": config.Temperature}
	var tokenKey string

	switch domain.ProviderKey(config.Provider) {
	case domain.OpenAI:
		tokenKey = "max_completion_tokens"
		if config.Reasoning != nil {
			result["reasoning_effort"] = "none"
			if *config.Reasoning {
				result["reasoning_effort"] = "medium"
				delete(result, "temperature")
			}
		}

		if hasModelPrefix(config.Model, "o1", "o3", "o4-mini", "gpt-5", "gpt-6") || strings.HasPrefix(config.Model, "gpt-5.") || strings.HasPrefix(config.Model, "gpt-6.") {
			delete(result, "temperature")
		}
	case domain.Anthropic:
		result := anthropic.MessageNewParams{}
		if config.MaxOutputTokens != nil {
			result.MaxTokens = int64(*config.MaxOutputTokens)
		}
		adaptive := hasModelPrefix(config.Model,
			"claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8", "claude-sonnet-4-6",
			"claude-opus-5", "claude-sonnet-5", "claude-mythos", "claude-fable")
		if !adaptive && (config.Reasoning == nil || !*config.Reasoning) {
			result.Temperature = anthropic.Float(config.Temperature)
		}
		if config.Reasoning != nil {
			disabled := anthropic.NewThinkingConfigDisabledParam()
			result.Thinking = anthropic.ThinkingConfigParamUnion{OfDisabled: &disabled}
			if *config.Reasoning {
				if adaptive {
					thinking := anthropic.NewThinkingConfigAdaptiveParam()
					result.Thinking = anthropic.ThinkingConfigParamUnion{OfAdaptive: &thinking}
				} else {
					const budget = 1024
					if config.MaxOutputTokens != nil && *config.MaxOutputTokens <= budget {
						return nil, errors.New("anthropic reasoning requires max_output_tokens greater than 1024")
					}
					result.Thinking = anthropic.ThinkingConfigParamOfEnabled(budget)
				}
			}
		}
		return result, nil
	case domain.Ollama:
		tokenKey = "num_predict"
		reasoning := config.Reasoning
		if reasoning == nil {
			reasoning = config.Think
		}
		if reasoning != nil {
			result["think"] = *reasoning
			if hasModelPrefix(strings.Split(config.Model, ":")[0], "gpt-oss") {
				if !*reasoning {
					return nil, errors.New("gpt-oss does not support disabling reasoning")
				}
				result["think"] = "medium"
			}
		}
	case domain.OpenRouter:
		tokenKey = "maxOutputTokens"
		if config.Reasoning != nil {
			result["reasoning"] = map[string]any{"enabled": *config.Reasoning}
		}
	default:
		return nil, fmt.Errorf("unsupported model provider %q", config.Provider)
	}

	if config.MaxOutputTokens != nil {
		result[tokenKey] = *config.MaxOutputTokens
	}
	return result, nil
}

func hasModelPrefix(model string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if model == prefix || strings.HasPrefix(model, prefix+"-") {
			return true
		}
	}
	return false
}
