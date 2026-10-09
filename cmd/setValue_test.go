package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
)

func TestSetValueCommand(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"valid update", []string{"example", "agent.temperature", "0.6"}, false},
		{"invalid value", []string{"example", "agent.temperature", "3"}, true},
		{"missing value", []string{"example", "agent.temperature"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			path := filepath.Join(workspace, "agents", "example", config.AgentFile)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			original := []byte("version: '1.0'\nagent:\n  name: example\n  provider: openai\n  model: gpt-4o\n  temperature: 0.2\n  work_dir: .\n  system_prompt: Be helpful\n  tools: []\n")
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workspace, config.ProjectFile), []byte("agents_dir: agents\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var output, errorOutput bytes.Buffer
			root := &cobra.Command{Use: "atc", SilenceErrors: rootCmd.SilenceErrors, SilenceUsage: rootCmd.SilenceUsage}
			root.PersistentFlags().String("workspace-dir", workspace, "workspace")
			root.AddCommand(&cobra.Command{Use: setValueCmd.Use, Args: setValueCmd.Args, RunE: setValueCmd.RunE})
			root.SetOut(&output)
			root.SetErr(&errorOutput)
			root.SetArgs(append([]string{"set-value"}, tc.args...))
			err := execute(context.Background(), root)
			if (err != nil) != tc.wantErr {
				t.Fatalf("command error = %v; wantErr=%v", err, tc.wantErr)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantErr {
				if !bytes.Equal(content, original) || strings.Count(ansi.Strip(errorOutput.String()), "✗ ") != 1 {
					t.Fatalf("invalid command modified config or did not report one error: %q", errorOutput.String())
				}
			} else {
				configuration, err := config.NewConfigService(nil, workspace).LoadAgent(context.Background(), "example")
				if err != nil || configuration.Agent.Temperature != 0.6 || errorOutput.Len() != 0 || !strings.Contains(output.String(), "Updated example: agent.temperature") {
					t.Fatalf("update failed: %+v, %v, stdout=%q, stderr=%q", configuration, err, output.String(), errorOutput.String())
				}
			}
		})
	}
}
