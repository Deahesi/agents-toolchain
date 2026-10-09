package tools

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/Deahesi/agents-toolchain/internal/tools/builtin"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ToolRegistration struct {
	g           *genkit.Genkit
	Tools       []ai.ToolRef
	configTools []domain.ToolConfig
	root        *files.Root
}

// NewToolRegistration binds filesystem tools to root for their entire lifetime.
// The caller owns root and must keep it open until all tool calls finish.
func NewToolRegistration(g *genkit.Genkit, configTools []domain.ToolConfig, root *files.Root) *ToolRegistration {
	tools := make([]ai.ToolRef, 0, len(configTools))

	return &ToolRegistration{
		g:           g,
		Tools:       tools,
		configTools: configTools,
		root:        root,
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
		return builtin.DefineBuiltinTool(r.g, tool, r.root)
	default:
		return nil
	}
}
