// Package generator turns a parsed FeatureOptions into the eleven Go source
// files that constitute a new feature in a goplate-style project.
package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"github.com/haivh2111/goplate/internal/fields"
	"github.com/haivh2111/goplate/internal/naming"
	"github.com/haivh2111/goplate/internal/project"
	"github.com/haivh2111/goplate/internal/templates"
)

// FeatureOptions configures a single `goplate new-feature` run.
type FeatureOptions struct {
	Name         string // raw feature name, e.g. "product"
	FieldsSpec   string // raw --fields DSL string
	ModulePath   string // Go module path of the target project
	ProjectRoot  string // absolute path the writer treats as root
	NoAuth       bool
	NoPagination bool
}

// OverwriteMode controls how Writers behave when a target file already exists.
type OverwriteMode uint8

const (
	// OverwriteFail (default) makes FSWriter refuse the entire batch if any
	// non-replace file already exists on disk.
	OverwriteFail OverwriteMode = iota
	// OverwriteReplace unconditionally writes the file, replacing any existing
	// content. Used for AST-merged files like an appended port.go.
	OverwriteReplace
)

// File is one rendered file ready to be written.
type File struct {
	RelPath   string // project-relative, forward-slash, e.g. "internal/features/product/model.go"
	Content   []byte
	Overwrite OverwriteMode // default OverwriteFail
}

// featureData is the value passed to every template.
type featureData struct {
	FeatureName       string
	PackageName       string
	EntityName        string
	EntityPlural      string
	RoutePath         string
	VarName           string
	ModulePath        string
	FeatureImportPath string
	Fields            []fields.Field
	NoAuth            bool
	NoPagination      bool
	HasTimeField      bool
}

// Generate validates opts, parses fields, renders every feature template, and
// returns the resulting file list. It does NOT touch the filesystem.
func Generate(opts FeatureOptions) ([]File, error) {
	if err := validateName(opts.Name); err != nil {
		return nil, err
	}
	parsed, err := fields.Parse(opts.FieldsSpec)
	if err != nil {
		return nil, fmt.Errorf("--fields: %w", err)
	}
	if opts.ModulePath == "" {
		return nil, errors.New("module path is required (run from a Go project root)")
	}

	data := featureData{
		FeatureName:       opts.Name,
		PackageName:       opts.Name,
		EntityName:        naming.ToPascal(opts.Name),
		EntityPlural:      naming.ToPascal(naming.Pluralize(opts.Name)),
		RoutePath:         naming.Pluralize(opts.Name),
		VarName:           string(opts.Name[0]),
		ModulePath:        opts.ModulePath,
		FeatureImportPath: project.FeatureImportPath(opts.ModulePath, opts.Name),
		Fields:            parsed,
		NoAuth:            opts.NoAuth,
		NoPagination:      opts.NoPagination,
		HasTimeField:      fields.NeedsTimeImport(parsed),
	}

	templatePairs := []struct {
		tmpl string // file under templates/feature/
		out  string // output filename
	}{
		{"model.go.tmpl", "model.go"},
		{"dto.go.tmpl", "dto.go"},
		{"repository.go.tmpl", "repository.go"},
		{"repository_mysql.go.tmpl", "repository_mysql.go"},
		{"service.go.tmpl", "service.go"},
		{"service_impl.go.tmpl", "service_impl.go"},
		{"handler.go.tmpl", "handler.go"},
		{"module.go.tmpl", "module.go"},
		{"service_impl_test.go.tmpl", "service_impl_test.go"},
		{"handler_test.go.tmpl", "handler_test.go"},
		{"repository_mysql_test.go.tmpl", "repository_mysql_test.go"},
	}

	dir := project.FeatureRelDir(opts.Name)
	out := make([]File, 0, len(templatePairs))
	for _, tp := range templatePairs {
		body, err := render(tp.tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", tp.tmpl, err)
		}
		formatted, err := format.Source(body)
		if err != nil {
			return nil, fmt.Errorf("gofmt %s: %w\n--- rendered output ---\n%s", tp.tmpl, err, body)
		}
		out = append(out, File{
			RelPath: path.Join(dir, tp.out),
			Content: formatted,
		})
	}
	return out, nil
}

func render(name string, data featureData) ([]byte, error) {
	raw, err := fs.ReadFile(templates.Feature, "feature/"+name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Funcs(template.FuncMap{
		"title":  naming.ToPascal,
		"camel":  naming.ToCamel,
		"plural": naming.Pluralize,
		"lower":  strings.ToLower,
	}).Parse(string(raw))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func validateName(name string) error {
	return validateLowerName("feature", name)
}

// NextSteps renders the post-generation reminder block. Pulled out so the
// CLI layer can print it without re-deriving the entity name.
func NextSteps(opts FeatureOptions) string {
	entity := naming.ToPascal(opts.Name)
	var b strings.Builder
	b.WriteString("\n  Next steps:\n")
	fmt.Fprintf(&b, "    1. Add %s.Register(api, p) in internal/server/server.go\n", opts.Name)
	fmt.Fprintf(&b, "    2. Add &%s.%s{} to internal/infra/database/migrate.go\n", opts.Name, entity)
	b.WriteString("    3. Run: make swag && make test\n")
	return b.String()
}
