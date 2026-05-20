package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/ast/astutil"

	"github.com/haivh2111/goplate/internal/methods"
	"github.com/haivh2111/goplate/internal/naming"
	"github.com/haivh2111/goplate/internal/templates"
)

// AdapterOptions configures a single `goplate new-adapter` run.
type AdapterOptions struct {
	Service     string // raw service name, e.g. "payment"
	Provider    string // raw provider name, e.g. "stripe"
	MethodsSpec string // raw --methods DSL string
	ModulePath  string // Go module path of the target project
	ProjectRoot string // absolute path; used to detect an existing port.go

	// StubSiblings, when true, automatically appends panic-stub receiver
	// methods for any newly-added methods to every existing sibling
	// provider's adapter.go. Lets the project keep compiling when the user
	// extends an interface via a subsequent new-adapter run.
	StubSiblings bool
}

// renderMethod is the per-method shape passed to templates. Decl is the bare
// signature usable inside the port's own package; DeclQual prefixes local
// types with "<service>." for use inside the provider sub-package.
type renderMethod struct {
	Name     string
	Decl     string
	DeclQual string
	UsesPkgs []string // stdlib package names referenced in the signature
}

// adapterData is the value handed to the adapter templates.
type adapterData struct {
	Service         string // "payment"
	ServicePascal   string // "Payment"
	Provider        string // "stripe"
	ModulePath      string // "github.com/acme/svc"
	Methods         []renderMethod
	RequiredImports []string // deduped stdlib import paths
	LocalTypeStubs  []string // deduped placeholder type names

	parsed []methods.Method // kept for the AST-merge path
}

// GenerateAdapter validates opts, parses methods, and returns the file list
// for an adapter run plus a human-readable notice (or empty string) the CLI
// should print after the writer runs.
//
// When <svc>/port.go already exists, that file is AST-merged
// (Overwrite=Replace) only if the merge actually introduces new methods —
// otherwise it is omitted from the file list, leaving the existing port.go
// untouched (the typical "add a second provider with the same interface" flow).
func GenerateAdapter(opts AdapterOptions) ([]File, string, error) {
	if err := validateLowerName("service", opts.Service); err != nil {
		return nil, "", err
	}
	if err := validateLowerName("provider", opts.Provider); err != nil {
		return nil, "", err
	}
	parsed, err := methods.Parse(opts.MethodsSpec)
	if err != nil {
		return nil, "", fmt.Errorf("--methods: %w", err)
	}
	if opts.ModulePath == "" {
		return nil, "", fmt.Errorf("module path is required (run from a Go project root)")
	}

	pkgs := dedupeSorted(allPkgs(parsed))
	locals := dedupeSorted(allLocals(parsed))

	rendered := make([]renderMethod, len(parsed))
	for i, m := range parsed {
		rendered[i] = renderMethod{
			Name:     m.Name,
			Decl:     m.Decl,
			DeclQual: qualifyDecl(m.Decl, m.LocalTypes, opts.Service),
			UsesPkgs: append([]string(nil), m.UsesPkgs...),
		}
	}

	data := adapterData{
		Service:         opts.Service,
		ServicePascal:   naming.ToPascal(opts.Service),
		Provider:        opts.Provider,
		ModulePath:      opts.ModulePath,
		Methods:         rendered,
		RequiredImports: pkgs,
		LocalTypeStubs:  locals,
		parsed:          parsed,
	}

	svcDir := path.Join("internal", "adapters", opts.Service)
	provDir := path.Join(svcDir, opts.Provider)

	portRel := path.Join(svcDir, "port.go")
	adapterRel := path.Join(provDir, "adapter.go")
	adapterTestRel := path.Join(provDir, "adapter_test.go")

	var (
		out    []File
		notice string
	)

	// port.go — create fresh or attempt to AST-append into the existing file.
	portAbs := filepath.Join(opts.ProjectRoot, filepath.FromSlash(portRel))
	if existing, ok := readExisting(portAbs); ok {
		merged, addedNames, err := mergePort(existing, data)
		if err != nil {
			return nil, "", fmt.Errorf("merge port.go: %w", err)
		}
		if !bytes.Equal(merged, existing) {
			out = append(out, File{
				RelPath:   portRel,
				Content:   merged,
				Overwrite: OverwriteReplace,
			})
			// New methods extended the interface — handle existing siblings.
			siblings := siblingProviders(opts.ProjectRoot, opts.Service, opts.Provider)
			if len(siblings) > 0 {
				if opts.StubSiblings {
					added := filterByName(rendered, addedNames)
					stubs, err := buildSiblingStubs(opts.ProjectRoot, svcDir, siblings, added)
					if err != nil {
						return nil, "", fmt.Errorf("stub siblings: %w", err)
					}
					out = append(out, stubs...)
				} else {
					notice = fmt.Sprintf(
						"\n  Note: %s/port.go gained new methods — existing providers must implement them:\n    %s\n  Re-run with --stub-siblings to auto-append panic stubs.\n",
						svcDir,
						strings.Join(siblings, ", "),
					)
				}
			}
		}
		// If merged == existing (typical multi-provider case, same interface),
		// silently leave port.go alone.
	} else {
		body, err := renderAdapter("port.go.tmpl", data)
		if err != nil {
			return nil, "", err
		}
		out = append(out, File{RelPath: portRel, Content: body})
	}

	// adapter.go and adapter_test.go are always fresh — overwrite=Fail.
	body, err := renderAdapter("adapter.go.tmpl", data)
	if err != nil {
		return nil, "", err
	}
	out = append(out, File{RelPath: adapterRel, Content: body})

	body, err = renderAdapter("adapter_test.go.tmpl", data)
	if err != nil {
		return nil, "", err
	}
	out = append(out, File{RelPath: adapterTestRel, Content: body})

	return out, notice, nil
}

// filterByName returns the subset of rendered methods whose Name appears in keep.
func filterByName(all []renderMethod, keep []string) []renderMethod {
	set := make(map[string]struct{}, len(keep))
	for _, k := range keep {
		set[k] = struct{}{}
	}
	out := make([]renderMethod, 0, len(keep))
	for _, m := range all {
		if _, ok := set[m.Name]; ok {
			out = append(out, m)
		}
	}
	return out
}

// buildSiblingStubs reads each sibling provider's adapter.go and returns a
// File entry that appends panic-stub receiver methods for every method in
// added. Each sibling file is marked Overwrite=Replace.
func buildSiblingStubs(projectRoot, svcDir string, siblings []string, added []renderMethod) ([]File, error) {
	if len(added) == 0 {
		return nil, nil
	}
	out := make([]File, 0, len(siblings))
	for _, sib := range siblings {
		relPath := path.Join(svcDir, sib, "adapter.go")
		absPath := filepath.Join(projectRoot, filepath.FromSlash(relPath))
		existing, err := os.ReadFile(absPath)
		if err != nil {
			// Skip siblings without an adapter.go (unusual but not fatal).
			continue
		}
		appended, err := appendStubMethods(existing, sib, added)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relPath, err)
		}
		out = append(out, File{
			RelPath:   relPath,
			Content:   appended,
			Overwrite: OverwriteReplace,
		})
	}
	return out, nil
}

// appendStubMethods textually appends panic-stub receiver methods to the end
// of an existing adapter.go and ensures any stdlib packages those methods
// reference are imported.
func appendStubMethods(existing []byte, provider string, methods []renderMethod) ([]byte, error) {
	// 1. Add missing imports via AST.
	requiredPkgs := map[string]struct{}{}
	for _, m := range methods {
		for _, p := range m.UsesPkgs {
			requiredPkgs[p] = struct{}{}
		}
	}
	withImports, err := ensureImports(existing, requiredPkgs)
	if err != nil {
		return nil, err
	}

	// 2. Textually append the stub functions.
	var b strings.Builder
	b.Write(withImports)
	if len(withImports) > 0 && withImports[len(withImports)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteString("\n// Stubs auto-appended by goplate --stub-siblings. Implement these.\n")
	for _, m := range methods {
		fmt.Fprintf(&b, "func (a *%sAdapter) %s {\n", provider, m.DeclQual)
		fmt.Fprintf(&b, "\tpanic(\"not implemented: %s.%s\")\n", provider, m.Name)
		b.WriteString("}\n\n")
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("gofmt sibling stubs: %w\n--- pre-format ---\n%s", err, b.String())
	}
	return formatted, nil
}

// ensureImports adds any pkgs not already imported by src.
func ensureImports(src []byte, pkgs map[string]struct{}) ([]byte, error) {
	if len(pkgs) == 0 {
		return src, nil
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "adapter.go", src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse adapter.go: %w", err)
	}
	changed := false
	for p := range pkgs {
		if astutil.AddImport(fset, file, p) {
			changed = true
		}
	}
	if !changed {
		return src, nil
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("format adapter.go after import-add: %w", err)
	}
	return buf.Bytes(), nil
}

// siblingProviders lists subdirectories of internal/adapters/<svc>/ other
// than the one we're creating now. Empty on first run.
func siblingProviders(projectRoot, svc, exclude string) []string {
	svcAbs := filepath.Join(projectRoot, "internal", "adapters", svc)
	entries, err := os.ReadDir(svcAbs)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() || e.Name() == exclude {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// renderAdapter loads, parses, executes, and gofmt's one adapter template.
func renderAdapter(name string, data adapterData) ([]byte, error) {
	raw, err := fs.ReadFile(templates.Adapter, "adapter/"+name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute %s: %w", name, err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt %s: %w\n--- rendered ---\n%s", name, err, buf.String())
	}
	return formatted, nil
}

func readExisting(absPath string) ([]byte, bool) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, false
	}
	return data, true
}

// AdapterNextSteps prints the post-generation reminder block.
func AdapterNextSteps(opts AdapterOptions) string {
	svcP := naming.ToPascal(opts.Service)
	var b bytes.Buffer
	b.WriteString("\n  Next steps:\n")
	fmt.Fprintf(&b, "    1. Add  %s %s.%sGateway  to Providers in internal/server/providers.go\n",
		svcP, opts.Service, svcP)
	fmt.Fprintf(&b, "    2. Wire the concrete adapter in cmd/main.go:\n")
	fmt.Fprintf(&b, "         p.%s = %s.NewAdapter()\n", svcP, opts.Provider)
	fmt.Fprintf(&b, "    3. Inject p.%s into your feature via module.go\n", svcP)
	return b.String()
}

// qualifyDecl rewrites a method declaration so that any unqualified local
// PascalCase type reference becomes "<pkg>.<Type>". This is what we need when
// rendering a method declaration that lives in a different package from the
// types — e.g. stripeAdapter.CreateCharge using payment.ChargeRequest.
//
// Word boundaries are used so types that share a prefix with surrounding
// identifiers (e.g. "Charge" inside "CreateCharge") are not matched.
func qualifyDecl(decl string, locals []string, pkg string) string {
	out := decl
	for _, l := range locals {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(l) + `\b`)
		out = re.ReplaceAllString(out, pkg+"."+l)
	}
	return out
}

func allPkgs(ms []methods.Method) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.UsesPkgs...)
	}
	return out
}

func allLocals(ms []methods.Method) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.LocalTypes...)
	}
	return out
}

func dedupeSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
