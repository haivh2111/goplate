package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGeneratedFeatureCompiles is the strong correctness gate: it actually
// invokes `go build` and `go vet` against generated feature code in a tempdir
// scaffolded from internal/generator/testdata/compile_harness/.
//
// Catches the class of bugs where a template renders + parses but fails to
// type-check (wrong import path, missing receiver field, type mismatch, etc.).
//
// Slow on first run (downloads gorm + echo into the module cache). Skipped
// under `go test -short`.
func TestGeneratedFeatureCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile-harness test in -short mode")
	}
	harness := setupHarness(t)

	files, err := Generate(FeatureOptions{
		Name:        "product",
		FieldsSpec:  "name:string,price:float64,stock:int,active:bool",
		ModulePath:  "example.com/harness",
		ProjectRoot: harness,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	writeFiles(t, harness, files)

	runOrFail(t, harness, "go", "mod", "tidy")
	runOrFail(t, harness, "go", "build", "./...")
	runOrFail(t, harness, "go", "vet", "./...")
}

// TestGeneratedAdapterCompiles does the same drill for a new-adapter run.
// Verifies that port.go + provider adapter.go + adapter_test.go all type-check
// in a realistic project layout.
func TestGeneratedAdapterCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile-harness test in -short mode")
	}
	harness := setupHarness(t)

	files, _, err := GenerateAdapter(AdapterOptions{
		Service:     "payment",
		Provider:    "stripe",
		MethodsSpec: "CreateCharge(req ChargeRequest) (*ChargeResponse, error); RefundCharge(id string) error",
		ModulePath:  "example.com/harness",
		ProjectRoot: harness,
	})
	if err != nil {
		t.Fatalf("GenerateAdapter: %v", err)
	}
	writeFiles(t, harness, files)

	runOrFail(t, harness, "go", "mod", "tidy")
	runOrFail(t, harness, "go", "build", "./...")
	runOrFail(t, harness, "go", "vet", "./...")
}

// setupHarness copies testdata/compile_harness into a fresh tempdir.
func setupHarness(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if err := os.CopyFS(tmp, os.DirFS("testdata/compile_harness")); err != nil {
		t.Fatalf("copy harness: %v", err)
	}
	return tmp
}

// writeFiles materialises a generator's File list onto disk under root.
func writeFiles(t *testing.T, root string, files []File) {
	t.Helper()
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, f.Content, 0o644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
	}
}

// runOrFail runs a command in dir and fails the test on non-zero exit.
func runOrFail(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed in %s: %v\n%s", name, args, dir, err, out)
	}
}
