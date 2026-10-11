package builtin

import (
	"context"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/firebase/genkit/go/genkit"
)

func TestDefineBuiltinTool(t *testing.T) {

	for _, tc := range []struct {
		name     string
		toolName string
		wantErr  bool
	}{
		{name: "test read_file define", toolName: "read_file"},
		{name: "test write_file define", toolName: "write_file"},
		{name: "test edit_file define", toolName: "edit_file"},
		{name: "test list_files define", toolName: "list_files"},
		{name: "test search_files define", toolName: "search_files"},

		{name: "test execute_command define", toolName: "execute_command"},
		{name: "test get_datetime define", toolName: "get_datetime"},
		{name: "test get_environment define", toolName: "get_environment"},

		{name: "test undefined command", toolName: "undefined", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := files.OpenRoot(t.TempDir())

			if err != nil {
				t.Fatal(err)
			}

			ctx := context.Background()
			g := genkit.Init(ctx)
			config := domain.ToolConfig{
				Name: tc.toolName,
			}

			tool, err := DefineBuiltinTool(g, &config, root)

			if err != nil {
				if tc.wantErr {
					return
				}

				t.Fatal(err)
			}

			if tool.Name() != tc.toolName {
				t.Errorf("define tool %s fail. got: %s", tc.toolName, tool.Name())
			}
		})
	}
}
