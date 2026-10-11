package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
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
		return agentService.Run(uiService.Context(), args[0], prompt, cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringP("prompt", "p", "", "User prompt; defaults to the task in system_prompt")
}
