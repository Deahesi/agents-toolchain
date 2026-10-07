package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "agent-toolchain",
	Short:        "Manage and run YAML-configured agents",
	SilenceUsage: true,
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	godotenv.Load()
	rootCmd.PersistentFlags().String("workspace-dir", ".", "workspace directory")
}
