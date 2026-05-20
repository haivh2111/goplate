package methods

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"strings"
)

// Parse turns a --methods DSL string into a list of Methods.
func Parse(spec string) ([]Method, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, fmt.Errorf("--methods is empty; example: \"Create,Refund\" or \"Create(req Req) (*Resp, error)\"")
	}
	if !strings.ContainsRune(trimmed, '(') {
		return parseNamesOnly(trimmed)
	}
	return parseGoSignatures(trimmed)
}

var pascalRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

func parseNamesOnly(spec string) ([]Method, error) {
	tokens := strings.Split(spec, ",")
	out := make([]Method, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, raw := range tokens {
		name := strings.TrimSpace(raw)
		if name == "" {
			return nil, fmt.Errorf("empty method name in spec %q", spec)
		}
		if !pascalRE.MatchString(name) {
			return nil, fmt.Errorf("method %q is not a valid exported Go identifier", name)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate method name %q", name)
		}
		seen[name] = struct{}{}
		out = append(out, Method{
			Name:     name,
			Decl:     fmt.Sprintf("%s(ctx context.Context) error", name),
			UsesPkgs: []string{"context"},
			Field:    nameOnlyField(name),
		})
	}
	return out, nil
}

// nameOnlyField builds the *ast.Field for a synthesized "Name(ctx context.Context) error".
func nameOnlyField(name string) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(name)},
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{
				{
					Names: []*ast.Ident{ast.NewIdent("ctx")},
					Type: &ast.SelectorExpr{
						X:   ast.NewIdent("context"),
						Sel: ast.NewIdent("Context"),
					},
				},
			}},
			Results: &ast.FieldList{List: []*ast.Field{
				{Type: ast.NewIdent("error")},
			}},
		},
	}
}

// parseGoSignatures wraps the DSL into a fake interface, parses it with
// go/parser, and walks the resulting interface methods.
func parseGoSignatures(spec string) ([]Method, error) {
	// Top-level ';' separates methods. Inside parens, ';' is illegal in our
	// DSL so a plain replace is safe.
	body := strings.ReplaceAll(spec, ";", "\n")
	src := "package x\ntype _ interface {\n" + body + "\n}\n"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "methods.go", src, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("could not parse method signatures: %w", err)
	}
	iface, err := findInterface(file)
	if err != nil {
		return nil, err
	}

	out := make([]Method, 0, len(iface.Methods.List))
	seen := make(map[string]struct{})
	for _, f := range iface.Methods.List {
		if len(f.Names) == 0 {
			return nil, fmt.Errorf("embedded interfaces are not allowed in --methods")
		}
		if len(f.Names) > 1 {
			return nil, fmt.Errorf("method %q: declare one method per signature", f.Names[0].Name)
		}
		name := f.Names[0].Name
		if !pascalRE.MatchString(name) {
			return nil, fmt.Errorf("method %q must be an exported Go identifier", name)
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("duplicate method name %q", name)
		}
		seen[name] = struct{}{}

		fn, ok := f.Type.(*ast.FuncType)
		if !ok {
			return nil, fmt.Errorf("method %q: invalid signature", name)
		}
		decl, err := formatDecl(name, fn, fset)
		if err != nil {
			return nil, fmt.Errorf("method %q: %w", name, err)
		}
		pkgs, locals := collectReferences(fn)
		out = append(out, Method{
			Name:       name,
			Decl:       decl,
			UsesPkgs:   pkgs,
			LocalTypes: locals,
			Field:      f,
		})
	}
	return out, nil
}

func findInterface(file *ast.File) (*ast.InterfaceType, error) {
	for _, d := range file.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, s := range gen.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if iface, ok := ts.Type.(*ast.InterfaceType); ok {
				return iface, nil
			}
		}
	}
	return nil, fmt.Errorf("internal error: synthesized interface not found")
}

// formatDecl prints "Name(params) results" by serializing the FuncType and
// stripping the leading "func" keyword.
func formatDecl(name string, fn *ast.FuncType, fset *token.FileSet) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, fn); err != nil {
		return "", err
	}
	body := strings.TrimPrefix(buf.String(), "func")
	return name + body, nil
}

// predeclared types we should not generate stubs for.
var predeclared = map[string]struct{}{
	"any": {}, "bool": {}, "byte": {}, "rune": {}, "string": {}, "error": {},
	"int": {}, "int8": {}, "int16": {}, "int32": {}, "int64": {},
	"uint": {}, "uint8": {}, "uint16": {}, "uint32": {}, "uint64": {}, "uintptr": {},
	"float32": {}, "float64": {}, "complex64": {}, "complex128": {},
	"comparable": {},
}

// collectReferences walks the signature and returns:
//   - the deduped list of package names referenced via SelectorExpr
//   - the deduped list of unqualified PascalCase types that look user-defined
func collectReferences(fn *ast.FuncType) (pkgs []string, locals []string) {
	pkgSet := map[string]struct{}{}
	localSet := map[string]struct{}{}

	visit := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, f := range list.List {
			ast.Inspect(f.Type, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					if id, ok := v.X.(*ast.Ident); ok {
						pkgSet[id.Name] = struct{}{}
					}
					return false // do not recurse into selector
				case *ast.Ident:
					if _, pre := predeclared[v.Name]; pre {
						return false
					}
					if len(v.Name) > 0 && v.Name[0] >= 'A' && v.Name[0] <= 'Z' {
						localSet[v.Name] = struct{}{}
					}
				}
				return true
			})
		}
	}
	visit(fn.Params)
	visit(fn.Results)

	for k := range pkgSet {
		pkgs = append(pkgs, k)
	}
	for k := range localSet {
		locals = append(locals, k)
	}
	return pkgs, locals
}
