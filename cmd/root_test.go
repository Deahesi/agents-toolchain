package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
)

func TestExecuteFormatsErrorsOnce(t *testing.T) {
	failure := errors.New("operation failed")
	for _, tc := range []struct {
		name           string
		args           []string
		runError       error
		preRunError    error
		requiredFlag   bool
		childErrorSink bool
	}{
		{name: "RunE", args: []string{"sample", "agent"}, runError: failure},
		{name: "PreRunE", args: []string{"sample", "agent"}, preRunError: failure},
		{name: "argument validation", args: []string{"sample"}},
		{name: "unknown command", args: []string{"unknown"}},
		{name: "unknown flag", args: []string{"sample", "agent", "--unknown"}},
		{name: "invalid flag value", args: []string{"sample", "agent", "--count=bad"}},
		{name: "required flag", args: []string{"sample", "agent"}, requiredFlag: true},
		{name: "child error stream", args: []string{"sample", "agent"}, runError: failure, childErrorSink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output, errorOutput, childErrors bytes.Buffer
			root := &cobra.Command{
				Use:           "atc",
				SilenceErrors: rootCmd.SilenceErrors,
				SilenceUsage:  rootCmd.SilenceUsage,
			}
			root.SetOut(&output)
			root.SetErr(&errorOutput)
			root.SetArgs(tc.args)
			child := &cobra.Command{
				Use:  "sample <agent>",
				Args: cobra.ExactArgs(1),
				PreRunE: func(*cobra.Command, []string) error {
					return tc.preRunError
				},
				RunE: func(*cobra.Command, []string) error {
					return tc.runError
				},
			}
			child.Flags().Int("count", 0, "count")
			if tc.requiredFlag {
				if err := child.MarkFlagRequired("count"); err != nil {
					t.Fatal(err)
				}
			}
			if tc.childErrorSink {
				child.SetErr(&childErrors)
			}
			root.AddCommand(child)
			err := execute(context.Background(), root)
			if err == nil {
				t.Fatal("command failure was swallowed")
			}
			if (tc.runError != nil || tc.preRunError != nil) && !errors.Is(err, failure) {
				t.Fatalf("error identity was lost: %v", err)
			}
			got := errorOutput.String()
			if tc.childErrorSink {
				if errorOutput.Len() != 0 {
					t.Fatalf("error also printed to root stream: %q", errorOutput.String())
				}
				got = childErrors.String()
			}
			if got, want := ansi.Strip(got), "✗ "+err.Error()+"\n"; got != want {
				t.Fatalf("stderr = %q; want one UI-formatted error %q", got, want)
			}
			if output.Len() != 0 {
				t.Fatalf("error or usage leaked to stdout: %q", output.String())
			}
		})
	}
}

func TestExecuteSuccessAndHelp(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}} {
		var output, errorOutput bytes.Buffer
		root := &cobra.Command{
			Use:           "atc",
			SilenceErrors: rootCmd.SilenceErrors,
			SilenceUsage:  rootCmd.SilenceUsage,
			RunE:          func(*cobra.Command, []string) error { return nil },
		}
		root.SetOut(&output)
		root.SetErr(&errorOutput)
		root.SetArgs(args)
		if err := execute(context.Background(), root); err != nil {
			t.Fatal(err)
		}
		if errorOutput.Len() != 0 {
			t.Fatalf("successful command printed an error: %q", errorOutput.String())
		}
		if len(args) > 0 && !strings.Contains(output.String(), "Usage:") {
			t.Fatalf("help output missing: %q", output.String())
		}
	}
}

func TestCommandHandlersReturnErrorsWithoutLoggingThem(t *testing.T) {
	for _, command := range []*cobra.Command{addCmd, checkCmd, initCmd, listCmd, runCmd} {
		t.Run(command.Name(), func(t *testing.T) {
			var output, errorOutput bytes.Buffer
			invocation := &cobra.Command{Use: command.Use}
			invocation.SetContext(context.Background())
			invocation.SetOut(&output)
			invocation.SetErr(&errorOutput)
			invocation.Flags().String("workspace-dir", t.TempDir(), "workspace")
			invocation.Flags().String("prompt", "", "prompt")
			invocation.Flags().String("agents-dir", "../outside", "agents directory")
			if err := command.RunE(invocation, []string{"example"}); err == nil {
				t.Fatal("command must return its error to the shared handler")
			}
			if errorOutput.Len() != 0 {
				t.Fatalf("command logged an error before returning it: %q", errorOutput.String())
			}
		})
	}
}
