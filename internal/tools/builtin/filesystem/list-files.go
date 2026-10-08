package filesystem

import (
	"os"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ListFilesInput struct {
	Path string `json:"path"`
}

type FileItem struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
}

type ListFilesOutput struct {
	Success bool       `json:"success"`
	Message string     `json:"message,omitempty"`
	Files   []FileItem `json:"files,omitempty"`
}

func DefineListFilesTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[ListFilesInput, ListFilesOutput] {
	return genkit.DefineTool(
		g,
		"list_files",
		"Lists files and folders inside the specified directory (non-recursive).",
		func(ctx *ai.ToolContext, input ListFilesInput) (ListFilesOutput, error) {
			dirPath := input.Path
			if dirPath == "" {
				dirPath = "."
			}

			entries, err := os.ReadDir(dirPath)
			if err != nil {
				return ListFilesOutput{
					Success: false,
					Message: err.Error(),
				}, nil
			}

			var files []FileItem
			for _, entry := range entries {
				files = append(files, FileItem{
					Name:  entry.Name(),
					IsDir: entry.IsDir(),
				})
			}

			return ListFilesOutput{
				Success: true,
				Files:   files,
			}, nil
		},
	)
}
