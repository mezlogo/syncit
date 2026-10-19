package processrunner

// ProcessRunnerPort executes OS commands.
type ProcessRunnerPort interface {
	// RunCommand runs name with args. When dryRun is true, prints the command and returns nil.
	RunCommand(dryRun bool, name string, args []string) error
}
