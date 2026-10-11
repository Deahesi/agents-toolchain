package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestPrintFieldsAlignsUnicodeAndMultilineValues(t *testing.T) {
	var output, errorOutput bytes.Buffer
	s := NewUIService(WithOutput(&output), WithErrorOutput(&errorOutput))
	s.PrintFields("Agent: docs", []Field{
		{Label: "名", Value: "文档"},
		{Label: "Description", Value: "First line\nSecond line"},
		{Label: "Empty", Value: ""},
	})
	text := output.String()
	if strings.Contains(text, "\x1b") || errorOutput.Len() != 0 {
		t.Fatalf("redirected output must be plain text on stdout: %q, stderr=%q", text, errorOutput.String())
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 5 || lines[0] != "Agent: docs" {
		t.Fatalf("unexpected block: %q", text)
	}
	for i, value := range []string{"文档", "First line", "Second line"} {
		line := lines[i+1]
		index := strings.Index(line, value)
		if index < 0 {
			t.Fatalf("value %q missing from %q", value, line)
		}
		if width := ansi.StringWidth(line[:index]); width != 15 {
			t.Errorf("value %q starts at column %d; want 15", value, width)
		}
	}
	if strings.TrimSpace(lines[3]) != "Second line" || strings.TrimSpace(lines[4]) != "Empty" {
		t.Fatalf("continuation or empty field was lost: %q", text)
	}
}

func TestPrintTableKeepsRaggedRowsAndMultilineCells(t *testing.T) {
	var output bytes.Buffer
	s := NewUIService(WithOutput(&output))
	// Spare header capacity must not be overwritten when adding blank headers.
	backing := []string{"NAME", "MODEL", "sentinel"}
	headers := backing[:2]
	rows := [][]string{
		{"docs", "gpt-4o\nfallback"},
		{"review"},
		{"界", "qwen", "extra"},
	}
	before := [][]string{
		{"docs", "gpt-4o\nfallback"},
		{"review"},
		{"界", "qwen", "extra"},
	}
	s.PrintTable(headers, rows)
	text := output.String()
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if len(lines) != 5 || strings.Contains(text, "\x1b") {
		t.Fatalf("unexpected plain table: %q", text)
	}
	for _, item := range []struct {
		line  int
		value string
	}{
		{0, "MODEL"}, {1, "gpt-4o"}, {2, "fallback"}, {4, "qwen"},
	} {
		line := lines[item.line]
		index := strings.Index(line, item.value)
		if index < 0 || ansi.StringWidth(line[:index]) != 8 {
			t.Errorf("model column misaligned or lost: %q in %q", item.value, line)
		}
	}
	if strings.TrimSpace(lines[2]) != "fallback" || !strings.Contains(lines[3], "review") || !strings.Contains(lines[4], "extra") {
		t.Fatalf("missing cells, extra cells or continuations changed: %q", text)
	}
	if backing[2] != "sentinel" || !reflect.DeepEqual(rows, before) {
		t.Fatalf("input was modified: headers=%q, rows=%q", backing, rows)
	}
}

func TestStructuredOutputEmptyAndOptionalHeaders(t *testing.T) {
	for _, tc := range []struct {
		name  string
		print func(*UIService)
		want  string
	}{
		{"empty fields", func(s *UIService) { s.PrintFields("", nil) }, ""},
		{"title only", func(s *UIService) { s.PrintFields("Agent", nil) }, "Agent\n"},
		{"empty table", func(s *UIService) { s.PrintTable(nil, nil) }, ""},
		{"empty rows", func(s *UIService) { s.PrintTable(nil, [][]string{nil, {}}) }, ""},
		{"headers only", func(s *UIService) { s.PrintTable([]string{"NAME"}, nil) }, "NAME\n"},
		{"rows only", func(s *UIService) { s.PrintTable(nil, [][]string{{"docs"}, {"review"}}) }, "docs\nreview\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			tc.print(NewUIService(WithOutput(&output)))
			// Table cells may contain alignment spaces at the end of a line.
			lines := strings.Split(output.String(), "\n")
			for i := range lines {
				lines[i] = strings.TrimRight(lines[i], " ")
			}
			if got := strings.Join(lines, "\n"); got != tc.want {
				t.Fatalf("output = %q; want %q", got, tc.want)
			}
		})
	}
}
