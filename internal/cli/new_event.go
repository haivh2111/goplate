package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/generator"
	"github.com/haivh2111/goplate/internal/project"
)

func newNewEventCmd() *cobra.Command {
	var (
		payload    string
		subscriber string
	)
	cmd := &cobra.Command{
		Use:   "new-event <EventName>",
		Short: "Register a domain event and optionally scaffold a subscriber in a target feature",
		Long: `Appends a new constant + payload struct to internal/events/event_types.go
(creating the file if missing). When --subscriber is given, also creates or
inserts a Subscribe block into internal/features/<subscriber>/subscribers.go.

Re-running with the same event name is idempotent — already-declared constants
and existing Subscribe blocks are left untouched.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			modulePath, err := project.DetectModulePath(cwd)
			if err != nil {
				if errors.Is(err, project.ErrNoGoMod) {
					return fmt.Errorf("not inside a Go module — run from your project root (the directory containing go.mod)")
				}
				return err
			}

			opts := generator.EventOptions{
				EventName:   args[0],
				PayloadSpec: payload,
				Subscriber:  subscriber,
				ModulePath:  modulePath,
				ProjectRoot: cwd,
			}
			files, notice, err := generator.GenerateEvent(opts)
			if err != nil {
				return err
			}

			var w generator.Writer
			if DryRun() {
				w = generator.DryRunWriter{Out: cmd.OutOrStdout()}
			} else {
				w = generator.FSWriter{Root: cwd, Stdout: cmd.OutOrStdout()}
			}
			if err := w.Write(files); err != nil {
				return err
			}
			if notice != "" {
				cmd.Print(notice)
			}
			if !DryRun() {
				cmd.Print(generator.EventNextSteps(opts))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&payload, "payload", "p", "",
		"Comma-separated payload fields, e.g. \"OrderID:uint,UserID:uint,Total:float64\"")
	cmd.Flags().StringVarP(&subscriber, "subscriber", "s", "",
		"Feature that subscribes to this event (creates/updates subscribers.go)")
	return cmd
}
