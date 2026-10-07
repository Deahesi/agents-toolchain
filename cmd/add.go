package cmd

import (
	"fmt"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <agent-name>",
	Short: "Create an agent config in the configured directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}

		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		configService := config.NewConfigService(uiService, workspaceDir)

		path, err := configService.CreateAgent(uiService.Context(), args[0])
		if err != nil {
			uiService.LogError(err)
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Created", path)
		return err
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
