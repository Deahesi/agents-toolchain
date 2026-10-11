package filesystem

import (
	"context"
	"fmt"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/helpers"
	"github.com/firebase/genkit/go/genkit"
)

func TestReadFile(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	toolConfig := domain.ToolConfig{
		Type: domain.ToolTypeBuiltin,
		Name: "read_file",
	}

	fs := createTestFilesystem(t)
	defer fs.root.Close()

	tool := DefineReadFileTool(g, &toolConfig, fs.root)

	for _, tc := range []struct {
		name     string
		input    map[string]any
		response ReadFileOutput
	}{
		{
			name: "test read test file in root dir",
			input: map[string]any{
				"path": fs.TestFileInRootChildPath,
			},
			response: ReadFileOutput{
				Success: true,
				Content: TestFileContent,
			},
		},
		{
			name: "test read no exists file",
			input: map[string]any{
				"path": TestFileNameNoExist,
			},
			response: ReadFileOutput{
				Success: false,
			},
		},
		{
			name: "test read test file in external dir",
			input: map[string]any{
				"path": fs.TestFileInExternalPath,
			},
			response: ReadFileOutput{
				Success: false,
				Message: fmt.Sprintf("read %s: relative path without '..' required: permission denied", fs.TestFileInExternalPath),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tool.Name() != "read_file" {
				t.Errorf("tool name err: want %s; got %s", "read_file", tool.Name())
			}

			response, err := tool.RunRawMultipart(ctx, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			output, err := helpers.AnyToStruct[ReadFileOutput](response.Output)
			if err != nil {
				t.Fatal(err)
			}

			if !tc.response.Success && output.Message == "" {
				t.Error("expected an error message")
			}

			if (tc.response.Success || tc.response.Message != "") && output.Message != tc.response.Message {
				t.Errorf("output Message err: want %s; got %s", tc.response.Message, output.Message)
			}

			if output.Success != tc.response.Success {
				t.Errorf("output Success err: want %t; got %t", tc.response.Success, output.Success)
			}

			if output.Content != tc.response.Content {
				t.Errorf("output Content err: want %s; got %s", tc.response.Content, output.Content)
			}
		})
	}
}
