/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}

		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		configService := config.NewConfigService(uiService, workspaceDir)

		agents, err := configService.GetAllAgents(uiService.Context())
		if err != nil {
			return err
		}

		for _, agent := range agents {
			uiService.PrintFields(agent.Agent.Name, []ui.Field{
				{
					Label: "Description",
					Value: agent.Agent.Description,
				},
				{
					Label: "Provider",
					Value: agent.Agent.Provider,
				},
				{
					Label: "Model",
					Value: agent.Agent.Model,
				},
			})
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
