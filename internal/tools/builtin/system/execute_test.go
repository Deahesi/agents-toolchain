package system

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/genkit"
)

func TestExecuteCommand(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "output with spaces.txt")
	quotedPath := "'" + strings.ReplaceAll(outputPath, "'", "'\\''") + "'"
	pipe := "printf 'hello world' | tr a-z A-Z"
	redirect := "printf 'saved value' > " + quotedPath
	failure := "printf 'partial output'; printf 'failure details' >&2; exit 7"
	if runtime.GOOS == "windows" {
		quotedPath = "'" + strings.ReplaceAll(outputPath, "'", "''") + "'"
		pipe = "Write-Output 'hello world' | ForEach-Object { $_.ToUpper() }"
		redirect = "Set-Content -LiteralPath " + quotedPath + " -Value 'saved value' -NoNewline -Encoding ascii"
		failure = "[Console]::Out.Write('partial output'); [Console]::Error.Write('failure details'); exit 7"
	}

	ctx := context.Background()
	tool := DefineExecuteCommandTool(genkit.Init(ctx), nil)
	for _, tc := range []struct {
		name    string
		command string
		success bool
		stdout  string
		stderr  string
		message string
	}{
		{name: "arguments and quotes", command: `echo "hello world"`, success: true, stdout: "hello world"},
		{name: "pipeline", command: pipe, success: true, stdout: "HELLO WORLD"},
		{name: "redirection to path with spaces", command: redirect, success: true},
		{name: "nonzero exit", command: failure, stdout: "partial output", stderr: "failure details", message: "exit status 7"},
		{name: "empty command", message: "command cannot be empty"},
		{name: "whitespace command", command: " \t\n", message: "command cannot be empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := json.Marshal(ExecuteCommandInput{Command: tc.command})
			if err != nil {
				t.Fatal(err)
			}
			result, err := tool.RunJSON(ctx, input, nil)
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				Output ExecuteCommandOutput `json:"output"`
			}
			if err := json.Unmarshal(result, &response); err != nil {
				t.Fatal(err)
			}
			output := response.Output
			if output.Success != tc.success || strings.TrimSpace(output.Stdout) != tc.stdout || strings.TrimSpace(output.Stderr) != tc.stderr || output.Message != tc.message {
				t.Fatalf("unexpected output: %+v", output)
			}
		})
	}
	contents, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "saved value" {
		t.Fatalf("redirected contents = %q", contents)
	}
}
