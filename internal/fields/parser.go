package fields

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/haivh2111/goplate/internal/naming"
)

var identRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// Parse turns a DSL spec like "name:string,price:float64,active:bool" into a
// slice of Fields. It returns a precise error citing the offending token when
// the input is malformed.
//
// Whitespace around tokens is tolerated. Empty input is an error — callers
// (e.g. new-feature) require at least one field.
func Parse(spec string) ([]Field, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, fmt.Errorf("empty field spec; example: --fields \"name:string,price:float64,active:bool\"")
	}

	tokens := strings.Split(trimmed, ",")
	out := make([]Field, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))

	for _, raw := range tokens {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			return nil, fmt.Errorf("empty field token in spec %q", spec)
		}
		parts := strings.SplitN(tok, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("field %q: expected name:type", tok)
		}
		name := strings.TrimSpace(parts[0])
		typeName := strings.TrimSpace(parts[1])

		if name == "" {
			return nil, fmt.Errorf("field %q: missing name", tok)
		}
		if !identRE.MatchString(name) {
			return nil, fmt.Errorf("field %q: invalid name (must match [a-zA-Z][a-zA-Z0-9_]*)", tok)
		}
		spec, ok := typeTable[typeName]
		if !ok {
			return nil, fmt.Errorf("field %q: unknown type %q, supported: %s",
				tok, typeName, strings.Join(SupportedTypes(), ", "))
		}

		pascal := naming.ToPascal(name)
		if _, dup := seen[pascal]; dup {
			return nil, fmt.Errorf("field %q: duplicate name %q", tok, pascal)
		}
		seen[pascal] = struct{}{}

		out = append(out, Field{
			Name:         pascal,
			JSONName:     naming.ToCamel(name),
			GoType:       spec.goType,
			ValidatorTag: spec.validatorTag,
			GORMTag:      spec.gormTag,
			SwaggerType:  spec.swaggerType,
			NeedsTime:    spec.needsTime,
		})
	}
	return out, nil
}

// NeedsTimeImport reports whether any field in the slice uses time.Time.
func NeedsTimeImport(fs []Field) bool {
	for _, f := range fs {
		if f.NeedsTime {
			return true
		}
	}
	return false
}
