package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Writer is the interface RunE uses to materialize generator output.
type Writer interface {
	Write(files []File) error
}

// FSWriter writes files under Root. It refuses to overwrite any existing path.
type FSWriter struct {
	Root   string
	Stdout io.Writer
}

// Write creates or replaces files. Files with Overwrite==OverwriteFail abort
// the entire batch if they already exist (check-then-write atomicity).
// Files with Overwrite==OverwriteReplace are written unconditionally.
func (w FSWriter) Write(files []File) error {
	if w.Stdout == nil {
		w.Stdout = os.Stdout
	}
	var conflicts []string
	for _, f := range files {
		if f.Overwrite == OverwriteReplace {
			continue
		}
		abs := filepath.Join(w.Root, filepath.FromSlash(f.RelPath))
		if _, err := os.Stat(abs); err == nil {
			conflicts = append(conflicts, f.RelPath)
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("refusing to overwrite existing files:\n  - %s\n(delete them and re-run, or pick a different name)",
			strings.Join(conflicts, "\n  - "))
	}
	for _, f := range files {
		abs := filepath.Join(w.Root, filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, f.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", abs, err)
		}
		marker := "✓"
		if f.Overwrite == OverwriteReplace {
			marker = "✎"
		}
		fmt.Fprintf(w.Stdout, "  %s  %s\n", marker, f.RelPath)
	}
	return nil
}

// DryRunWriter prints proposed file paths and contents to Out without touching
// disk. Output is identical in bytes to what FSWriter would write.
type DryRunWriter struct {
	Out io.Writer
}

// Write prints each file's banner + content, then a footer.
func (w DryRunWriter) Write(files []File) error {
	out := w.Out
	if out == nil {
		out = os.Stdout
	}
	for _, f := range files {
		suffix := ""
		if f.Overwrite == OverwriteReplace {
			suffix = " (would REPLACE existing)"
		}
		fmt.Fprintf(out, "── %s%s ──\n", f.RelPath, suffix)
		_, _ = out.Write(f.Content)
		if len(f.Content) > 0 && f.Content[len(f.Content)-1] != '\n' {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintf(out, "(dry-run: %d files — no files written)\n", len(files))
	return nil
}
