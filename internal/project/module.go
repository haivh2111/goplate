// Package project introspects the target Go project that goplate is being run
// against — currently just locating its go.mod and extracting the module path.
package project

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// ErrNoGoMod is returned when no go.mod can be found in the starting directory
// or any of its parents.
var ErrNoGoMod = errors.New("no go.mod found in current directory or any parent")

// DetectModulePath walks up from start (inclusive) looking for a go.mod, and
// returns the module path declared inside it. It returns ErrNoGoMod if no
// go.mod exists in start or any ancestor.
func DetectModulePath(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	dir := abs
	for {
		candidate := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(candidate); err == nil {
			mod, err := modfile.Parse(candidate, data, nil)
			if err != nil {
				return "", err
			}
			if mod.Module == nil || mod.Module.Mod.Path == "" {
				return "", errors.New("go.mod has no module declaration")
			}
			return mod.Module.Mod.Path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoGoMod
		}
		dir = parent
	}
}
