package filesystem

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type EditFileInput struct {
	Path         string `json:"path"`
	SearchStr    string `json:"search_str"`
	ReplaceStr   string `json:"replace_str"`
	ReplaceCount int    `json:"replace_count"`
}
type EditFileOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func DefineEditFileTool(g *genkit.Genkit, tool *domain.ToolConfig, root *files.Root) *ai.ToolAction[EditFileInput, EditFileOutput] {
	return genkit.DefineTool(
		g,
		"edit_file",
		"Edits an existing file relative to the configured work directory by replacing search_str with replace_str. Set replace_count=1 for single or -1 for all occurrences. Absolute paths, '..' and links outside that directory are forbidden.",
		func(ctx *ai.ToolContext, input EditFileInput) (EditFileOutput, error) {
			err := root.Replace(input.Path, input.SearchStr, input.ReplaceStr, input.ReplaceCount)
			if err != nil {
				return EditFileOutput{Success: false, Message: err.Error()}, nil
			}

			return EditFileOutput{Success: true, Message: "File successfully updated"}, nil
		},
	)
}
