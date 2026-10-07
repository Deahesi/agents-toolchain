package cmd

import (
	"fmt"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create agents-toolchain.yml and the agents directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		agentsDir, err := cmd.Flags().GetString("agents-dir")
		if err != nil {
			return err
		}
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}

		output := cmd.OutOrStdout()
		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(output), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		uiService.LogStep("Project initialization")
		uiService.LogStep(fmt.Sprintf("Workspace: %s\nAgents:    %s\n", workspaceDir, agentsDir))
		configService := config.NewConfigService(uiService, workspaceDir)

		path, err := configService.Init(uiService.Context(), agentsDir)
		if err != nil {
			uiService.LogError(err)
			cmd.SilenceErrors = true
			return err
		}

		uiService.LogSuccess("Project initialized: ", path)
		return nil
	},
}

func init() {
	initCmd.Flags().String("agents-dir", "agents", "directory for agents configs")
	rootCmd.AddCommand(initCmd)
}
