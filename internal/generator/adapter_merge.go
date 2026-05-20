package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// mergePort updates an existing port.go to include any new methods from the
// adapter run. Strategy:
//
//  1. Parse the existing source.
//  2. Add any newly-required imports via astutil.AddImport (AST-level).
//  3. Append placeholder struct stubs for newly-referenced local types.
//  4. Re-print, then textually inject the new method-declaration lines just
//     before the closing brace of the <Service>Gateway interface body. We use
//     a textual splice (rather than AST mutation) because appended *ast.Field
//     nodes carry positions from the DSL parse, which can confuse go/printer
//     into emitting unusual line breaks inside the new method signatures.
func mergePort(existing []byte, data adapterData) ([]byte, []string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "port.go", existing, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse existing port.go: %w", err)
	}

	ifaceName := data.ServicePascal + "Gateway"
	iface, err := findGatewayInterface(file, ifaceName)
	if err != nil {
		return nil, nil, err
	}

	// Index existing method names.
	existingMethods := map[string]struct{}{}
	for _, f := range iface.Methods.List {
		for _, n := range f.Names {
			existingMethods[n.Name] = struct{}{}
		}
	}
	// Index existing top-level type names so we don't re-stub.
	existingTypes := map[string]struct{}{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, s := range gen.Specs {
			if ts, ok := s.(*ast.TypeSpec); ok {
				existingTypes[ts.Name.Name] = struct{}{}
			}
		}
	}

	// Determine which DSL methods are new and what they bring with them.
	var (
		newDecls []string
		added    []string
	)
	addedPkgs := map[string]struct{}{}
	addedLocals := map[string]struct{}{}
	for _, m := range data.parsed {
		if _, ok := existingMethods[m.Name]; ok {
			continue
		}
		newDecls = append(newDecls, m.Decl)
		added = append(added, m.Name)
		for _, p := range m.UsesPkgs {
			addedPkgs[p] = struct{}{}
		}
		for _, l := range m.LocalTypes {
			addedLocals[l] = struct{}{}
		}
	}

	// Add missing stdlib imports via AST so import block layout is correct.
	for p := range addedPkgs {
		astutil.AddImport(fset, file, p)
	}
	// Append placeholder structs for newly-referenced local types.
	for l := range addedLocals {
		if _, ok := existingTypes[l]; ok {
			continue
		}
		file.Decls = append(file.Decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: ast.NewIdent(l),
					Type: &ast.StructType{Fields: &ast.FieldList{}},
				},
			},
		})
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, nil, fmt.Errorf("format merged port.go: %w", err)
	}

	merged := buf.Bytes()
	if len(newDecls) > 0 {
		merged, err = injectInterfaceMethods(merged, ifaceName, newDecls)
		if err != nil {
			return nil, nil, fmt.Errorf("inject methods: %w", err)
		}
	}

	out, err := format.Source(merged)
	if err != nil {
		return merged, added, nil
	}
	return out, added, nil
}

// injectInterfaceMethods finds `type <name> interface { ... }` in src and
// inserts each line of decls (tab-indented) just before the closing brace.
func injectInterfaceMethods(src []byte, name string, decls []string) ([]byte, error) {
	headerRE := regexp.MustCompile(`type\s+` + regexp.QuoteMeta(name) + `\s+interface\s*\{`)
	var insertion strings.Builder
	for _, d := range decls {
		insertion.WriteString("\t")
		insertion.WriteString(d)
		insertion.WriteString("\n")
	}
	out, err := injectBeforeCloseBrace(src, headerRE, insertion.String())
	if err != nil {
		return nil, fmt.Errorf("interface %q: %w", name, err)
	}
	return out, nil
}

func findGatewayInterface(file *ast.File, name string) (*ast.InterfaceType, error) {
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
			if ts.Name.Name != name {
				continue
			}
			iface, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				return nil, fmt.Errorf("existing port.go declares %s but it isn't an interface — refusing to merge", name)
			}
			return iface, nil
		}
	}
	return nil, fmt.Errorf("existing port.go has no %s interface — refusing to merge", name)
}
