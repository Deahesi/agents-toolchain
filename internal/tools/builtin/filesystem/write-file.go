package filesystem

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
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

func DefineWriteFileTool(g *genkit.Genkit, tool *domain.ToolConfig, root *files.Root) *ai.ToolAction[WriteFileInput, WriteFileOutput] {
	return genkit.DefineTool(
		g,
		"write_file",
		"Creates or overwrites a file relative to the configured work directory. Absolute paths, '..' and links outside that directory are forbidden.",
		func(ctx *ai.ToolContext, input WriteFileInput) (WriteFileOutput, error) {
			err := root.WriteFile(input.Path, []byte(input.Text), 0o666)
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
