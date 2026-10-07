package cmd

import (
	"github.com/Deahesi/agents-toolchain/internal/config"
	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/pterm/pterm"
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
		pterm.DefaultSection.WithWriter(output).Println("Project initialization")
		pterm.DefaultBasicText.WithWriter(output).Printfln("Workspace: %s\nAgents:    %s\n", workspaceDir, agentsDir)

		uiService := ui.NewUIService()
		configService := config.NewConfigService(uiService, workspaceDir)

		path, err := configService.Init(cmd.Context(), agentsDir)
		if err != nil {
			pterm.Error.WithWriter(cmd.ErrOrStderr()).Println(err)
			cmd.SilenceErrors = true
			return err
		}

		pterm.Success.WithWriter(output).Printfln("Project initialized: %s", path)
		return nil
	},
}

func init() {
	initCmd.Flags().String("agents-dir", "agents", "directory for agents configs")
	rootCmd.AddCommand(initCmd)
}
