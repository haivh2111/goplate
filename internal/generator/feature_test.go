package generator

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGenerate_AllTemplatesRenderAndParse(t *testing.T) {
	opts := FeatureOptions{
		Name:       "product",
		FieldsSpec: "name:string,price:float64,stock:int,active:bool",
		ModulePath: "github.com/acme/demo",
	}
	files, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	wantPaths := []string{
		"internal/features/product/model.go",
		"internal/features/product/dto.go",
		"internal/features/product/repository.go",
		"internal/features/product/repository_mysql.go",
		"internal/features/product/service.go",
		"internal/features/product/service_impl.go",
		"internal/features/product/handler.go",
		"internal/features/product/module.go",
		"internal/features/product/service_impl_test.go",
		"internal/features/product/handler_test.go",
		"internal/features/product/repository_mysql_test.go",
	}
	if len(files) != len(wantPaths) {
		t.Fatalf("got %d files, want %d", len(files), len(wantPaths))
	}
	for i, w := range wantPaths {
		if files[i].RelPath != w {
			t.Errorf("file[%d] = %q, want %q", i, files[i].RelPath, w)
		}
	}

	// Every file must parse as Go syntax. Each file is already piped through
	// go/format before reaching us, but this is a second-layer guard against
	// any template bug that produces parse-but-not-format-able output.
	fset := token.NewFileSet()
	for _, f := range files {
		if _, err := parser.ParseFile(fset, f.RelPath, f.Content, parser.AllErrors); err != nil {
			t.Errorf("%s does not parse:\n%v\n--- file ---\n%s", f.RelPath, err, f.Content)
		}
	}
}

func TestGenerate_TimeFieldImportsTime(t *testing.T) {
	files, err := Generate(FeatureOptions{
		Name:       "event",
		FieldsSpec: "title:string,occurredAt:time.Time",
		ModulePath: "github.com/acme/demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	var model []byte
	for _, f := range files {
		if strings.HasSuffix(f.RelPath, "model.go") {
			model = f.Content
		}
	}
	if !bytes.Contains(model, []byte(`"time"`)) {
		t.Errorf("expected model.go to import \"time\", got:\n%s", model)
	}
}

func TestGenerate_NoAuthOmitsMiddleware(t *testing.T) {
	files, err := Generate(FeatureOptions{
		Name:       "report",
		FieldsSpec: "name:string",
		ModulePath: "github.com/acme/demo",
		NoAuth:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var module []byte
	for _, f := range files {
		if strings.HasSuffix(f.RelPath, "module.go") {
			module = f.Content
		}
	}
	if bytes.Contains(module, []byte("middleware")) {
		t.Errorf("module.go should not import middleware when --no-auth, got:\n%s", module)
	}
	if !bytes.Contains(module, []byte("g.GET(\"\", h.List)")) {
		t.Errorf("expected unauth-listed GET route, got:\n%s", module)
	}
}

func TestGenerate_RejectsBadNames(t *testing.T) {
	bad := []string{"", "Product", "my_feature", "1abc", "func", "ab-cd"}
	for _, name := range bad {
		_, err := Generate(FeatureOptions{
			Name:       name,
			FieldsSpec: "name:string",
			ModulePath: "github.com/acme/demo",
		})
		if err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestGenerate_RequiresFields(t *testing.T) {
	_, err := Generate(FeatureOptions{
		Name:       "product",
		FieldsSpec: "",
		ModulePath: "github.com/acme/demo",
	})
	if err == nil {
		t.Fatal("expected error for empty fields")
	}
}

func TestNextSteps_FormatMatchesSpec(t *testing.T) {
	out := NextSteps(FeatureOptions{Name: "product"})
	wants := []string{
		"product.Register(api, p) in internal/server/server.go",
		"&product.Product{} to internal/infra/database/migrate.go",
		"make swag && make test",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("next-steps missing %q\nfull output:\n%s", w, out)
		}
	}
}
