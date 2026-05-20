package doctor

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	present map[string]string // binary → version output
	ran     []string          // record of Run() calls (name + args, space-joined)
	runErr  map[string]error  // optional: per-binary Run error
}

func (f *fakeRunner) LookPath(name string) (string, error) {
	if _, ok := f.present[name]; ok {
		return "/fake/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.ran = append(f.ran, name+" "+strings.Join(args, " "))
	if err, ok := f.runErr[name]; ok {
		return "", err
	}
	if out, ok := f.present[name]; ok {
		return out, nil
	}
	return "", errors.New("binary not present")
}

func TestRun_AllPresent(t *testing.T) {
	r := &fakeRunner{present: map[string]string{
		"go":            "go version go1.22.0 darwin/arm64",
		"git":           "git version 2.43.0",
		"air":           "v1.52.3",
		"swag":          "swag version v1.16.2",
		"mockery":       "v2.43.0",
		"golangci-lint": "golangci-lint has version 1.57.2",
	}}
	results := Run(context.Background(), r, DefaultChecks())
	for _, res := range results {
		if !res.Found {
			t.Errorf("%s should be found", res.Name)
		}
	}
	var buf bytes.Buffer
	if ok := Report(&buf, results); !ok {
		t.Errorf("expected all satisfied")
	}
	if !strings.Contains(buf.String(), "All prerequisites satisfied") {
		t.Errorf("missing summary line:\n%s", buf.String())
	}
}

func TestRun_SomeMissing(t *testing.T) {
	r := &fakeRunner{present: map[string]string{
		"go":  "go version go1.22.0 darwin/arm64",
		"git": "git version 2.43.0",
	}}
	results := Run(context.Background(), r, DefaultChecks())
	var buf bytes.Buffer
	if ok := Report(&buf, results); ok {
		t.Errorf("expected not satisfied")
	}
	out := buf.String()
	for _, name := range []string{"air", "swag", "mockery", "golangci-lint"} {
		if !strings.Contains(out, "✗  "+name) {
			t.Errorf("missing ✗ for %s:\n%s", name, out)
		}
	}
}

func TestFix_InstallsMissing(t *testing.T) {
	r := &fakeRunner{present: map[string]string{
		"go":  "go version go1.22",
		"git": "git version 2.43.0",
		"air": "v1",
		// swag, mockery, golangci-lint missing
	}}
	// `go install …` calls are routed through Run("go", "install", "…@latest").
	// Our fake reports "go" as present so Run succeeds for it.
	results := Run(context.Background(), r, DefaultChecks())

	var buf bytes.Buffer
	err := Fix(context.Background(), r, results, &buf)
	if err != nil {
		t.Fatalf("Fix returned error: %v\n%s", err, buf.String())
	}
	out := buf.String()
	for _, name := range []string{"swag", "mockery", "golangci-lint"} {
		if !strings.Contains(out, "installing "+name) {
			t.Errorf("expected install message for %s:\n%s", name, out)
		}
	}
}

func TestFix_ManualOnly(t *testing.T) {
	r := &fakeRunner{present: map[string]string{}} // nothing present
	results := Run(context.Background(), r, DefaultChecks())
	var buf bytes.Buffer
	err := Fix(context.Background(), r, results, &buf)
	if err == nil {
		t.Fatal("expected error for unresolved manual tools")
	}
	if !strings.Contains(err.Error(), "go") || !strings.Contains(err.Error(), "git") {
		t.Errorf("expected go and git in error: %v", err)
	}
}
