package atctools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/pkg/atctools"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

func TestRegister(t *testing.T) {
	names := []string{
		atctools.ReadFile, atctools.WriteFile, atctools.EditFile,
		atctools.ListFiles, atctools.SearchFiles, atctools.ExecuteCommand,
		atctools.GetEnvironment, atctools.GetDateTime,
	}
	g := genkit.Init(context.Background())
	r, err := atctools.Register(g, atctools.Config{WorkDir: t.TempDir(), Tools: names})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	if len(r.Tools) != len(names) {
		t.Fatalf("got %d tools, want %d", len(r.Tools), len(names))
	}
	for i, ref := range r.Tools {
		tool, ok := ref.(ai.Tool)
		if !ok {
			t.Fatalf("tool does not implement ai.Tool: %T", ref)
		}
		if tool.Name() != names[i] {
			t.Errorf("got tool %q, want %q", tool.Name(), names[i])
		}
	}
}

func TestFilesystemRootAndClose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	g := genkit.Init(context.Background())
	r, err := atctools.Register(g, atctools.Config{WorkDir: dir, Tools: []string{atctools.ReadFile}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	tool := r.Tools[0].(ai.Tool)
	ctx := context.Background()
	read := func(path string) bool {
		t.Helper()
		result, err := tool.RunRawMultipart(ctx, map[string]any{"path": path})
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(result.Output)
		if err != nil {
			t.Fatal(err)
		}
		var output struct {
			Success bool   `json:"success"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(data, &output); err != nil {
			t.Fatal(err)
		}
		if output.Success && output.Content != "hello" {
			t.Fatalf("unexpected file content: %q", output.Content)
		}
		return output.Success
	}
	if !read("hello.txt") {
		t.Fatal("expected read in configured root to succeed")
	}
	if read("../outside.txt") {
		t.Fatal("expected path escape to fail")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if read("hello.txt") {
		t.Fatal("expected read after close to fail")
	}
}

func TestRegisterValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config atctools.Config
		want   string
	}{
		{"unknown", atctools.Config{Tools: []string{"missing"}}, "unknown built-in"},
		{"duplicate", atctools.Config{Tools: []string{atctools.ReadFile, atctools.ReadFile}}, "duplicate"},
		{"missing directory", atctools.Config{Tools: []string{atctools.ReadFile}}, "open work directory"},
		{"nonexistent directory", atctools.Config{WorkDir: filepath.Join(t.TempDir(), "missing"), Tools: []string{atctools.WriteFile}}, "open work directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := atctools.Register(genkit.Init(context.Background()), tc.config)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want error containing %q", err, tc.want)
			}
		})
	}
	if _, err := atctools.Register(nil, atctools.Config{}); err == nil {
		t.Fatal("expected nil Genkit instance to fail")
	}
}

func TestRegisterWithoutFilesystem(t *testing.T) {
	for _, names := range [][]string{nil, {atctools.GetDateTime, atctools.GetEnvironment}} {
		r, err := atctools.Register(genkit.Init(context.Background()), atctools.Config{Tools: names})
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
