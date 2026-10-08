package filesystem

import (
	"os"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ReadFileInput struct {
	Path string `json:"path"`
}
type ReadFileOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Content string `json:"content"`
}

func DefineReadFileTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[ReadFileInput, ReadFileOutput] {
	return genkit.DefineTool(
		g,
		"read_file",
		"Read file",
		func(ctx *ai.ToolContext, input ReadFileInput) (ReadFileOutput, error) {
			bytes, err := os.ReadFile(input.Path)

			if err != nil {
				return ReadFileOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			return ReadFileOutput{
				Success: false,
				Content: string(bytes),
			}, nil
		},
	)
}
