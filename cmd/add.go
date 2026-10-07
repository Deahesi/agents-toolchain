package cmd

import (
	"fmt"

	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/pterm/pterm"
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

		uiService := ui.NewUIService()
		configService := config.NewConfigService(uiService, workspaceDir)

		path, err := configService.CreateAgent(cmd.Context(), args[0])
		if err != nil {
			pterm.Error.WithWriter(cmd.ErrOrStderr()).Println(err)
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), "Created", path)
		return err
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
