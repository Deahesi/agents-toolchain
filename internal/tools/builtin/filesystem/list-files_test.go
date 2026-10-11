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

func TestListFiles(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	toolConfig := domain.ToolConfig{
		Type: domain.ToolTypeBuiltin,
		Name: "list_files",
	}

	fs := createTestFilesystem(t)
	defer fs.root.Close()

	tool := DefineListFilesTool(g, &toolConfig, fs.root)

	for _, tc := range []struct {
		name     string
		input    map[string]any
		response ListFilesOutput
	}{
		{
			name: "list root dir files",
			input: map[string]any{
				"path": ".",
			},
			response: ListFilesOutput{
				Success: true,
				Files: []FileItem{
					{Name: TestChildDir, IsDir: true},
					{Name: TestSearchFileName, IsDir: false},
					{Name: TestFileName, IsDir: false},
				},
			},
		},
		{
			name: "list root child dir files",
			input: map[string]any{
				"path": TestChildDir,
			},
			response: ListFilesOutput{
				Success: true,
				Files: []FileItem{
					{Name: TestSearchFileName, IsDir: false},
					{Name: TestFileName, IsDir: false},
				},
			},
		},
		{
			name: "list external dir files",
			input: map[string]any{
				"path": filepath.Join("..", TestExternalDir),
			},
			response: ListFilesOutput{
				Success: false,
				Message: fmt.Sprintf("readdir %s: relative path without '..' required: permission denied", filepath.Join("..", TestExternalDir)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tool.Name() != "list_files" {
				t.Errorf("tool name err: want %s; got %s", "list_files", tool.Name())
			}

			response, err := tool.RunRawMultipart(ctx, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			output, err := helpers.AnyToStruct[ListFilesOutput](response.Output)
			if err != nil {
				t.Fatal(err)
			}

			if output.Message != tc.response.Message {
				t.Errorf("output Message err: want %s; got %s", tc.response.Message, output.Message)
			}

			if output.Success != tc.response.Success {
				t.Errorf("output Success err: want %t; got %t", tc.response.Success, output.Success)
			}

			if !reflect.DeepEqual(output.Files, tc.response.Files) {
				t.Errorf("output Mathes err: want %v; got %v", tc.response.Files, output.Files)
			}
		})
	}
}
