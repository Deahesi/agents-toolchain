package filesystem

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ReadFileInput struct {
	Path string `json:"path"`
}
type ReadFileOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
}

func DefineReadFileTool(g *genkit.Genkit, tool *domain.ToolConfig, root *files.Root) *ai.ToolAction[ReadFileInput, ReadFileOutput] {
	return genkit.DefineTool(
		g,
		"read_file",
		"Reads a file relative to the configured work directory. Absolute paths, '..' and links outside that directory are forbidden.",
		func(ctx *ai.ToolContext, input ReadFileInput) (ReadFileOutput, error) {
			bytes, err := root.ReadFile(input.Path)

			if err != nil {
				return ReadFileOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			return ReadFileOutput{
				Success: true,
				Content: string(bytes),
			}, nil
		},
	)
}
