package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModulePath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module github.com/acme/demo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		root:                 "github.com/acme/demo",
		filepath.Join(root, "a"): "github.com/acme/demo",
		nested:               "github.com/acme/demo",
	}
	for dir, want := range cases {
		got, err := DetectModulePath(dir)
		if err != nil {
			t.Errorf("DetectModulePath(%s) error: %v", dir, err)
			continue
		}
		if got != want {
			t.Errorf("DetectModulePath(%s) = %q, want %q", dir, got, want)
		}
	}
}

func TestDetectModulePath_NotFound(t *testing.T) {
	// Use a temp dir that contains no go.mod and whose parents (above tempdir
	// root) are guaranteed not to have one in CI; we test for the sentinel.
	dir := t.TempDir()
	// Create a deep nested dir so walking up still terminates without ever
	// finding our own go.mod (the goplate one) — we cross filesystem root
	// from the temp root which is on /tmp.
	_, err := DetectModulePath(dir)
	if err == nil {
		t.Skip("environment unexpectedly has go.mod above tempdir; cannot test ErrNoGoMod")
	}
	if !errors.Is(err, ErrNoGoMod) {
		t.Errorf("got %v, want ErrNoGoMod", err)
	}
}

func TestFeatureImportPath(t *testing.T) {
	got := FeatureImportPath("github.com/acme/svc", "product")
	want := "github.com/acme/svc/internal/features/product"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
