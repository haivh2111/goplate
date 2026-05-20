package fields

import (
	"strings"
	"testing"
)

func TestParse_Valid(t *testing.T) {
	got, err := Parse("name:string,price:float64,stock:int,active:bool,created_at:time.Time")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(got))
	}

	want := []struct {
		Name, GoType, JSONName, GORMTag, ValidatorTag string
		NeedsTime                                     bool
	}{
		{"Name", "string", "name", `gorm:"not null;size:255"`, `validate:"required,min=2,max=255"`, false},
		{"Price", "float64", "price", `gorm:"not null"`, `validate:"required,gt=0"`, false},
		{"Stock", "int", "stock", `gorm:"default:0"`, `validate:"min=0"`, false},
		{"Active", "bool", "active", `gorm:"default:true"`, "", false},
		{"CreatedAt", "time.Time", "createdAt", `gorm:"not null"`, `validate:"required"`, true},
	}
	for i, w := range want {
		g := got[i]
		if g.Name != w.Name || g.GoType != w.GoType || g.JSONName != w.JSONName ||
			g.GORMTag != w.GORMTag || g.ValidatorTag != w.ValidatorTag || g.NeedsTime != w.NeedsTime {
			t.Errorf("field %d:\n got  %+v\n want %+v", i, g, w)
		}
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		spec, contains string
	}{
		{"", "empty field spec"},
		{"   ", "empty field spec"},
		{"name", "expected name:type"},
		{"name:", "unknown type"},
		{":string", "missing name"},
		{"name:flot64", "unknown type"},
		{"1name:string", "invalid name"},
		{"name:string,name:int", "duplicate name"},
		{"name:string,,price:float64", "empty field token"},
	}
	for _, c := range cases {
		_, err := Parse(c.spec)
		if err == nil {
			t.Errorf("Parse(%q): expected error, got nil", c.spec)
			continue
		}
		if !strings.Contains(err.Error(), c.contains) {
			t.Errorf("Parse(%q): error %q does not contain %q", c.spec, err.Error(), c.contains)
		}
	}
}

func TestNeedsTimeImport(t *testing.T) {
	none, _ := Parse("name:string,price:float64")
	if NeedsTimeImport(none) {
		t.Error("expected false for non-time fields")
	}
	some, _ := Parse("name:string,createdAt:time.Time")
	if !NeedsTimeImport(some) {
		t.Error("expected true when a time.Time field is present")
	}
}
