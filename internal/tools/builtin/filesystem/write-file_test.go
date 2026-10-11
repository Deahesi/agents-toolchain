package filesystem

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/Deahesi/agents-toolchain/internal/helpers"
	"github.com/firebase/genkit/go/genkit"
)

const (
	TestWriteFileName = "test.txt"
	TestWriteChildDir = "test"
	TestWriteForbDir  = "forbidden"
)

func TestWriteFile(t *testing.T) {
	ctx := context.Background()
	g := genkit.Init(ctx)

	toolConfig := domain.ToolConfig{
		Type: domain.ToolTypeBuiltin,
		Name: "write_file",
	}

	root, err := files.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	err = root.Mkdir(TestWriteChildDir, 0o777)
	if err != nil {
		t.Fatal(err)
	}
	err = root.Mkdir(TestWriteForbDir, 0o666)
	if err != nil {
		t.Fatal(err)
	}

	tool := DefineWriteFileTool(g, &toolConfig, root)

	for _, tc := range []struct {
		name     string
		input    map[string]any
		response WriteFileOutput
	}{
		{
			name: "test create base file",
			input: map[string]any{
				"path": TestWriteFileName,
				"text": "TEST FILE TEXT CONTENT",
			},
			response: WriteFileOutput{
				Success: true,
			},
		},
		{
			name: "test create tree file",
			input: map[string]any{
				"path": filepath.Join(TestWriteChildDir, TestWriteFileName),
				"text": "TEST FILE TEXT CONTENT",
			},
			response: WriteFileOutput{
				Success: true,
			},
		},
		{
			name: "test fail when creating file in no root dir",
			input: map[string]any{
				"path": filepath.Join("..", TestWriteFileName),
				"text": "TEST FILE TEXT CONTENT",
			},
			response: WriteFileOutput{
				Success: false,
				Message: fmt.Sprintf("write ../%s: relative path without '..' required: permission denied", TestWriteFileName),
			},
		},
		{
			name: "test fail when creating file with no permissions",
			input: map[string]any{
				"path": filepath.Join(TestWriteForbDir, TestWriteFileName),
				"text": "TEST FILE TEXT CONTENT",
			},
			response: WriteFileOutput{
				Success: false,
				Message: fmt.Sprintf("openat %s/%s: permission denied", TestWriteForbDir, TestWriteFileName),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tool.Name() != "write_file" {
				t.Errorf("tool name err: want %s; got %s", "write_file", tool.Name())
			}

			response, err := tool.RunRawMultipart(ctx, tc.input)
			if err != nil {
				t.Fatal(err)
			}

			output, err := helpers.AnyToStruct[WriteFileOutput](response.Output)
			if err != nil {
				t.Fatal(err)
			}

			if output.Message != tc.response.Message {
				t.Errorf("output message err: want %s; got %s", tc.response.Message, output.Message)
			}

			if output.Success != tc.response.Success {
				t.Errorf("output Success err: want %t; got %t", tc.response.Success, output.Success)
			}

			if output.Success {
				b, err := root.ReadFile(tc.input["path"].(string))
				if err != nil {
					t.Fatal(err)
				}

				if string(b) != tc.input["text"].(string) {
					t.Errorf("file creation fail: text content: want %s; got %s", tc.input["text"].(string), string(b))
				}
			}
		})
	}
}
