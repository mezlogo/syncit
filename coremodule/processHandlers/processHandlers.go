package processhandlers

import (
	"fmt"
	"strings"

	port "syncit/port/processHandlers"
	processrunner "syncit/port/processRunner"
)

var (
	preFileTools  = []string{"pacman", "aur"}
	postFileTools = []string{"systemd", "usersystemd"}
)

type UseCase struct {
	runner processrunner.ProcessRunnerPort
}

func New(runner processrunner.ProcessRunnerPort) *UseCase { return &UseCase{runner: runner} }

var _ port.ProcessHandlersUseCase = (*UseCase)(nil)

func (u *UseCase) ProcessPreFileHandlers(cmd port.ProcessHandlersCommand) port.ProcessHandlersResult {
	return u.processTools(preFileTools, cmd)
}

func (u *UseCase) ProcessPostFileHandlers(cmd port.ProcessHandlersCommand) port.ProcessHandlersResult {
	return u.processTools(postFileTools, cmd)
}

func (u *UseCase) processTools(tools []string, cmd port.ProcessHandlersCommand) port.ProcessHandlersResult {
	decisions := make([]port.ProcessDecision, 0)
	for _, tool := range tools {
		template, ok := cmd.Handlers[tool]
		if !ok {
			continue
		}
		values := uniqueNonEmpty(cmd.Declarations[tool])
		if len(values) == 0 {
			continue
		}
		decisions = append(decisions, u.executeTemplate(cmd.DryRun, tool, template, values)...)
	}
	return port.ProcessHandlersResult{Decisions: decisions}
}

func (u *UseCase) executeTemplate(dryRun bool, tool, template string, values []string) []port.ProcessDecision {
	switch {
	case strings.Contains(template, placeholderAllUnique):
		return []port.ProcessDecision{
			u.run(dryRun, tool, substituteAllUnique(template, values), values),
		}
	case strings.Contains(template, placeholderSingle):
		out := make([]port.ProcessDecision, 0, len(values))
		for _, v := range values {
			out = append(out, u.run(dryRun, tool, substituteSingle(template, v), []string{v}))
		}
		return out
	default:
		return []port.ProcessDecision{u.run(dryRun, tool, template, values)}
	}
}

func (u *UseCase) run(dryRun bool, tool, command string, values []string) port.ProcessDecision {
	d := port.ProcessDecision{ToolName: tool, Command: command, Values: values}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		d.Code = port.ProcessDecisionCodeFailed
		d.Err = fmt.Errorf("empty command for tool %q", tool)
		return d
	}
	if err := u.runner.RunCommand(dryRun, parts[0], parts[1:]); err != nil {
		d.Code = port.ProcessDecisionCodeFailed
		d.Err = err
		return d
	}
	if dryRun {
		d.Code = port.ProcessDecisionCodeDryRun
	} else {
		d.Code = port.ProcessDecisionCodeExecuted
	}
	return d
}
