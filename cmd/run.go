package cmd

import (
	"path/filepath"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/memory"
	"github.com/Deahesi/agents-toolchain/internal/runtime"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <agent-name>",
	Short: "Run an agent and stream its response",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}

		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		configService := config.NewConfigService(uiService, workspaceDir)
		agentService := runtime.NewRuntimeService(uiService, configService)

		prompt, err := cmd.Flags().GetString("prompt")
		if err != nil {
			return err
		}

		//CONFIG
		config, err := configService.LoadAgent(uiService.Context(), args[0])
		if err != nil {
			return err
		}
		err = config.Validate()
		if err != nil {
			return err
		}

		//MEMORY

		project, err := configService.GetProjectConfig()
		if err != nil {
			return err
		}
		store, err := memory.OpenMemoryStore(
			config.Agent.Memory,
			filepath.Join(project.AgentsDir, config.Agent.Name),
		)
		if err != nil {
			return err
		}

		var runtimeMemory runtime.MemoryStore = nil

		if store != nil {
			runtimeMemory = store
			defer store.Close()
		}

		return agentService.Run(
			uiService.Context(),
			prompt,
			config,
			runtimeMemory,
		)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("prompt", "p", "", "User prompt; defaults to the task in system_prompt")
}
