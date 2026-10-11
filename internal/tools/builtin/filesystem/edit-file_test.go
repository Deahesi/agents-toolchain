package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/helpers"
	"github.com/firebase/genkit/go/genkit"
)

func TestEditFile(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	toolConfig := domain.ToolConfig{
		Type: domain.ToolTypeBuiltin,
		Name: "edit_file",
	}

	fs := createTestFilesystem(t)
	defer fs.root.Close()

	tool := DefineEditFileTool(g, &toolConfig, fs.root)

	for _, tc := range []struct {
		name     string
		input    map[string]any
		response EditFileOutput
	}{
		{
			name: "edit root dir test file once",
			input: map[string]any{
				"path":          fs.TestFileInRootPath,
				"search_str":    "TEST",
				"replace_str":   "TEXT",
				"replace_count": 1,
			},
			response: EditFileOutput{
				Success: true,
			},
		},
		{
			name: "edit root dir test file multiple",
			input: map[string]any{
				"path":          fs.TestFileInRootPath,
				"search_str":    "TEST",
				"replace_str":   "TEXT",
				"replace_count": 2,
			},
			response: EditFileOutput{
				Success: true,
			},
		},
		{
			name: "edit child dir test file multiple",
			input: map[string]any{
				"path":          fs.TestFileInRootChildPath,
				"search_str":    "TEST",
				"replace_str":   "TEXT",
				"replace_count": 2,
			},
			response: EditFileOutput{
				Success: true,
			},
		},
		{
			name: "edit external dir test file multiple",
			input: map[string]any{
				"path":          filepath.Join("..", TestExternalDir, TestFileName),
				"search_str":    "TEST",
				"replace_str":   "TEXT",
				"replace_count": 1,
			},
			response: EditFileOutput{
				Success: false,
				Message: fmt.Sprintf("replace %s: relative path without '..' required: permission denied", filepath.Join("..", TestExternalDir, TestFileName)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tool.Name() != "edit_file" {
				t.Errorf("tool name err: want %s; got %s", "edit_file", tool.Name())
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

			if output.Success {
				content, err := fs.root.ReadFile(tc.input["path"].(string))
				if err != nil {
					t.Fatal(err)
				}

				strResult := string(content)

				replacedCorrect := strings.Replace(
					TestFileContent,
					tc.input["search_str"].(string),
					tc.input["replace_str"].(string),
					tc.input["replace_count"].(int),
				)

				if strResult != replacedCorrect {
					t.Errorf("file result err: want %s; got %s", replacedCorrect, strResult)
				}
			}
		})
	}
}
