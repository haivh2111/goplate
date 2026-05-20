package generator

import (
	"bytes"
	"fmt"
	"regexp"
)

// injectBeforeCloseBrace splices content into src just before the closing
// brace of the named block whose header matches headerRE.
//
// "Block" = anything with a `{` opener and matching `}` closer: an interface
// body, struct body, or function body. Brace counting respects depth so
// nested braces inside the block are not mistaken for the closer.
//
// headerRE must match the substring ending at the opening `{` (inclusive) —
// for example `regexp.MustCompile(`type\s+Foo\s+interface\s*\{`)`.
func injectBeforeCloseBrace(src []byte, headerRE *regexp.Regexp, content string) ([]byte, error) {
	loc := headerRE.FindIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("injection target not found")
	}
	depth := 1
	pos := loc[1] // just after the opening brace
	for ; pos < len(src) && depth > 0; pos++ {
		switch src[pos] {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced braces at injection target")
	}
	closeIdx := pos - 1

	var out bytes.Buffer
	out.Write(src[:closeIdx])
	out.WriteString(content)
	out.Write(src[closeIdx:])
	return out.Bytes(), nil
}
