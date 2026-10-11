package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/helpers"
	"github.com/firebase/genkit/go/genkit"
)

func TestSearchFiles(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	toolConfig := domain.ToolConfig{
		Type: domain.ToolTypeBuiltin,
		Name: "search_files",
	}

	fs := createTestFilesystem(t)
	defer fs.root.Close()

	tool := DefineSearchFilesTool(g, &toolConfig, fs.root)

	for _, tc := range []struct {
		name     string
		input    map[string]any
		response SearchFilesOutput
	}{
		{
			name: "Search test search file in root dir",
			input: map[string]any{
				"dir":     ".",
				"pattern": TestSearchFileName,
			},
			response: SearchFilesOutput{
				Success: true,
				Matches: []string{TestSearchFileName},
			},
		},
		{
			name: "Search test search file in child dir",
			input: map[string]any{
				"dir":     TestChildDir,
				"pattern": TestSearchFileName,
			},
			response: SearchFilesOutput{
				Success: true,
				Matches: []string{fs.TestSearchFileInRootChildPath},
			},
		},
		{
			name: "Search test files in root dir by pattern",
			input: map[string]any{
				"dir":     ".",
				"pattern": "*.txt",
			},
			response: SearchFilesOutput{
				Success: true,
				Matches: []string{TestSearchFileName, TestFileName},
			},
		},
		{
			name: "Search test files in root child dir by pattern",
			input: map[string]any{
				"dir":     TestChildDir,
				"pattern": "*.txt",
			},
			response: SearchFilesOutput{
				Success: true,
				Matches: []string{fs.TestSearchFileInRootChildPath, fs.TestFileInRootChildPath},
			},
		},
		{
			name: "Search test test file in external dir",
			input: map[string]any{
				"dir":     filepath.Join("..", TestExternalDir),
				"pattern": TestFileName,
			},
			response: SearchFilesOutput{
				Success: false,
				Message: fmt.Sprintf("readdir %s: relative path without '..' required: permission denied", filepath.Join("..", TestExternalDir)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tool.Name() != "search_files" {
				t.Errorf("tool name err: want %s; got %s", "search_files", tool.Name())
			}

			response, err := tool.RunRawMultipart(ctx, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			output, err := helpers.AnyToStruct[SearchFilesOutput](response.Output)
			if err != nil {
				t.Fatal(err)
			}

			if output.Message != tc.response.Message {
				t.Errorf("output Message err: want %s; got %s", tc.response.Message, output.Message)
			}

			if output.Success != tc.response.Success {
				t.Errorf("output Success err: want %t; got %t", tc.response.Success, output.Success)
			}

			if !reflect.DeepEqual(output.Matches, tc.response.Matches) {
				t.Errorf("output Mathes err: want %v; got %s", tc.response.Matches, output.Matches)
			}
		})
	}
}
