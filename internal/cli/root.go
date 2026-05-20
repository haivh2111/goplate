// Package cli composes the cobra command tree for the goplate binary.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/version"
)

// NewRootCmd builds and returns the goplate root command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "goplate",
		Short:   "goplate — companion CLI for go-rest-api-boilerplate",
		Long:    "Bootstraps new projects from the boilerplate template and scaffolds features, adapters, and domain events that follow the project's architectural rules.",
		Version: version.Version,

		// We print errors ourselves in Execute(); don't let cobra spam usage.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&globalFlags.DryRun, "dry-run", false,
		"Preview generated files without writing to disk")

	root.AddCommand(
		newDoctorCmd(),
		newNewCmd(),
		newNewFeatureCmd(),
		newNewAdapterCmd(),
		newNewEventCmd(),
	)
	return root
}

// Execute runs the root command and returns its error. main() turns a non-nil
// result into a non-zero exit code.
func Execute() error {
	cmd := NewRootCmd()
	err := cmd.Execute()
	if err != nil {
		cmd.PrintErrln("Error:", err)
	}
	return err
}
