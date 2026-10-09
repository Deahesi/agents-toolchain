package runtime

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/tools"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestStreamFinalAnswerAndThinkingConfig(t *testing.T) {
	disabled, enabled := false, true
	for _, tc := range []struct {
		name    string
		think   *bool
		answer  string
		wantErr bool
	}{
		{name: "reasoning without answer", wantErr: true},
		{name: "whitespace without answer", answer: " \n", wantErr: true},
		{name: "default thinking", answer: "No staged changes."},
		{name: "thinking disabled", think: &disabled, answer: "No staged changes."},
		{name: "thinking enabled", think: &enabled, answer: "No staged changes."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			g := genkit.Init(ctx)
			genkit.DefineModel(g, "ollama/test", &ai.ModelOptions{Supports: &ai.ModelSupports{Multiturn: true, SystemRole: true}}, func(ctx context.Context, request *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
				config, ok := request.Config.(map[string]any)
				if !ok {
					t.Fatalf("unexpected config type %T", request.Config)
				}
				value, present := config["think"]
				if tc.think == nil {
					if present {
						t.Errorf("think must be omitted by default, got %v", value)
					}
				} else if !present || value != *tc.think {
					t.Errorf("think = %v (present %v), want %v", value, present, *tc.think)
				}
				parts := []*ai.Part{ai.NewReasoningPart("Inspect repository first.", nil)}
				if tc.answer != "" {
					parts = append(parts, ai.NewTextPart(tc.answer))
				}
				if cb != nil {
					if err := cb(ctx, &ai.ModelResponseChunk{Content: parts}); err != nil {
						return nil, err
					}
				}
				return &ai.ModelResponse{FinishReason: "stop", Message: &ai.Message{Role: ai.RoleModel, Content: parts}}, nil
			})
			var output bytes.Buffer
			service := NewRuntimeService(ui.NewUIService(ui.WithOutput(&output), ui.WithErrorOutput(&output)), nil)
			service.runtime = g
			service.toolRegistration = &tools.ToolRegistration{}
			err := service.stream(ctx, domain.Agent{Provider: "ollama", Model: "test", Think: tc.think}, "Inspect repository")
			if (err != nil) != tc.wantErr {
				t.Fatalf("stream error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "no final text") {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantErr && !strings.Contains(output.String(), tc.answer) {
				t.Fatalf("final answer missing from output: %q", output.String())
			}
		})
	}
}
