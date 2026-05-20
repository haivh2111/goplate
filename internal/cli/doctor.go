package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/doctor"
)

// errDoctorUnsatisfied causes a non-zero exit when doctor (without --fix)
// finds missing prerequisites.
var errDoctorUnsatisfied = errors.New("prerequisites missing")

func newDoctorCmd() *cobra.Command {
	var fix bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify host has the prerequisite toolchain (go, git, air, swag, mockery, golangci-lint)",
		Long:  "Checks that every tool goplate-generated projects depend on is installed and reachable on PATH. With --fix, runs `go install` for any missing tool that supports automated installation.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			results := doctor.Run(ctx, nil, doctor.DefaultChecks())
			ok := doctor.Report(cmd.OutOrStdout(), results)
			if !fix {
				if !ok {
					return errDoctorUnsatisfied
				}
				return nil
			}
			cmd.Println()
			if err := doctor.Fix(ctx, nil, results, cmd.OutOrStdout()); err != nil {
				return err
			}
			cmd.Println("  All fixable tools installed.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to `go install` any missing tools")
	return cmd
}
