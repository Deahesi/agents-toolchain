package filesystem

import (
	"os"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
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

func DefineEditFileTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[EditFileInput, EditFileOutput] {
	return genkit.DefineTool(
		g,
		"edit_file",
		"Edits existing file by replacing search_str with replace_str. Set replace_count=1 for single or -1 for all occurrences. Do not use for creating files.",
		func(ctx *ai.ToolContext, input EditFileInput) (EditFileOutput, error) {
			bytes, err := os.ReadFile(input.Path)
			if err != nil {
				return EditFileOutput{Success: false, Message: err.Error()}, nil
			}

			result := strings.Replace(string(bytes), input.SearchStr, input.ReplaceStr, input.ReplaceCount)

			err = os.WriteFile(input.Path, []byte(result), 0666)
			if err != nil {
				return EditFileOutput{Success: false, Message: err.Error()}, nil
			}

			return EditFileOutput{Success: true, Message: "File successfully updated"}, nil
		},
	)
}
