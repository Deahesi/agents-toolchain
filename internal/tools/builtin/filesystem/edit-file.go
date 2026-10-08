package filesystem

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type EditFileInput struct {
	Path       string `json:"path"`
	NewContent string `json:"text"`
}
type EditFileOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func DefineEditFileTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[EditFileInput, EditFileOutput] {
	return genkit.DefineTool(
		g,
		"edit_file",
		"Edit file",
		func(ctx *ai.ToolContext, input EditFileInput) (EditFileOutput, error) {
			// file, err := os.ReadFile(input.Path)

			return EditFileOutput{
				Success: false,
				Message: "Tool not available",
			}, nil
		},
	)
}
