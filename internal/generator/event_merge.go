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

// appendEventTypes inserts a new const + payload struct into an existing
// event_types.go. If <EventName> or <EventName>Payload is already declared at
// the top level, returns alreadyPresent=true and leaves the file unchanged.
//
// When the payload has a time.Time field, "time" is added to the import block.
func appendEventTypes(existing []byte, data eventData) ([]byte, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "event_types.go", existing, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("parse event_types.go: %w", err)
	}

	// Scan top-level decls for existing const <EventName> or type <EventName>Payload.
	payloadType := data.EventName + "Payload"
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gen.Specs {
			switch v := s.(type) {
			case *ast.ValueSpec:
				for _, n := range v.Names {
					if n.Name == data.EventName {
						return nil, true, nil
					}
				}
			case *ast.TypeSpec:
				if v.Name.Name == payloadType {
					return nil, true, nil
				}
			}
		}
	}

	// Add "time" import if needed.
	if data.HasTimeField {
		astutil.AddImport(fset, file, "time")
	}

	// Re-emit file with any new imports, then textually append the block.
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, false, fmt.Errorf("format event_types.go: %w", err)
	}

	block, err := renderEventFragment("event_types_block.go.tmpl", data)
	if err != nil {
		return nil, false, err
	}
	merged := append(buf.Bytes(), []byte(block)...)
	out, err := format.Source(merged)
	if err != nil {
		return merged, false, nil
	}
	return out, false, nil
}

// injectSubscribeCall splices a new p.EventBus.Subscribe(...) block into the
// body of an existing RegisterSubscribers function. If the body already
// contains "events.<EventName>", returns alreadyPresent=true unchanged.
//
// If RegisterSubscribers doesn't exist, the function is appended at the end of
// the file.
func injectSubscribeCall(existing []byte, data eventData) ([]byte, bool, error) {
	// Quick check: if the file already references events.<EventName>, treat as present.
	needle := "events." + data.EventName
	if bytes.Contains(existing, []byte(needle)) {
		return nil, true, nil
	}

	block, err := renderEventFragment("subscribe_call.go.tmpl", data)
	if err != nil {
		return nil, false, err
	}

	// If RegisterSubscribers exists, splice the block into its body.
	headerRE := regexp.MustCompile(`func\s+RegisterSubscribers\s*\([^)]*\)\s*\{`)
	if headerRE.Find(existing) != nil {
		injected, err := injectBeforeCloseBrace(existing, headerRE, block)
		if err != nil {
			return nil, false, fmt.Errorf("inject RegisterSubscribers body: %w", err)
		}
		out, err := format.Source(injected)
		if err != nil {
			return injected, false, nil
		}
		return out, false, nil
	}

	// Otherwise, append a brand-new RegisterSubscribers function.
	full, err := renderEventFragment("subscribers.go.tmpl", data)
	if err != nil {
		return nil, false, err
	}
	// We strip the package decl + imports from the rendered subscribers.go (the
	// existing file already has them) — keep only the func declaration.
	funcStart := strings.Index(full, "func RegisterSubscribers")
	if funcStart < 0 {
		return nil, false, fmt.Errorf("subscribers.go template missing RegisterSubscribers func")
	}
	var b bytes.Buffer
	b.Write(existing)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(full[funcStart:])
	out, err := format.Source(b.Bytes())
	if err != nil {
		return b.Bytes(), false, nil
	}
	return out, false, nil
}
