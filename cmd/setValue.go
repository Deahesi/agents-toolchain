package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/spf13/cobra"
)

var setValueCmd = &cobra.Command{
	Use:   "set-value <agent-name> <field> <value>",
	Short: "Update an agent config field by its YAML path",
	Long:  "Update a dot-separated YAML path, such as agent.model or agent.tools.0.name. Strings are literal text; other values use YAML. Optional values can be cleared with null.",
	Example: `  atc set-value docs agent.temperature 0.4
  atc set-value docs agent.model gpt-4o
  atc set-value docs agent.reasoning true`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspaceDir, err := cmd.Flags().GetString("workspace-dir")
		if err != nil {
			return err
		}
		uiService := ui.NewUIService(ui.WithInput(cmd.InOrStdin()), ui.WithOutput(cmd.OutOrStdout()), ui.WithErrorOutput(cmd.ErrOrStderr()), ui.WithContext(cmd.Context()))
		service := config.NewConfigService(uiService, workspaceDir)
		if err := service.SetAgentValue(uiService.Context(), args[0], args[1], args[2]); err != nil {
			return err
		}
		uiService.LogSuccess("Updated ", args[0], ": ", args[1])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setValueCmd)
}
