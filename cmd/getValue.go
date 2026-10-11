/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

var getValueCmd = &cobra.Command{
	Use:   "get-value <agent-name> <field>",
	Short: "Read an agent config field by its YAML path",
	Long:  "Read a scalar value from an agent configuration using a dot-separated YAML path, such as agent.model or agent.tools.0.name. List indices start at zero. The value is displayed in a table with the field path as its heading.",
	Example: `  atc get-value docs agent.model
  atc get-value docs agent.temperature
  atc get-value docs agent.tools.0.name`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}
		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		service := config.NewConfigService(uiService, workspaceDir)

		value, err := service.GetAgentValue(uiService.Context(), args[0], args[1])
		if err != nil {
			return err
		}
		uiService.PrintTable([]string{args[1]}, [][]string{{value}})

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getValueCmd)
}
