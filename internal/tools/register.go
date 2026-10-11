package tools

import (
	"fmt"

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

func NewToolRegistration(g *genkit.Genkit, configTools []domain.ToolConfig, root *files.Root) *ToolRegistration {
	tools := make([]ai.ToolRef, 0, len(configTools))

	return &ToolRegistration{
		g:           g,
		Tools:       tools,
		configTools: configTools,
		root:        root,
	}
}

func (r *ToolRegistration) Register() error {
	for _, tool := range r.configTools {
		tool, err := r.defineTool(&tool)
		if err != nil {
			return err
		}

		r.Tools = append(r.Tools, tool)
	}

	return nil
}

func (r *ToolRegistration) defineTool(tool *domain.ToolConfig) (ai.ToolRef, error) {
	switch tool.Type {
	case domain.ToolTypeBuiltin:
		return builtin.DefineBuiltinTool(r.g, tool, r.root)
	default:
		return nil, fmt.Errorf("unexpected tool type: %s", tool.Type)
	}
}
