package system

import (
	"os"
	"runtime"
	"strings"

	"github.com/Deahesi/agents-toolchain/internal/domain"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

type GetEnvironmentInput struct{}

type GetEnvironmentOutput struct {
	Success bool              `json:"success"`
	OS      string            `json:"os"`
	Arch    string            `json:"arch"`
	EnvVars map[string]string `json:"env_vars"`
}

func DefineGetEnvironmentTool(g *genkit.Genkit, tool *domain.ToolConfig) *ai.ToolAction[GetEnvironmentInput, GetEnvironmentOutput] {
	return genkit.DefineTool(
		g,
		"get_environment",
		"Returns OS, architecture, and safe non-sensitive environment variables.",
		func(ctx *ai.ToolContext, input GetEnvironmentInput) (GetEnvironmentOutput, error) {
			allowedKeys := map[string]bool{
				"PATH": true, "USER": true, "HOME": true,
				"SHELL": true, "PWD": true, "LANG": true,
			}

			safeEnv := make(map[string]string)
			for _, env := range os.Environ() {
				pair := strings.SplitN(env, "=", 2)
				if len(pair) == 2 {
					key := pair[0]

					if allowedKeys[key] || strings.HasPrefix(key, "GO") {
						safeEnv[key] = pair[1]
					}
				}
			}

			return GetEnvironmentOutput{
				Success: true,
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				EnvVars: safeEnv,
			}, nil
		},
	)
}
