package filesystem

import (
	"os"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type WriteFileInput struct {
	Path string `json:"path"`
	Text string `json:"text"`
}
type WriteFileOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func DefineWriteFileTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[WriteFileInput, WriteFileOutput] {
	return genkit.DefineTool(
		g,
		"write_file",
		"Write file",
		func(ctx *ai.ToolContext, input WriteFileInput) (WriteFileOutput, error) {
			file, err := os.Create(input.Path)

			if err != nil {
				return WriteFileOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			_, err = file.Write([]byte(input.Text))
			if err != nil {
				return WriteFileOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			return WriteFileOutput{
				Success: true,
			}, nil
		},
	)
}
