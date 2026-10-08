package builtin

import (
	"bytes"
	"fmt"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/tools/builtin/filesystem"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/goccy/go-yaml"
)

func DefineBuiltinTool(g *genkit.Genkit, tool *domain.ToolConfig) ai.ToolRef {
	switch tool.Name {
	case "read_file":
		return filesystem.DefineReadFileTool(g, tool)
	case "write_file":
		return filesystem.DefineWriteFileTool(g, tool)
	case "edit_file":
		return filesystem.DefineEditFileTool(g, tool)
	default:
		return nil
	}
}

func Decode[T any](raw map[string]any) (T, error) {
	var cfg T

	data, err := yaml.Marshal(raw)
	if err != nil {
		return cfg, fmt.Errorf("marshal config: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))

	if err := decoder.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}
