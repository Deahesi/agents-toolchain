package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/Deahesi/agents-toolchain/internal/files"
	"github.com/Deahesi/agents-toolchain/internal/tools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestFilesystemToolsUseConfiguredRoot(t *testing.T) {
	parent := t.TempDir()
	workDir := filepath.Join(parent, "work")
	if err := os.Mkdir(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(outside, []byte("outside secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := files.OpenRoot(workDir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	ctx := context.Background()
	g := genkit.Init(ctx)
	var configs []domain.ToolConfig
	for _, name := range []string{"read_file", "write_file", "edit_file", "list_files", "search_files"} {
		configs = append(configs, domain.ToolConfig{Type: domain.ToolTypeBuiltin, Name: name})
	}
	registration := tools.NewToolRegistration(g, configs, root)
	registration.Register()
	registered := make(map[string]ai.Tool)
	for _, ref := range registration.Tools {
		tool, ok := ref.(ai.Tool)
		if !ok {
			t.Fatalf("registered tool does not implement ai.Tool: %T", ref)
		}
		registered[tool.Name()] = tool
	}
	run := func(name string, input map[string]any, success bool) map[string]any {
		t.Helper()
		output, err := registered[name].RunRaw(ctx, input)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		// Normalize typed outputs and decoded JSON outputs alike.
		data, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result["success"] != success {
			t.Fatalf("%s(%v) = %s; want success=%v", name, input, data, success)
		}
		return result
	}
	run("write_file", map[string]any{"path": "note.txt", "text": "inside inside"}, true)
	run("edit_file", map[string]any{"path": "note.txt", "search_str": "inside", "replace_str": "edited", "replace_count": 1}, true)
	result := run("read_file", map[string]any{"path": "note.txt"}, true)
	if result["content"] != "edited inside" {
		t.Fatalf("read from wrong directory: %v", result)
	}
	result = run("list_files", map[string]any{"path": ""}, true)
	entries, ok := result["files"].([]any)
	if !ok || len(entries) != 1 || entries[0].(map[string]any)["name"] != "note.txt" {
		t.Fatalf("list from wrong directory: %v", result)
	}
	result = run("search_files", map[string]any{"dir": "", "pattern": "*.txt"}, true)
	matches, ok := result["matches"].([]any)
	if !ok || len(matches) != 1 || matches[0] != "note.txt" {
		t.Fatalf("search returned unexpected paths: %v", result)
	}
	for _, path := range []string{"../secret.txt", outside, filepath.Join(workDir, "note.txt")} {
		run("read_file", map[string]any{"path": path}, false)
		run("write_file", map[string]any{"path": path, "text": "modified"}, false)
		run("edit_file", map[string]any{"path": path, "search_str": "secret", "replace_str": "modified", "replace_count": -1}, false)
	}
	for _, path := range []string{"..", parent, workDir} {
		run("list_files", map[string]any{"path": path}, false)
		run("search_files", map[string]any{"dir": path, "pattern": "*"}, false)
	}
	run("search_files", map[string]any{"dir": "", "pattern": "../*"}, false)
	run("write_file", map[string]any{"path": "missing/file.txt", "text": "test"}, false)
	run("edit_file", map[string]any{"path": "missing.txt", "search_str": "x", "replace_str": "y", "replace_count": -1}, false)
	content, err := os.ReadFile(outside)
	if err != nil || string(content) != "outside secret" {
		t.Fatalf("outside file changed: %q, %v", content, err)
	}
}
