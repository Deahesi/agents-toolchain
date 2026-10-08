package system

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type ExecuteCommandInput struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
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
		"Executes a CLI command with arguments. Use separate args slice, do not string-bash.",
		func(ctx *ai.ToolContext, input ExecuteCommandInput) (ExecuteCommandOutput, error) {
			if input.Command == "" {
				return ExecuteCommandOutput{Success: false, Message: "command cannot be empty"}, nil
			}

			timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(timeoutCtx, input.Command, input.Args...)

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
