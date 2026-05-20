package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/haivh2111/goplate/internal/templates"
)

// ProjectOptions configures a single `goplate new` run.
type ProjectOptions struct {
	Name        string // raw project name, used in filenames + module path
	ModulePath  string // Go module path; required
	DBDriver    string // "mysql" | "postgres" | "sqlite"
	OutputDir   string // absolute path where the project tree will be written
	ProjectRoot string // alias of OutputDir for FSWriter
}

// projectData is the value handed to every project template.
type projectData struct {
	ProjectName string
	ModulePath  string
	DBDriver    string
}

var validDBDrivers = map[string]struct{}{
	"mysql":    {},
	"postgres": {},
	"sqlite":   {},
}

// GenerateProject walks the embedded project template tree, renders each file,
// and returns the list of files the writer should produce. It performs no
// filesystem writes itself.
func GenerateProject(opts ProjectOptions) ([]File, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if opts.ModulePath == "" {
		return nil, fmt.Errorf("module path is required (pass --module)")
	}
	if _, ok := validDBDrivers[opts.DBDriver]; !ok {
		return nil, fmt.Errorf("invalid --db %q (want mysql | postgres | sqlite)", opts.DBDriver)
	}

	data := projectData{
		ProjectName: opts.Name,
		ModulePath:  opts.ModulePath,
		DBDriver:    opts.DBDriver,
	}

	var out []File
	err := fs.WalkDir(templates.Project, "project", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		raw, err := fs.ReadFile(templates.Project, p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}

		// Compute relative path inside the new project.
		rel := strings.TrimPrefix(p, "project/")
		rel = strings.TrimSuffix(rel, ".tmpl")

		// Templated files (.tmpl extension on source) get text/template treatment.
		isTemplate := strings.HasSuffix(p, ".tmpl")
		content := raw
		if isTemplate {
			body, err := renderProjectTemplate(path.Base(p), raw, data)
			if err != nil {
				return fmt.Errorf("render %s: %w", p, err)
			}
			content = body
		}

		// .go files get gofmt'd.
		if strings.HasSuffix(rel, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return fmt.Errorf("gofmt %s: %w\n--- rendered ---\n%s", rel, err, content)
			}
			content = formatted
		}

		out = append(out, File{
			RelPath: filepath.ToSlash(rel),
			Content: content,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func renderProjectTemplate(name string, raw []byte, data projectData) ([]byte, error) {
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	return buf.Bytes(), nil
}

// ProjectNextSteps prints the post-creation reminder block. The CLI also
// prints driver-specific hints (e.g. `docker compose up -d db`).
func ProjectNextSteps(opts ProjectOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  Project ready at ./%s\n", opts.Name)
	b.WriteString("  Next: edit .env, then run  make dev\n")
	return b.String()
}
