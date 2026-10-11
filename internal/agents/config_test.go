package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// Exercise the actual provider adapters: accepting a map in a fake model does
// not prove that Genkit translates its keys into the outgoing API request.
func TestProviderConfigRequests(t *testing.T) {
	for _, tc := range []struct {
		provider string
		model    string
		tokenKey string
	}{
		{"openai", "gpt-5.4", "max_completion_tokens"},
		{"anthropic", "claude-sonnet-4-5", "max_tokens"},
		{"anthropic", "claude-sonnet-4-6", "max_tokens"},
		{"ollama", "qwen3:latest", "num_predict"},
		{"ollama", "gpt-oss:20b", "num_predict"},
		{"openrouter", "openai/gpt-5.4", "max_tokens"},
	} {
		for _, state := range []string{"default", "enabled", "disabled"} {
			if tc.model == "gpt-oss:20b" && state == "disabled" {
				continue // Rejected locally; covered below.
			}
			t.Run(tc.provider+"/"+tc.model+"/"+state, func(t *testing.T) {
				requests := make(chan map[string]any, 1)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/api/show" {
						fmt.Fprint(w, `{"capabilities":["completion","thinking"]}`)
						return
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode request: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					requests <- body
					switch tc.provider {
					case "anthropic":
						fmt.Fprint(w, `{"id":"msg_test","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
					case "ollama":
						fmt.Fprint(w, `{"model":"test","message":{"role":"assistant","content":"ok"},"done":true,"done_reason":"stop"}`)
					default:
						fmt.Fprint(w, `{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
					}
				}))
				defer server.Close()
				t.Setenv("OPENAI_API_KEY", "test-key")
				t.Setenv("OPENAI_BASE_URL", server.URL+"/v1")
				t.Setenv("ANTHROPIC_API_KEY", "test-key")
				t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
				t.Setenv("ANTHROPIC_BASE_URL", server.URL)
				t.Setenv("OPENROUTER_API_KEY", "test-key")
				t.Setenv("OPENROUTER_BASE_URL", server.URL+"/v1")
				t.Setenv("OLLAMA_HOST", server.URL)

				limit := 2048
				agent := &domain.Agent{Provider: tc.provider, Model: tc.model, Temperature: 0.2, MaxOutputTokens: &limit}
				if state != "default" {
					enabled := state == "enabled"
					agent.Reasoning = &enabled
				}
				ctx := context.Background()
				config, err := GetProviderConfig(ctx, agent)
				if err != nil {
					t.Fatal(err)
				}
				plugin, err := GetProviderPlugin(ctx, tc.provider)
				if err != nil {
					t.Fatal(err)
				}
				g := genkit.Init(ctx, plugin)
				if _, err := genkit.Generate(ctx, g, ai.WithModelName(tc.provider+"/"+tc.model), ai.WithConfig(config), ai.WithPrompt("test")); err != nil {
					t.Fatal(err)
				}
				var body map[string]any
				select {
				case body = <-requests:
				default:
					t.Fatal("no provider request received")
				}
				options := body
				if tc.provider == "ollama" {
					options = body["options"].(map[string]any)
				}
				if options[tc.tokenKey] != float64(limit) {
					t.Errorf("token limit missing from outgoing request: %v", body)
				}
				omitTemperature := tc.provider == "openai" || (tc.provider == "anthropic" && (tc.model == "claude-sonnet-4-6" || state == "enabled"))
				temperature, present := options["temperature"]
				if omitTemperature && present {
					t.Errorf("incompatible temperature in request: %v", body)
				} else if !omitTemperature && temperature != agent.Temperature {
					t.Errorf("temperature = %v, want %v", temperature, agent.Temperature)
				}
				for _, key := range []string{"max_output_tokens", "maxOutputTokens"} {
					if _, ok := options[key]; ok {
						t.Errorf("untranslated key %q in request", key)
					}
				}
				var key string
				var want any
				switch tc.provider {
				case "openai":
					key = "reasoning_effort"
					want = "none"
					if state == "enabled" {
						want = "medium"
					}
				case "anthropic":
					key = "thinking"
					want = map[string]any{"type": "disabled"}
					if state == "enabled" {
						want = map[string]any{"type": "enabled", "budget_tokens": float64(1024)}
						if tc.model == "claude-sonnet-4-6" {
							want = map[string]any{"type": "adaptive"}
						}
					}
				case "ollama":
					key = "think"
					want = state == "enabled"
					if tc.model == "gpt-oss:20b" {
						want = "medium"
					}
				case "openrouter":
					key = "reasoning"
					want = map[string]any{"enabled": state == "enabled"}
				}
				value, present := body[key]
				if state == "default" {
					if present {
						t.Errorf("default reasoning must be omitted: %v", body)
					}
				} else if !reflect.DeepEqual(value, want) {
					t.Errorf("%s = %v, want %v", key, value, want)
				}
			})
		}
	}
}

func TestProviderConfigValidationAndLegacyThink(t *testing.T) {
	disabled, enabled := false, true
	zero, small := 0, 1024
	for _, tc := range []struct {
		name  string
		agent *domain.Agent
		error string
	}{
		{"nil agent", nil, "agent config is required"},
		{"unknown provider", &domain.Agent{Provider: "unknown"}, "unsupported model provider"},
		{"invalid limit", &domain.Agent{Provider: "ollama", MaxOutputTokens: &zero}, "greater than zero"},
		{"conflicting flags", &domain.Agent{Provider: "ollama", Reasoning: &enabled, Think: &disabled}, "must agree"},
		{"think on other provider", &domain.Agent{Provider: "openai", Think: &enabled}, "only supported for the ollama"},
		{"small thinking budget", &domain.Agent{Provider: "anthropic", Model: "claude-sonnet-4-5", Reasoning: &enabled, MaxOutputTokens: &small}, "greater than 1024"},
		{"gpt-oss disabling", &domain.Agent{Provider: "ollama", Model: "gpt-oss:20b", Reasoning: &disabled}, "does not support disabling"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetProviderConfig(context.Background(), tc.agent)
			if err == nil || !strings.Contains(err.Error(), tc.error) {
				t.Fatalf("error = %v, want %q", err, tc.error)
			}
		})
	}
	for _, think := range []*bool{&disabled, &enabled} {
		config, err := GetProviderConfig(context.Background(), &domain.Agent{Provider: "ollama", Think: think})
		if err != nil {
			t.Fatal(err)
		}
		if config.(map[string]any)["think"] != *think {
			t.Fatalf("legacy think = %v, error = %v", config, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GetProviderConfig(ctx, &domain.Agent{Provider: "ollama"}); err != context.Canceled {
		t.Fatalf("canceled context error = %v", err)
	}
}
