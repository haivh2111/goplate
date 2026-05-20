package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/generator"
	"github.com/haivh2111/goplate/internal/project"
)

func newNewFeatureCmd() *cobra.Command {
	var (
		fieldsSpec   string
		noAuth       bool
		noPagination bool
	)
	cmd := &cobra.Command{
		Use:   "new-feature <name>",
		Short: "Generate all files for a new business feature (model → repo → service → handler → tests)",
		Long: `Generates the complete file set for a new feature under
internal/features/<name>/. Every generated file is already wired with the
correct imports, interface stubs, Swagger annotations, and table-driven test
skeletons.

Run from the project root (the directory containing go.mod).`,
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

			opts := generator.FeatureOptions{
				Name:         args[0],
				FieldsSpec:   fieldsSpec,
				ModulePath:   modulePath,
				ProjectRoot:  cwd,
				NoAuth:       noAuth,
				NoPagination: noPagination,
			}
			files, err := generator.Generate(opts)
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
			if !DryRun() {
				cmd.Print(generator.NextSteps(opts))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&fieldsSpec, "fields", "f", "",
		`Comma-separated model fields, e.g. "name:string,price:float64,active:bool"`)
	cmd.Flags().BoolVar(&noAuth, "no-auth", false, "Omit JWT auth middleware from generated routes")
	cmd.Flags().BoolVar(&noPagination, "no-pagination", false, "Skip pagination on the List endpoint")
	return cmd
}
