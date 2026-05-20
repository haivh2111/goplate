package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestInitGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InitGit(root, "initial commit"); err != nil {
		t.Fatalf("InitGit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Errorf("expected .git dir, got %v", err)
	}
}

func TestCopyEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyEnv(root, ".env.example", ".env"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "KEY=value\n" {
		t.Errorf("got %q, want KEY=value", got)
	}
}

func TestCopyEnv_MissingSourceIsNoOp(t *testing.T) {
	root := t.TempDir()
	if err := CopyEnv(root, "nonexistent.env", ".env"); err != nil {
		t.Errorf("expected no error for missing source, got %v", err)
	}
}
