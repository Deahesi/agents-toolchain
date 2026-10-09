package config

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"gopkg.in/yaml.v3"
)

const editableAgent = `# Keep this document comment.
version: "1.0"
agent:
  name: example
  description: Original description
  provider: openai
  model: gpt-4o # Keep this model comment.
  temperature: 0.2
  work_dir: .
  system_prompt: |
    Be helpful.
    Keep context.
  memory:
    type: local_file
    path: .history.json
  tools: []
`

func editableConfig(t *testing.T, content string) (*ConfigService, string) {
	t.Helper()
	workspace := t.TempDir()
	path := filepath.Join(workspace, "agents", "example", AgentFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ProjectFile), []byte("agents_dir: agents\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewConfigService(nil, workspace), path
}

func TestSetAgentValue(t *testing.T) {
	for _, tc := range []struct {
		field string
		value string
		check func(*domain.AgentConfig) bool
	}{
		{"agent.model", "123", func(c *domain.AgentConfig) bool { return c.Agent.Model == "123" }},
		{"agent.description", "true", func(c *domain.AgentConfig) bool { return c.Agent.Description == "true" }},
		{"agent.description", "null", func(c *domain.AgentConfig) bool { return c.Agent.Description == "null" }},
		{"agent.description", "", func(c *domain.AgentConfig) bool { return c.Agent.Description == "" }},
		{"agent.system_prompt", "First line\nSecond line\n", func(c *domain.AgentConfig) bool { return c.Agent.SystemPrompt == "First line\nSecond line\n" }},
		{"agent.temperature", "0.7", func(c *domain.AgentConfig) bool { return c.Agent.Temperature == 0.7 }},
		{"agent.max_output_tokens", "4096", func(c *domain.AgentConfig) bool {
			return c.Agent.MaxOutputTokens != nil && *c.Agent.MaxOutputTokens == 4096
		}},
		{"agent.reasoning", "false", func(c *domain.AgentConfig) bool { return c.Agent.Reasoning != nil && !*c.Agent.Reasoning }},
		{"agent.reasoning", "true", func(c *domain.AgentConfig) bool { return c.Agent.Reasoning != nil && *c.Agent.Reasoning }},
		{"agent.memory.path", ".new_history.json", func(c *domain.AgentConfig) bool { return c.Agent.Memory.Path == ".new_history.json" }},
		{"agent.memory", "null", func(c *domain.AgentConfig) bool { return c.Agent.Memory == nil }},
	} {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			service, path := editableConfig(t, editableAgent)
			if err := service.SetAgentValue(context.Background(), "example", tc.field, tc.value); err != nil {
				t.Fatal(err)
			}
			configuration, err := service.LoadAgent(context.Background(), "example")
			if err != nil || !tc.check(configuration) {
				t.Fatalf("unexpected config after update: %+v, %v", configuration, err)
			}
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, comment := range []string{"# Keep this document comment.", "# Keep this model comment."} {
				if !bytes.Contains(content, []byte(comment)) {
					t.Errorf("comment was lost: %s", comment)
				}
			}
			if tc.field == "agent.model" {
				var document yaml.Node
				if err := yaml.Unmarshal(content, &document); err != nil {
					t.Fatal(err)
				}
				node, err := yamlPath(document.Content[0], []string{"agent", "model"})
				if err != nil || node.Tag != "!!str" {
					t.Fatalf("numeric-looking model must remain a YAML string: %v, %v", node, err)
				}
			}
			assertNoTemporaryFiles(t, filepath.Dir(path))
		})
	}
}

func TestSetAgentValueListsMapsAndOptionalValues(t *testing.T) {
	service, _ := editableConfig(t, editableAgent)
	for _, update := range [][2]string{
		{"agent.tools", "[{type: builtin, name: read_file}]"},
		{"agent.tools.0.name", "write_file"},
		{"agent.tools.0.allow_without_confirm", "true"},
		{"agent.tools.0.config", "{timeout: 10, paths: [src, tests]}"},
		{"agent.tools.0.config.timeout", "20"},
		{"agent.tools.0.config.paths.1", "lib"},
		{"agent.max_output_tokens", "1024"},
		{"agent.max_output_tokens", "null"},
		{"agent.reasoning", "true"},
		{"agent.reasoning", "null"},
		{"agent.memory", "null"},
		{"agent.memory", "{type: local_file, path: .state.json}"},
	} {
		if err := service.SetAgentValue(context.Background(), "example", update[0], update[1]); err != nil {
			t.Fatalf("set %s=%s: %v", update[0], update[1], err)
		}
	}
	configuration, err := service.LoadAgent(context.Background(), "example")
	if err != nil {
		t.Fatal(err)
	}
	agent := configuration.Agent
	if len(agent.Tools) != 1 || agent.Tools[0].Name != "write_file" || !agent.Tools[0].AllowWithoutConfirm {
		t.Fatalf("unexpected tools: %+v", agent.Tools)
	}
	toolConfig := agent.Tools[0].Config
	if toolConfig["timeout"] != 20 || toolConfig["paths"].([]any)[1] != "lib" {
		t.Fatalf("unexpected tool configuration: %+v", toolConfig)
	}
	if agent.MaxOutputTokens != nil || agent.Reasoning != nil || agent.Memory.Path != ".state.json" {
		t.Fatalf("optional values were not updated: %+v", agent)
	}
}

func TestSetAgentValueFailuresLeaveFileUntouched(t *testing.T) {
	for _, update := range [][2]string{
		{"", "x"}, {"agent..model", "x"}, {"agent. model", "x"},
		{"agent.unknown", "x"}, {"path", "x"}, {"agent.model.name", "x"},
		{"agent.name", "other"}, {"version", "2.0"}, {"agent.provider", "unknown"},
		{"agent.model", "two words"}, {"agent.system_prompt", ""},
		{"agent.temperature", "3"}, {"agent.temperature", "null"}, {"agent.temperature", "\"0.4\""},
		{"agent.temperature", ".nan"}, {"agent.temperature", ".inf"},
		{"agent.max_output_tokens", "1.5"}, {"agent.max_output_tokens", "0"},
		{"agent.max_output_tokens", "999999999999999999999999999999"},
		{"agent.reasoning", "yes"}, {"agent.reasoning", "true\n---\nfalse"},
		{"agent.memory", "[bad"}, {"agent.memory", "{type: local_file, path: .state, unknown: x}"},
		{"agent.memory.path", "../outside"}, {"agent.think", "true"},
		{"agent.tools.0.name", "read_file"}, {"agent.tools.-1.name", "read_file"},
		{"agent.tools.x.name", "read_file"},
	} {
		t.Run(update[0]+"="+update[1], func(t *testing.T) {
			service, path := editableConfig(t, editableAgent)
			if err := service.SetAgentValue(context.Background(), "example", update[0], update[1]); err == nil {
				t.Fatal("invalid update succeeded")
			}
			content, err := os.ReadFile(path)
			if err != nil || string(content) != editableAgent {
				t.Fatalf("invalid update changed the file: %v", err)
			}
			assertNoTemporaryFiles(t, filepath.Dir(path))
		})
	}
}

func TestSetAgentValueRepairsInvalidField(t *testing.T) {
	service, _ := editableConfig(t, strings.Replace(editableAgent, "temperature: 0.2", "temperature: 3", 1))
	if err := service.SetAgentValue(context.Background(), "example", "agent.temperature", "0.5"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoadAgent(context.Background(), "example"); err != nil {
		t.Fatal(err)
	}
}

func TestSetAgentValueRejectsTrailingDocumentsAndSharedAnchors(t *testing.T) {
	for _, original := range []string{
		editableAgent + "---\nagent: {}\n",
		strings.Replace(editableAgent, "model: gpt-4o", "model: &shared gpt-4o", 1),
		strings.Replace(editableAgent, "agent:\n", "agent: &shared\n", 1),
	} {
		service, path := editableConfig(t, original)
		if err := service.SetAgentValue(context.Background(), "example", "agent.model", "new"); err == nil {
			t.Fatal("invalid document or anchored update was accepted")
		}
		content, err := os.ReadFile(path)
		if err != nil || string(content) != original {
			t.Fatalf("rejected update changed the file: %v", err)
		}
	}
}

func TestSetAgentValueCancellationAndTraversal(t *testing.T) {
	service, path := editableConfig(t, editableAgent)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.SetAgentValue(ctx, "example", "agent.model", "new"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled update = %v", err)
	}
	if err := service.SetAgentValue(context.Background(), "../example", "agent.model", "new"); err == nil {
		t.Fatal("traversing agent name was accepted")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != editableAgent {
		t.Fatalf("file changed: %v", err)
	}
}

func TestSetAgentValueRejectsSiblingSymlink(t *testing.T) {
	service, path := editableConfig(t, editableAgent)
	target := filepath.Join(filepath.Dir(filepath.Dir(path)), "other.yml")
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../other.yml", path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := service.SetAgentValue(context.Background(), "example", "agent.model", "new"); err == nil {
		t.Fatal("updated a config outside the selected agent root")
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != editableAgent {
		t.Fatalf("sibling config changed: %v", err)
	}
}

func assertNoTemporaryFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != AgentFile {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}
