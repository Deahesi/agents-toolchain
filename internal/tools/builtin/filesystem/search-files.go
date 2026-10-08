package filesystem

import (
	"path/filepath"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type SearchFilesInput struct {
	Dir     string `json:"dir"`     // Папка для поиска (например, "." или "internal/domain")
	Pattern string `json:"pattern"` // Маска поиска (например, "*.go", "main.*", "*user*")
}

type SearchFilesOutput struct {
	Success bool     `json:"success"`
	Message string   `json:"message,omitempty"`
	Matches []string `json:"matches,omitempty"`
}

func DefineSearchFilesTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[SearchFilesInput, SearchFilesOutput] {
	return genkit.DefineTool(
		g,
		"search_files",
		"Finds files in a specific directory using a glob pattern (e.g., '*.go' or '*config*'). Non-recursive.",
		func(ctx *ai.ToolContext, input SearchFilesInput) (SearchFilesOutput, error) {
			dirPath := input.Dir
			if dirPath == "" {
				dirPath = "."
			}

			fullPattern := filepath.Join(dirPath, input.Pattern)

			matches, err := filepath.Glob(fullPattern)
			if err != nil {
				return SearchFilesOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			var results []string
			for _, match := range matches {
				results = append(results, filepath.Clean(match))
			}

			return SearchFilesOutput{
				Success: true,
				Matches: results,
			}, nil
		},
	)
}
