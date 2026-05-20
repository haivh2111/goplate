package methods

import (
	"sort"
	"strings"
	"testing"
)

func TestParse_NamesOnly(t *testing.T) {
	ms, err := Parse("Send,SendBulk,Receive")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ms) != 3 {
		t.Fatalf("got %d methods, want 3", len(ms))
	}
	want := []string{
		"Send(ctx context.Context) error",
		"SendBulk(ctx context.Context) error",
		"Receive(ctx context.Context) error",
	}
	for i, m := range ms {
		if m.Decl != want[i] {
			t.Errorf("methods[%d].Decl = %q, want %q", i, m.Decl, want[i])
		}
		if len(m.UsesPkgs) != 1 || m.UsesPkgs[0] != "context" {
			t.Errorf("methods[%d].UsesPkgs = %v, want [context]", i, m.UsesPkgs)
		}
	}
}

func TestParse_FullSignatures(t *testing.T) {
	spec := `CreateCharge(req ChargeRequest) (*ChargeResponse, error);
RefundCharge(id string) error;
GetCharge(ctx context.Context, id string) (*ChargeResponse, error)`
	ms, err := Parse(spec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(ms) != 3 {
		t.Fatalf("got %d, want 3", len(ms))
	}

	// Method 0: local types ChargeRequest, ChargeResponse; no pkgs.
	if got := sortedCopy(ms[0].LocalTypes); !equal(got, []string{"ChargeRequest", "ChargeResponse"}) {
		t.Errorf("CreateCharge LocalTypes = %v, want [ChargeRequest ChargeResponse]", got)
	}
	if len(ms[0].UsesPkgs) != 0 {
		t.Errorf("CreateCharge UsesPkgs = %v, want []", ms[0].UsesPkgs)
	}

	// Method 1: no locals, no pkgs.
	if len(ms[1].LocalTypes) != 0 || len(ms[1].UsesPkgs) != 0 {
		t.Errorf("RefundCharge expected no refs, got locals=%v pkgs=%v", ms[1].LocalTypes, ms[1].UsesPkgs)
	}

	// Method 2: context.Context reference -> "context" in UsesPkgs; ChargeResponse local.
	if got := sortedCopy(ms[2].UsesPkgs); !equal(got, []string{"context"}) {
		t.Errorf("GetCharge UsesPkgs = %v, want [context]", got)
	}
	if !contains(ms[2].LocalTypes, "ChargeResponse") {
		t.Errorf("GetCharge LocalTypes = %v, expected ChargeResponse", ms[2].LocalTypes)
	}

	// Decl should be valid Go fragments.
	for _, m := range ms {
		if !strings.Contains(m.Decl, m.Name+"(") {
			t.Errorf("Decl %q missing name+(", m.Decl)
		}
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		spec, contains string
	}{
		{"", "is empty"},
		{"foo", "exported Go identifier"},
		{"Foo,Foo", "duplicate"},
		{"Foo,bar", "exported Go identifier"},
		{"Foo(unbalanced", "could not parse"},
		{"foo() error", "exported Go identifier"},
		{"Foo() error; Foo() error", "duplicate"},
	}
	for _, c := range cases {
		_, err := Parse(c.spec)
		if err == nil {
			t.Errorf("Parse(%q): expected error, got nil", c.spec)
			continue
		}
		if !strings.Contains(err.Error(), c.contains) {
			t.Errorf("Parse(%q) error %q does not contain %q", c.spec, err.Error(), c.contains)
		}
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
