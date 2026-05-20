// Package fields parses the `--fields` / `--payload` DSL used by the goplate
// generators and produces strongly-typed Field structs that carry every tag
// the templates need (GORM, validator, JSON, Swagger).
package fields

// Field is the resolved description of a single model/payload field after
// parsing a DSL token like "price:float64".
type Field struct {
	// Name is the Go identifier in PascalCase ("Price", "OrderID").
	Name string
	// JSONName is the camelCase JSON tag value ("price", "orderID").
	JSONName string
	// GoType is the Go type literal ("string", "float64", "time.Time").
	GoType string
	// ValidatorTag is the validator struct-tag fragment, e.g.
	// `validate:"required,gt=0"`. Empty string means "no validator tag".
	ValidatorTag string
	// GORMTag is the GORM struct-tag fragment, e.g. `gorm:"not null"`.
	GORMTag string
	// SwaggerType is the Swagger primitive name used in handler annotations.
	SwaggerType string
	// NeedsTime is true when GoType is time.Time — drives "time" import injection.
	NeedsTime bool
}

// typeSpec is the per-DSL-type template that Parse uses to populate Field tags.
type typeSpec struct {
	goType       string
	gormTag      string
	validatorTag string
	swaggerType  string
	needsTime    bool
}

// typeTable is the single source of truth for the supported field types and
// their generated tags. Adding a new type means adding one row here.
var typeTable = map[string]typeSpec{
	"string": {
		goType:       "string",
		gormTag:      `gorm:"not null;size:255"`,
		validatorTag: `validate:"required,min=2,max=255"`,
		swaggerType:  "string",
	},
	"int": {
		goType:       "int",
		gormTag:      `gorm:"default:0"`,
		validatorTag: `validate:"min=0"`,
		swaggerType:  "integer",
	},
	"int64": {
		goType:       "int64",
		gormTag:      `gorm:"default:0"`,
		validatorTag: `validate:"min=0"`,
		swaggerType:  "integer",
	},
	"uint": {
		goType:       "uint",
		gormTag:      `gorm:"default:0"`,
		validatorTag: `validate:"min=0"`,
		swaggerType:  "integer",
	},
	"float64": {
		goType:       "float64",
		gormTag:      `gorm:"not null"`,
		validatorTag: `validate:"required,gt=0"`,
		swaggerType:  "number",
	},
	"bool": {
		goType:       "bool",
		gormTag:      `gorm:"default:true"`,
		validatorTag: "",
		swaggerType:  "boolean",
	},
	"time.Time": {
		goType:       "time.Time",
		gormTag:      `gorm:"not null"`,
		validatorTag: `validate:"required"`,
		swaggerType:  "string",
		needsTime:    true,
	},
}

// SupportedTypes returns the DSL type keys in a stable order for error messages.
func SupportedTypes() []string {
	return []string{"string", "int", "int64", "uint", "float64", "bool", "time.Time"}
}
