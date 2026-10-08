package tools

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/tools/builtin"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ToolRegistration struct {
	g           *genkit.Genkit
	Tools       []ai.ToolRef
	configTools []domain.ToolConfig
}

func NewToolRegistration(g *genkit.Genkit, ConfigTools []domain.ToolConfig) *ToolRegistration {
	tools := make([]ai.ToolRef, 0, len(ConfigTools))

	return &ToolRegistration{
		g:           g,
		Tools:       tools,
		configTools: ConfigTools,
	}
}

func (r *ToolRegistration) Register() {
	for _, tool := range r.configTools {
		r.Tools = append(r.Tools, r.defineTool(&tool))
	}
}

func (r *ToolRegistration) defineTool(tool *domain.ToolConfig) ai.ToolRef {
	switch tool.Type {
	case domain.ToolTypeBuiltin:
		return builtin.DefineBuiltinTool(r.g, tool)
	default:
		return nil
	}
}
