package tools_test

import (
	"context"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/Deahesi/agents-toolchain/internal/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestFilesystemToolsUseConfiguredRoot(t *testing.T) {
	root, err := files.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	names := []string{
		"read_file",
		"write_file",
		"edit_file",
		"list_files",
		"search_files",
	}

	configs := make([]domain.ToolConfig, len(names))
	for i, name := range names {
		configs[i] = domain.ToolConfig{
			Type: domain.ToolTypeBuiltin,
			Name: name,
		}
	}

	g := genkit.Init(context.Background())
	registration := tools.NewToolRegistration(g, configs, root)
	registration.Register()

	if len(registration.Tools) != len(names) {
		t.Fatalf("registered %d tools, want %d",
			len(registration.Tools), len(names))
	}

	registered := make(map[string]bool)
	for _, ref := range registration.Tools {
		tool, ok := ref.(ai.Tool)
		if !ok {
			t.Fatalf("registered tool does not implement ai.Tool: %T", ref)
		}
		registered[tool.Name()] = true
	}

	for _, name := range names {
		if !registered[name] {
			t.Errorf("tool %q was not registered", name)
		}
	}
}
