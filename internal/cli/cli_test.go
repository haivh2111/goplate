package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestRootHelp_ListsAllSubcommands guards against accidental command removal.
func TestRootHelp_ListsAllSubcommands(t *testing.T) {
	out := runHelp(t, []string{"--help"})
	for _, name := range []string{"doctor", "new ", "new-feature", "new-adapter", "new-event"} {
		if !strings.Contains(out, name) {
			t.Errorf("root --help missing %q in:\n%s", name, out)
		}
	}
}

// TestSubcommandFlags_PreserveSpecNames guards against flag-name regressions.
// The spec is the user-facing contract; help text is part of it.
func TestSubcommandFlags_PreserveSpecNames(t *testing.T) {
	cases := []struct {
		cmd     []string
		flags   []string
		example string // example value from spec docs (for short-flag presence)
	}{
		{
			cmd:   []string{"new", "--help"},
			flags: []string{"--module", "-m", "--output", "-o", "--db", "--no-git", "--no-env", "--skip-tidy"},
		},
		{
			cmd:   []string{"new-feature", "--help"},
			flags: []string{"--fields", "-f", "--no-auth", "--no-pagination", "--dry-run"},
		},
		{
			cmd:   []string{"new-adapter", "--help"},
			flags: []string{"--methods", "-m", "--stub-siblings"},
		},
		{
			cmd:   []string{"new-event", "--help"},
			flags: []string{"--payload", "-p", "--subscriber", "-s"},
		},
		{
			cmd:   []string{"doctor", "--help"},
			flags: []string{"--fix"},
		},
	}
	for _, c := range cases {
		out := runHelp(t, c.cmd)
		for _, f := range c.flags {
			if !strings.Contains(out, f) {
				t.Errorf("%v: help missing flag %q in:\n%s", c.cmd, f, out)
			}
		}
	}
}

func runHelp(t *testing.T, args []string) string {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute %v: %v\noutput:\n%s", args, err, buf.String())
	}
	return buf.String()
}
