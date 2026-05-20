package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/generator"
	"github.com/haivh2111/goplate/internal/project"
)

func newNewAdapterCmd() *cobra.Command {
	var (
		methods      string
		stubSiblings bool
	)
	cmd := &cobra.Command{
		Use:   "new-adapter <service> <provider>",
		Short: "Generate port interface + concrete adapter implementation for an external service",
		Long: `Generates internal/adapters/<service>/port.go plus a concrete provider
implementation under internal/adapters/<service>/<provider>/adapter.go.

If port.go already exists (typical when adding a second provider to the same
service), the generator AST-merges any new methods into the existing interface
and leaves your edits intact.

--methods accepts two forms:
  NAMES ONLY:   "Send,SendBulk"  → each becomes <Name>(ctx context.Context) error
  FULL SIGS:    "CreateCharge(req ChargeRequest) (*ChargeResponse, error); RefundCharge(id string) error"
                (use ';' or newlines to separate methods; commas conflict with params)`,
		Args: cobra.ExactArgs(2),
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

			opts := generator.AdapterOptions{
				Service:      args[0],
				Provider:     args[1],
				MethodsSpec:  methods,
				ModulePath:   modulePath,
				ProjectRoot:  cwd,
				StubSiblings: stubSiblings,
			}
			files, notice, err := generator.GenerateAdapter(opts)
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
				cmd.Print(generator.AdapterNextSteps(opts))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&methods, "methods", "m", "",
		"Method names or Go signatures for the port interface (see --help for syntax)")
	cmd.Flags().BoolVar(&stubSiblings, "stub-siblings", false,
		"When port.go gains new methods, auto-append panic stubs to existing sibling providers' adapter.go")
	_ = cmd.MarkFlagRequired("methods")
	return cmd
}
