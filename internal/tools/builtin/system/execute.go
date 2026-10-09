package system

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ExecuteCommandInput struct {
	Command string `json:"command" jsonschema_description:"Complete shell command including arguments, for example git diff --staged. Use sh syntax on Linux/macOS and PowerShell syntax on Windows."`
}
type ExecuteCommandOutput struct {
	Success bool   `json:"success"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
	Message string `json:"message,omitempty"`
}

func DefineExecuteCommandTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[ExecuteCommandInput, ExecuteCommandOutput] {
	return genkit.DefineTool(
		g,
		"execute_command",
		`Runs a complete command string through sh on Linux/macOS or PowerShell on Windows. Example: {"command":"git diff --staged"}. Supports shell quoting, pipes and redirection. Use syntax appropriate to the OS; Windows PowerShell does not support &&. Commands run in the current working directory with a 30-second timeout.`,
		func(ctx *ai.ToolContext, input ExecuteCommandInput) (ExecuteCommandOutput, error) {
			if strings.TrimSpace(input.Command) == "" {
				return ExecuteCommandOutput{Success: false, Message: "command cannot be empty"}, nil
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(timeoutCtx, "sh", "-c", input.Command)
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(timeoutCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", input.Command)
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if timeoutCtx.Err() == context.DeadlineExceeded {
				return ExecuteCommandOutput{
					Success: false,
					Message: "command timed out after 30 seconds",
					Stdout:  stdout.String(),
					Stderr:  stderr.String(),
				}, nil
			}

			if err != nil {
				return ExecuteCommandOutput{
					Success: false,
					Message: err.Error(),
					Stdout:  stdout.String(),
					Stderr:  stderr.String(),
				}, nil
			}

			return ExecuteCommandOutput{
				Success: true,
				Stdout:  stdout.String(),
				Stderr:  stderr.String(),
			}, nil
		},
	)
}
