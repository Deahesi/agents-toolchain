package filesystem

import (
	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type SearchFilesInput struct {
	Dir     string `json:"dir"`
	Pattern string `json:"pattern"`
}

type SearchFilesOutput struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Matches []string `json:"matches,omitempty"`
}

func DefineSearchFilesTool(g *genkit.Genkit, tool *domain.ToolConfig, root *files.Root) *ai.ToolAction[SearchFilesInput, SearchFilesOutput] {
	return genkit.DefineTool(
		g,
		"search_files",
		"Finds entries non-recursively, relative to the configured work directory. Empty dir means the work directory. Pattern matches names only (e.g., '*.go'), without directory separators. Absolute paths, '..' and links outside the work directory are forbidden.",
		func(ctx *ai.ToolContext, input SearchFilesInput) (SearchFilesOutput, error) {
			dirPath := input.Dir
			if dirPath == "" {
				dirPath = "."
			}

			matches, err := root.Search(dirPath, input.Pattern)
			if err != nil {
				return SearchFilesOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			return SearchFilesOutput{
				Success: true,
				Matches: matches,
			}, nil
		},
	)
}
