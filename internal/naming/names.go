// Package naming provides identifier-case conversion and English pluralization
// helpers used by code generators. Pluralization is regular-English only —
// irregular plurals (child→children) are not supported.
package naming

import (
	"strings"
	"unicode"
)

// ToPascal converts a lowercase or snake-case identifier to PascalCase.
//
//	"product"        → "Product"
//	"order_item"     → "OrderItem"
//	"user_id"        → "UserID" (id is an acronym)
func ToPascal(s string) string {
	if s == "" {
		return ""
	}
	parts := splitWords(s)
	var b strings.Builder
	for _, p := range parts {
		if up, ok := commonAcronyms[strings.ToLower(p)]; ok {
			b.WriteString(up)
			continue
		}
		// If the segment already mixes cases (camelCase input), preserve
		// internal capitals — only force the first rune up.
		if hasInternalUpper(p) {
			b.WriteRune(unicode.ToUpper(rune(p[0])))
			b.WriteString(p[1:])
			continue
		}
		b.WriteRune(unicode.ToUpper(rune(p[0])))
		b.WriteString(strings.ToLower(p[1:]))
	}
	return b.String()
}

func hasInternalUpper(s string) bool {
	for i, r := range s {
		if i == 0 {
			continue
		}
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// ToCamel converts to camelCase (first word lowercased, rest as Pascal).
//
//	"product"   → "product"
//	"order_id"  → "orderID"
func ToCamel(s string) string {
	p := ToPascal(s)
	if p == "" {
		return ""
	}
	// If first segment is an acronym (e.g. "URL"), lowercase the whole acronym.
	parts := splitWords(s)
	first := strings.ToLower(parts[0])
	if _, ok := commonAcronyms[first]; ok {
		return first + p[len(commonAcronyms[first]):]
	}
	return strings.ToLower(p[:1]) + p[1:]
}

// ToDotted converts a PascalCase identifier to a dotted lowercase form
// suitable for event-bus topic strings.
//
//	"OrderPlaced"              → "order.placed"
//	"PaymentFailed"            → "payment.failed"
//	"OrderShippedToWarehouse"  → "order.shipped.to.warehouse"
//	"APIKeyRotated"            → "api.key.rotated"
func ToDotted(s string) string {
	if s == "" {
		return ""
	}
	var (
		segments []string
		current  strings.Builder
	)
	flush := func() {
		if current.Len() > 0 {
			segments = append(segments, strings.ToLower(current.String()))
			current.Reset()
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Acronym run: APIKey → "API", "Key". Detect by looking ahead at the
			// next rune — if it's lowercase, this uppercase starts a new word.
			if i > 0 {
				prev := runes[i-1]
				next := rune(0)
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				// Boundary cases:
				//   prev lower, this upper → new word ("orderPlaced" → order|Placed)
				//   prev upper, this upper, next lower → new word (acronym → word)
				if unicode.IsLower(prev) || (unicode.IsUpper(prev) && unicode.IsLower(next)) {
					flush()
				}
			}
		}
		current.WriteRune(r)
	}
	flush()
	return strings.Join(segments, ".")
}

// Pluralize returns the regular English plural of a lowercase noun.
//
//	"product"  → "products"
//	"category" → "categories"
//	"bus"      → "buses"
//	"box"      → "boxes"
//	"buzz"     → "buzzes"
func Pluralize(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "y") && len(lower) > 1 && !isVowel(rune(lower[len(lower)-2])):
		return s[:len(s)-1] + "ies"
	case strings.HasSuffix(lower, "s"),
		strings.HasSuffix(lower, "x"),
		strings.HasSuffix(lower, "z"),
		strings.HasSuffix(lower, "ch"),
		strings.HasSuffix(lower, "sh"):
		return s + "es"
	default:
		return s + "s"
	}
}

func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

func splitWords(s string) []string {
	if s == "" {
		return nil
	}
	// Split on underscores and dashes.
	raw := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(raw) == 0 {
		return []string{s}
	}
	return raw
}

// commonAcronyms maps a lowercase token to its canonical uppercased form.
// Following the Go style guide, identifiers like "URL", "ID", "API" are
// preserved as all-caps in PascalCase output.
var commonAcronyms = map[string]string{
	"id":    "ID",
	"url":   "URL",
	"uri":   "URI",
	"api":   "API",
	"http":  "HTTP",
	"https": "HTTPS",
	"json":  "JSON",
	"xml":   "XML",
	"sql":   "SQL",
	"db":    "DB",
	"uuid":  "UUID",
	"ip":    "IP",
}
