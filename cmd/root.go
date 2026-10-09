package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/Deahesi/agents-toolchain/internal/ui"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:           "atc",
	Short:         "Manage and run YAML-configured agents",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := execute(ctx, rootCmd)
	stop()
	if err != nil {
		os.Exit(1)
	}
}

// Report command, argument and flag errors once, using the selected command's
// output streams. Keep the error available to the caller for the exit status.
func execute(ctx context.Context, root *cobra.Command) error {
	command, err := root.ExecuteContextC(ctx)
	if err == nil {
		return nil
	}
	if command == nil {
		command = root
	}
	uiService := ui.NewUIService(
		ui.WithInput(command.InOrStdin()),
		ui.WithOutput(command.OutOrStdout()),
		ui.WithErrorOutput(command.ErrOrStderr()),
		ui.WithContext(ctx),
	)
	uiService.LogError(err)
	return err
}

func init() {
	godotenv.Load()
	rootCmd.PersistentFlags().String("workspace-dir", ".", "workspace directory")
}
