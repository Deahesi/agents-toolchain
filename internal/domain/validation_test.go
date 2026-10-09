package domain

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/validation"
)

func TestReusableValidationRules(t *testing.T) {
	message := "custom validation error"
	positive, zero := 32, 0
	text := "text"
	var missing *int
	for _, tc := range []struct {
		name    string
		rule    validation.Rule
		valid   []any
		invalid []any
	}{
		{"required text", requiredText(message), []any{"text", ToolTypeBuiltin, &text}, []any{nil, " \t", 42}},
		{"local path", localPath(message), []any{"agents", "agents/docs"}, []any{nil, "", "../outside", 42}},
		{"allowed values", oneOf(message, "a", "b"), []any{"a", "b"}, []any{"c", 42}},
		{"whitespace", withoutWhitespace(message), []any{nil, "model:latest", ProviderKey("openai")}, []any{"two words", "model\tname", "model\r\n", 42}},
		{"finite number", finiteNumber(message), []any{nil, missing, 0, uint64(8), float32(0.2), 0.2, &positive}, []any{math.NaN(), math.Inf(1), math.Inf(-1), "number"}},
		{"positive integer", positiveInteger(message), []any{nil, missing, 1, uint64(8), &positive}, []any{0, -1, uint(0), &zero, 1.5, "32"}},
		{"unique strings", uniqueBy(message, func(s string) string { return s }), []any{nil, []string{}, []string{"a", "b"}}, []any{[]string{"a", "a"}, 42}},
		{"unique tool names", uniqueBy(message, func(tool ToolConfig) string { return tool.Name }), []any{[]ToolConfig{{Name: "a"}, {Name: "b"}}}, []any{[]ToolConfig{{Name: "a"}, {Name: "a"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, value := range tc.valid {
				if err := tc.rule.Validate(value); err != nil {
					t.Errorf("valid value %#v: %v", value, err)
				}
			}
			for _, value := range tc.invalid {
				if err := tc.rule.Validate(value); err == nil || !strings.Contains(err.Error(), message) {
					t.Errorf("invalid value %#v: error = %v, want custom message", value, err)
				}
			}
		})
	}
}

func TestValidateWorkDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "directory with spaces")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		path  any
		valid bool
	}{
		{"absolute directory with spaces", dir, true},
		{"relative directory", ".", true},
		{"parent directory", "..", true},
		{"directory pointer", &dir, true},
		{"named string type", ProviderKey(dir), true},
		{"nil", nil, false},
		{"empty", "", false},
		{"whitespace", " \t", false},
		{"file", file, false},
		{"missing directory", filepath.Join(dir, "missing"), false},
		{"invalid path", "invalid\x00path", false},
		{"wrong type", 42, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorkDir(tc.path)
			if (err == nil) != tc.valid {
				t.Fatalf("ValidateWorkDir(%v) = %v, want valid=%v", tc.path, err, tc.valid)
			}
			if err != nil && !strings.Contains(err.Error(), "Work dir") {
				t.Errorf("error lacks field context: %v", err)
			}
		})
	}
}

func TestAgentOutputTokenValidation(t *testing.T) {
	zero, negative, positive := 0, -1, 32
	for _, limit := range []*int{nil, &zero, &negative, &positive} {
		agent := Agent{Name: "test", Provider: "ollama", Model: "qwen3", WorkDir: ".", SystemPrompt: "Test", MaxOutputTokens: limit}
		err := agent.Validate()
		valid := limit == nil || *limit > 0
		if valid && err != nil {
			t.Errorf("valid limit %v: %v", limit, err)
		} else if !valid && (err == nil || !strings.Contains(err.Error(), "max_output_tokens")) {
			t.Errorf("invalid limit %v: error = %v", *limit, err)
		}
	}
}
