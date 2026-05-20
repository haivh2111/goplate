package generator

import (
	"fmt"
	"regexp"
)

var (
	// lowerNameRE matches the package-name shape used for both feature names
	// and adapter service/provider names.
	lowerNameRE = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

	// goKeywords is the set of reserved words that may never be used as a
	// package name.
	goKeywords = map[string]struct{}{
		"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
		"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
		"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
		"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
		"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
	}
)

// validateLowerName verifies that name is a valid lowercase Go package name
// (no underscores, no leading digit, not a keyword). Used for feature names,
// adapter service names, and adapter provider names.
func validateLowerName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is required", kind)
	}
	if !lowerNameRE.MatchString(name) {
		return fmt.Errorf("%s name %q is invalid (must match ^[a-z][a-z0-9]*$)", kind, name)
	}
	if _, isKw := goKeywords[name]; isKw {
		return fmt.Errorf("%s name %q is a Go keyword", kind, name)
	}
	return nil
}
