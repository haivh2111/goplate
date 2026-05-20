package cli

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/haivh2111/goplate/internal/generator"
	"github.com/haivh2111/goplate/internal/project"
	"github.com/haivh2111/goplate/internal/version"
)

func newNewCmd() *cobra.Command {
	var (
		module    string
		output    string
		db        string
		noGit     bool
		noEnv     bool
		skipTidy  bool
	)
	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Scaffold a new Go REST API project from the boilerplate template",
		Long: `Materializes a minimal Echo + GORM + hexagonal project from the embedded
template into <output>/<project-name>. After file write the command runs the
optional bootstrap steps:

  - copies .env.example → .env (skip with --no-env)
  - runs `+"`git init`"+` + initial commit (skip with --no-git)
  - runs `+"`go mod tidy`"+` (skip with --skip-tidy)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			outDir, err := filepath.Abs(filepath.Join(output, name))
			if err != nil {
				return err
			}

			// Refuse to scaffold into an existing non-empty dir.
			if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 {
				return fmt.Errorf("target directory %s already exists and is not empty", outDir)
			}

			modulePath := module
			if modulePath == "" {
				modulePath, err = defaultModulePath(name)
				if err != nil {
					return err
				}
			}

			opts := generator.ProjectOptions{
				Name:       name,
				ModulePath: modulePath,
				DBDriver:   db,
				OutputDir:  outDir,
			}
			files, err := generator.GenerateProject(opts)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "  ✓  Rendering template…")
			fmt.Fprintf(out, "  ✓  Replacing module path → %s\n", modulePath)

			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			w := generator.FSWriter{Root: outDir}
			// Pipe the per-file ✓ output to a discarded sink — we print a
			// single summary line instead, matching the spec format.
			w.Stdout = devNull{}
			if err := w.Write(files); err != nil {
				return err
			}
			fmt.Fprintf(out, "  ✓  Writing %d files\n", len(files))

			if !noEnv {
				if err := project.CopyEnv(outDir, ".env.example", ".env"); err != nil {
					return fmt.Errorf("copy .env: %w", err)
				}
				fmt.Fprintln(out, "  ✓  Copying .env.example → .env")
			}
			if !noGit {
				msg := fmt.Sprintf("initial commit from goplate %s", version.Version)
				if err := project.InitGit(outDir, msg); err != nil {
					fmt.Fprintf(out, "  ✗  git init failed: %v (continuing)\n", err)
				} else {
					fmt.Fprintln(out, "  ✓  git init + initial commit")
				}
			}
			if !skipTidy {
				if err := project.GoModTidy(outDir); err != nil {
					fmt.Fprintf(out, "  ✗  go mod tidy failed: %v (re-run with --skip-tidy to bypass)\n", err)
				} else {
					fmt.Fprintln(out, "  ✓  go mod tidy")
				}
			}

			fmt.Fprint(out, generator.ProjectNextSteps(opts))
			return nil
		},
	}
	cmd.Flags().StringVarP(&module, "module", "m", "",
		"Go module path for the new project (default: github.com/<user>/<name>)")
	cmd.Flags().StringVarP(&output, "output", "o", "./",
		"Directory where the project will be created")
	cmd.Flags().StringVar(&db, "db", "mysql",
		"Database driver: mysql | postgres | sqlite")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "Skip initialising a git repository")
	cmd.Flags().BoolVar(&noEnv, "no-env", false, "Skip copying .env.example → .env")
	cmd.Flags().BoolVar(&skipTidy, "skip-tidy", false, "Skip `go mod tidy` (use when offline)")
	return cmd
}

// defaultModulePath builds "github.com/<user>/<name>" from the current OS user.
func defaultModulePath(name string) (string, error) {
	u, err := user.Current()
	if err != nil || u.Username == "" {
		return "", fmt.Errorf("could not determine OS user — pass --module explicitly")
	}
	return "github.com/" + u.Username + "/" + name, nil
}

// devNull discards writes; used to silence the per-file ✓ output during `new`.
type devNull struct{}

func (devNull) Write(p []byte) (int, error) { return len(p), nil }
