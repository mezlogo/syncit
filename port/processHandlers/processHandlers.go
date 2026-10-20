package processhandlers

// ProcessHandlersCommand carries declared values and dry-run flag.
type ProcessHandlersCommand struct {

	// Declarations maps tool name -> declared values aggregated from modules.
	Declarations map[string][]string
	// DryRun prints commands instead of executing them.
	DryRun bool
}

type ProcessDecisionCode string

const (
	ProcessDecisionCodeExecuted ProcessDecisionCode = "EXECUTED"
	ProcessDecisionCodeDryRun   ProcessDecisionCode = "DRY_RUN"
	ProcessDecisionCodeFailed   ProcessDecisionCode = "FAILED"
)

type ProcessDecision struct {
	ToolName string
	Command  string
	Values   []string
	Code     ProcessDecisionCode
	Err      error
}

type ProcessHandlersResult struct {
	Decisions []ProcessDecision
}

// ProcessHandlersUseCase runs process handlers in fixed phases.
type ProcessHandlersUseCase interface {
	// ProcessPreFileHandlers runs grouped handlers (pacman, aur) before file sync.
	ProcessPreFileHandlers(cmd ProcessHandlersCommand) ProcessHandlersResult
	// ProcessPostFileHandlers runs per-value handlers (systemd, usersystemd) after file sync.
	ProcessPostFileHandlers(cmd ProcessHandlersCommand) ProcessHandlersResult
}
