package cli

// globalFlags holds the cobra persistent-flag bag for the root command.
// It is populated by cobra before any RunE is invoked.
var globalFlags struct {
	DryRun bool
}

// DryRun reports whether --dry-run was set anywhere on the command line.
func DryRun() bool { return globalFlags.DryRun }
