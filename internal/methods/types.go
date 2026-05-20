// Package methods parses the `--methods` DSL used by `goplate new-adapter`.
// Two input forms are accepted:
//
//  1. Name-only, comma-separated: "CreateCharge,RefundCharge,GetCharge"
//     → each becomes `<Name>(ctx context.Context) error`.
//  2. Full Go interface-method syntax, ';' or newline separated:
//     "CreateCharge(req ChargeRequest) (*ChargeResponse, error); RefundCharge(id string) error"
//
// Form 2 is auto-detected by the presence of '(' in the input.
package methods

import "go/ast"

// Method is a resolved interface method ready to be rendered into a template.
type Method struct {
	// Name is the exported PascalCase method identifier.
	Name string
	// Decl is the formatted method declaration excluding the leading "func "
	// or interface-receiver — i.e. "Foo(req Bar) (Baz, error)". This is what
	// follows `func (a *T)` in adapter.go and is also a valid interface body line.
	Decl string
	// UsesPkgs lists short import-path keys referenced via selector exprs in
	// the signature (e.g. "context" for context.Context, "io" for io.Reader).
	// The methods package does NOT resolve these to full module paths — the
	// generator uses a known-stdlib lookup table.
	UsesPkgs []string
	// LocalTypes lists unqualified PascalCase type names referenced in the
	// signature. They are candidates for placeholder struct stubs in port.go.
	LocalTypes []string

	// raw AST nodes preserved for the merge path so we can append directly
	// into an existing interface declaration.
	Field *ast.Field
}
