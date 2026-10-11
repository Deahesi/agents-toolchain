/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check <agent-name>",
	Short: "Check an agent configuration for errors",
	Long:  "Check an agent's YAML syntax, config values, and whether its name matches its directory. Prints Valid on success or reports an error. This command does not run the agent or call the model provider.",
	Example: `  atc check hello
  atc check hello --workspace-dir ./my-project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}

		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		configService := config.NewConfigService(uiService, workspaceDir)

		config, err := configService.LoadAgent(uiService.Context(), args[0])

		if err != nil {
			return err
		}

		err = config.Validate()

		if err == nil {
			uiService.LogSuccess("Valid")
		}

		return err
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
