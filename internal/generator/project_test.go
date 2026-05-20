package generator

import (
	"bytes"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestGenerateProject_MySQL_AllFilesPresent(t *testing.T) {
	files, err := GenerateProject(ProjectOptions{
		Name:       "demo-svc",
		ModulePath: "github.com/acme/demo-svc",
		DBDriver:   "mysql",
		OutputDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("GenerateProject: %v", err)
	}

	// Spot-check the critical files exist (no .tmpl extension in output).
	required := []string{
		"go.mod",
		"Makefile",
		"Dockerfile",
		".gitignore",
		".env.example",
		"README.md",
		"docker-compose.yml",
		"cmd/main.go",
		"config/config.go",
		"internal/server/server.go",
		"internal/server/providers.go",
		"internal/server/subscribers.go",
		"internal/server/features.go",
		"internal/middleware/jwt.go",
		"internal/middleware/error_handler.go",
		"internal/shared/response.go",
		"internal/shared/errors.go",
		"internal/shared/validator.go",
		"internal/events/event_bus.go",
		"internal/events/event_types.go",
		"internal/infra/database/db.go",
		"internal/infra/database/migrate.go",
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.RelPath] = true
	}
	for _, want := range required {
		if !got[want] {
			t.Errorf("missing file %s", want)
		}
	}
}

func TestGenerateProject_AllGoFilesParse(t *testing.T) {
	for _, driver := range []string{"mysql", "postgres", "sqlite"} {
		t.Run(driver, func(t *testing.T) {
			files, err := GenerateProject(ProjectOptions{
				Name:       "demo",
				ModulePath: "github.com/acme/demo",
				DBDriver:   driver,
				OutputDir:  t.TempDir(),
			})
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			for _, f := range files {
				if !strings.HasSuffix(f.RelPath, ".go") {
					continue
				}
				if _, err := parser.ParseFile(fset, f.RelPath, f.Content, parser.AllErrors); err != nil {
					t.Errorf("%s: %v\n---\n%s", f.RelPath, err, f.Content)
				}
			}
		})
	}
}

func TestGenerateProject_ModulePathPropagates(t *testing.T) {
	files, err := GenerateProject(ProjectOptions{
		Name:       "demo",
		ModulePath: "github.com/acme/demo",
		DBDriver:   "mysql",
		OutputDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.RelPath == "go.mod" {
			if !bytes.Contains(f.Content, []byte("module github.com/acme/demo")) {
				t.Errorf("go.mod missing module path:\n%s", f.Content)
			}
		}
		if f.RelPath == "cmd/main.go" {
			if !bytes.Contains(f.Content, []byte(`"github.com/acme/demo/config"`)) {
				t.Errorf("cmd/main.go should import config with module path:\n%s", f.Content)
			}
		}
	}
}

func TestGenerateProject_RejectsBadInput(t *testing.T) {
	cases := []ProjectOptions{
		{Name: "", ModulePath: "x", DBDriver: "mysql"},
		{Name: "demo", ModulePath: "", DBDriver: "mysql"},
		{Name: "demo", ModulePath: "x", DBDriver: "oracle"},
	}
	for i, c := range cases {
		if _, err := GenerateProject(c); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}

func TestGenerateProject_SQLiteOmitsFmtImport(t *testing.T) {
	files, err := GenerateProject(ProjectOptions{
		Name:       "demo",
		ModulePath: "github.com/acme/demo",
		DBDriver:   "sqlite",
		OutputDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.RelPath == "internal/infra/database/db.go" {
			if bytes.Contains(f.Content, []byte(`"fmt"`)) {
				t.Errorf("sqlite db.go should not import fmt:\n%s", f.Content)
			}
		}
	}
}
