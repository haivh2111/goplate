// Package doctor verifies that the host has the toolchain goplate-generated
// projects expect (go, git, air, swag, mockery, golangci-lint) and can
// optionally install the auto-installable ones via `go install`.
package doctor

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Check describes one prerequisite binary and how to detect / install it.
type Check struct {
	Name        string   // user-facing name ("swag", "golangci-lint")
	Binary      string   // executable to look up on PATH
	VersionArgs []string // args to print the tool's version
	InstallCmd  []string // `go install ...@latest` argv, or nil if manual-only
	ManualHint  string   // shown when InstallCmd is nil and the tool is missing
}

// Result captures the outcome of running a single check.
type Result struct {
	Check
	Found    bool
	Version  string // first non-empty line of version output (trimmed)
	DetectErr error
}

// Runner abstracts exec.LookPath / exec.Command so tests can inject behaviour.
type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
}

// realRunner is the production Runner backed by os/exec.
type realRunner struct{}

func (realRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (realRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// DefaultChecks lists the prerequisites in the order doctor reports them.
func DefaultChecks() []Check {
	return []Check{
		{
			Name:        "go",
			Binary:      "go",
			VersionArgs: []string{"version"},
			ManualHint:  "install Go 1.22+ from https://go.dev/dl/",
		},
		{
			Name:        "git",
			Binary:      "git",
			VersionArgs: []string{"--version"},
			ManualHint:  "install git from your OS package manager",
		},
		{
			Name:        "air",
			Binary:      "air",
			VersionArgs: []string{"-v"},
			InstallCmd:  []string{"go", "install", "github.com/air-verse/air@latest"},
		},
		{
			Name:        "swag",
			Binary:      "swag",
			VersionArgs: []string{"--version"},
			InstallCmd:  []string{"go", "install", "github.com/swaggo/swag/cmd/swag@latest"},
		},
		{
			Name:        "mockery",
			Binary:      "mockery",
			VersionArgs: []string{"--version"},
			InstallCmd:  []string{"go", "install", "github.com/vektra/mockery/v2@latest"},
		},
		{
			Name:        "golangci-lint",
			Binary:      "golangci-lint",
			VersionArgs: []string{"--version"},
			InstallCmd:  []string{"go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"},
		},
	}
}

// Run executes every check using r (or the real runner if r is nil).
func Run(ctx context.Context, r Runner, checks []Check) []Result {
	if r == nil {
		r = realRunner{}
	}
	out := make([]Result, 0, len(checks))
	for _, c := range checks {
		res := Result{Check: c}
		if _, err := r.LookPath(c.Binary); err != nil {
			res.DetectErr = err
			out = append(out, res)
			continue
		}
		res.Found = true
		stdout, err := r.Run(ctx, c.Binary, c.VersionArgs...)
		if err != nil {
			// Found but version probe failed — still treat as present.
			res.Version = "(version probe failed)"
		} else {
			res.Version = firstLine(stdout)
		}
		out = append(out, res)
	}
	return out
}

// Report prints results in the spec's checkmark format and returns true iff
// every check passed.
func Report(w io.Writer, results []Result) bool {
	all := true
	for _, r := range results {
		if r.Found {
			fmt.Fprintf(w, "  ✓  %s %s\n", r.Name, r.Version)
		} else {
			all = false
			fmt.Fprintf(w, "  ✗  %s not found\n", r.Name)
		}
	}
	if all {
		fmt.Fprintln(w, "  All prerequisites satisfied.")
		return true
	}
	missing := make([]string, 0)
	for _, r := range results {
		if !r.Found {
			missing = append(missing, r.Name)
		}
	}
	fmt.Fprintf(w, "  Missing: %s — run with --fix to install (where supported)\n",
		strings.Join(missing, ", "))
	return false
}

// Fix installs any missing tool that has an InstallCmd. Tools without one
// (go, git) only print their ManualHint. Returns nil only if every fixable
// tool installed successfully AND every manual-only tool was already present.
func Fix(ctx context.Context, r Runner, results []Result, w io.Writer) error {
	if r == nil {
		r = realRunner{}
	}
	var unresolved []string
	for _, res := range results {
		if res.Found {
			continue
		}
		if res.InstallCmd == nil {
			fmt.Fprintf(w, "  ✗  %s: %s\n", res.Name, res.ManualHint)
			unresolved = append(unresolved, res.Name)
			continue
		}
		fmt.Fprintf(w, "  ➜  installing %s: %s\n", res.Name, strings.Join(res.InstallCmd, " "))
		stdout, err := r.Run(ctx, res.InstallCmd[0], res.InstallCmd[1:]...)
		if err != nil {
			fmt.Fprintf(w, "       failed: %v\n%s\n", err, indent(stdout))
			unresolved = append(unresolved, res.Name)
			continue
		}
		fmt.Fprintf(w, "       installed\n")
	}
	if len(unresolved) > 0 {
		return fmt.Errorf("unresolved tools: %s", strings.Join(unresolved, ", "))
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	first, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(first)
}

func indent(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "       " + l
	}
	return strings.Join(lines, "\n")
}
