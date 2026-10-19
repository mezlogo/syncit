# Milestone: OS Process Support

Interpretation of the handler format: handlers live in a repo-root `handlers.yaml` (satisfies `map[string]string` shape from your JSON) while module values stay in `module.yaml` (`map[string][]string`). Fixed execution order: **pacman → aur → user files → root files → systemd → usersystemd**. Placeholders:
- `$ALL_UNIQUE_VALUES_JOIN_WHITESPACE` → one command with all unique values
- `$UNIQUE_SINGLE_VALUE` → one command per unique value

---

## New files

### `port/processRunner/processRunner.go`
```go
package processrunner

// ProcessRunnerPort executes OS commands.
type ProcessRunnerPort interface {
	// RunCommand runs name with args. When dryRun is true, prints the command and returns nil.
	RunCommand(dryRun bool, name string, args []string) error
}
```

### `port/processHandlers/processHandlers.go`
```go
package processhandlers

// ProcessHandlersCommand carries handler templates and declared values.
type ProcessHandlersCommand struct {
	// Handlers maps tool name -> command template.
	Handlers map[string]string
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
```

### `adapter/processRunner/processRunner.go`
```go
package processrunner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	port "syncit/port/processRunner"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

var _ port.ProcessRunnerPort = (*Adapter)(nil)

func (a *Adapter) RunCommand(dryRun bool, name string, args []string) error {
	if dryRun {
		_, _ = fmt.Fprintln(os.Stdout, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

### `coremodule/processHandlers/convert.go`
```go
package processhandlers

import (
	"sort"
	"strings"
)

const (
	placeholderAllUnique = "$ALL_UNIQUE_VALUES_JOIN_WHITESPACE"
	placeholderSingle    = "$UNIQUE_SINGLE_VALUE"
)

func substituteAllUnique(template string, values []string) string {
	return strings.ReplaceAll(template, placeholderAllUnique, strings.Join(values, " "))
}

func substituteSingle(template, value string) string {
	return strings.ReplaceAll(template, placeholderSingle, value)
}

// uniqueNonEmpty trims, deduplicates and sorts values for deterministic output.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
```

### `coremodule/processHandlers/processHandlers.go`
```go
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
```

### `coremodule/processHandlers/processHandlers_test.go`
```go
package processhandlers

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	port "syncit/port/processHandlers"
)

type fakeRunner struct {
	calls []string
	err   error
}

func (f *fakeRunner) RunCommand(dryRun bool, name string, args []string) error {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	return f.err
}

func TestUseCase_ProcessPreFileHandlers(t *testing.T) {
	tests := []struct {
		name      string
		cmd       port.ProcessHandlersCommand
		runnerErr error
		wantCalls []string
		wantCodes []port.ProcessDecisionCode
	}{
		{
			name: "pacman grouped unique",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim", "fish", "nvim"}},
			},
			wantCalls: []string{"sudo pacman -S --needed fish nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeExecuted},
		},
		{
			name: "aur grouped",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"aur": "yay -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"aur": {"vscode", "leaf"}},
			},
			wantCalls: []string{"yay -S --needed leaf vscode"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeExecuted},
		},
		{
			name: "dry run",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim"}},
				DryRun:       true,
			},
			wantCalls: []string{"sudo pacman -S --needed nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeDryRun},
		},
		{
			name: "runner error",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S --needed $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"pacman": {"nvim"}},
			},
			runnerErr: errors.New("boom"),
			wantCalls: []string{"sudo pacman -S --needed nvim"},
			wantCodes: []port.ProcessDecisionCode{port.ProcessDecisionCodeFailed},
		},
		{
			name: "skips missing handler and empty values",
			cmd: port.ProcessHandlersCommand{
				Handlers:     map[string]string{"pacman": "sudo pacman -S $ALL_UNIQUE_VALUES_JOIN_WHITESPACE"},
				Declarations: map[string][]string{"aur": {"vscode"}},
			},
			wantCalls: nil,
			wantCodes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{err: tt.runnerErr}
			result := New(runner).ProcessPreFileHandlers(tt.cmd)

			if !reflect.DeepEqual(runner.calls, tt.wantCalls) {
				t.Errorf("calls: got %#v, want %#v", runner.calls, tt.wantCalls)
			}
			if len(result.Decisions) != len(tt.wantCodes) {
				t.Fatalf("decisions: got %d, want %d", len(result.Decisions), len(tt.wantCodes))
			}
			for i, wantCode := range tt.wantCodes {
				if result.Decisions[i].Code != wantCode {
					t.Errorf("decision %d: got %q, want %q", i, result.Decisions[i].Code, wantCode)
				}
			}
		})
	}
}

func TestUseCase_ProcessPostFileHandlers_PerValue(t *testing.T) {
	runner := &fakeRunner{}
	cmd := port.ProcessHandlersCommand{
		Handlers: map[string]string{
			"systemd":     "sudo systemctl enable $UNIQUE_SINGLE_VALUE",
			"usersystemd": "systemctl --user enable $UNIQUE_SINGLE_VALUE",
		},
		Declarations: map[string][]string{
			"systemd":     {"netd.service", "resolved.service"},
			"usersystemd": {"ssh-agent.service"},
		},
	}

	result := New(runner).ProcessPostFileHandlers(cmd)

	wantCalls := []string{
		"sudo systemctl enable netd.service",
		"sudo systemctl enable resolved.service",
		"systemctl --user enable ssh-agent.service",
	}
	if !reflect.DeepEqual(runner.calls, wantCalls) {
		t.Errorf("calls: got %#v, want %#v", runner.calls, wantCalls)
	}
	if len(result.Decisions) != 3 {
		t.Errorf("decisions: got %d, want 3", len(result.Decisions))
	}
}
```

---

## Modified files

### `port/repoAO/repoAO.go` — add handler reading + sentinel
```go
// ... existing imports/types unchanged ...

var (
	ErrNoRepositoryFound             = errors.New("NO_REPOSITORY_FOUND")
	ErrNoHostFileFound               = errors.New("NO_HOST_FILE_FOUND")
	ErrBadHostFile                   = errors.New("BAD_HOST_FILE")
	ErrDeclaredModuleNotFound        = errors.New("DECLARED_MODULE_NOT_FOUND")
	ErrModuleDeclarationFileNotFound = errors.New("MODULE_DECLARATION_FILE_NOT_FOUND")
	ErrBadModuleDeclaration          = errors.New("BAD_MODULE_DECLARATION")
	ErrBadHandlersFile               = errors.New("BAD_HANDLERS_FILE")
)

type RepositoryDeclarationPort interface {
	ParseRepository(command RepositoryDeclarationCommand) (RepositoryDeclarationResult, error)

	// ParseHandlers reads the repo-root handlers.yaml (tool -> template).
	// Missing handlers.yaml returns an empty map and no error.
	ParseHandlers(repoPath string) (map[string]string, error)
}
```

### `adapter/repoAO/repoAO.go` — implement `ParseHandlers`
```go
const handlersFileName = "handlers.yaml"

func (a *Adapter) ParseHandlers(repoPath string) (map[string]string, error) {
	if repoPath == "" {
		return nil, fmt.Errorf("%w: repository path is empty", port.ErrNoRepositoryFound)
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repository path: %w", err)
	}

	handlersFile := filepath.Join(absRepo, handlersFileName)
	data, err := os.ReadFile(handlersFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read handlers file: %w", err)
	}

	handlers := map[string]string{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return handlers, nil
	}
	if err := yaml.Unmarshal(data, &handlers); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", port.ErrBadHandlersFile, handlersFile, err)
	}
	return handlers, nil
}
```

### `port/repoReport/repoReport.go` — add process report
```go
import (
	// ...
	processhandlers "syncit/port/processHandlers"
)

type RepoReportUseCase interface {
	// ... existing methods ...

	// ReportProcess reports process decisions (dry-run or executed) with colorized codes.
	ReportProcess(repoPath string, decisions []processhandlers.ProcessDecision, mergeErr error)
}
```

### `adapter/repoReport/repoReport.go` — add process report + colorizer
```go
import (
	// ...
	processhandlers "syncit/port/processHandlers"
)

func (a *Adapter) ReportProcess(repoPath string, decisions []processhandlers.ProcessDecision, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("process repository: " + repoPath)
	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}
	if len(decisions) == 0 {
		pterm.Println("[]")
		return
	}
	rows := pterm.TableData{{"tool", "values", "command", "decision", "error"}}
	for _, d := range decisions {
		errStr := ""
		if d.Err != nil {
			errStr = d.Err.Error()
		}
		rows = append(rows, []string{
			d.ToolName,
			strings.Join(d.Values, " "),
			d.Command,
			colorizeProcessCode(d.Code),
			errStr,
		})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

func colorizeProcessCode(code processhandlers.ProcessDecisionCode) string {
	switch code {
	case processhandlers.ProcessDecisionCodeExecuted:
		return colorGreen + string(code) + colorReset
	case processhandlers.ProcessDecisionCodeDryRun:
		return colorCyan + string(code) + colorReset
	case processhandlers.ProcessDecisionCodeFailed:
		return colorRed + string(code) + colorReset
	}
	return string(code)
}
```

### `cmd/sync/sync.go` — full replacement
```go
package sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	processhandlers "syncit/port/processHandlers"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	syncfileconfiguration "syncit/port/syncFileConfiguration"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	syncPort syncfileconfiguration.SyncFileConfigurationUseCase,
	processPort processhandlers.ProcessHandlersUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName, outputHome, outputRoot string
	var overwrite, reset, dryRun bool
	var includeUser, includeRoot, includeProcess bool

	defaultRepositories := splitCommaSeparated(os.Getenv("SYNCIT_REPO"))
	defaultHostName := strings.TrimSpace(os.Getenv("SYNCIT_HOST"))
	if defaultHostName == "" {
		if h, err := os.Hostname(); err == nil {
			defaultHostName = h
		}
	}
	defaultOutputHome := strings.TrimSpace(os.Getenv("HOME"))
	if defaultOutputHome == "" {
		if h, err := os.UserHomeDir(); err == nil {
			defaultOutputHome = h
		}
	}
	defaultOutputRoot := "/"

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize file configuration and run process handlers",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := normalizeList(repositories)
			if len(repos) == 0 {
				return errors.New("repository is required: use --repo or SYNCIT_REPO")
			}
			host := strings.TrimSpace(hostName)
			if host == "" {
				return errors.New("host is required: use --host or SYNCIT_HOST")
			}

			modules := make([]repositorydeclaration.RepositoryDeclarationModule, 0)
			mergedHandlers := map[string]string{}
			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				modules = append(modules, result.Modules...)

				handlers, err := declarationPort.ParseHandlers(repo)
				if err != nil {
					return fmt.Errorf("parse handlers %q: %w", repo, err)
				}
				for k, v := range handlers {
					mergedHandlers[k] = v
				}
			}

			mergedDeclarations := map[string][]string{}
			for _, m := range modules {
				for k, vals := range m.Declarations {
					mergedDeclarations[k] = append(mergedDeclarations[k], vals...)
				}
			}

			anyInclude := includeUser || includeRoot || includeProcess
			processUser := includeUser || !anyInclude
			processRoot := includeRoot || !anyInclude
			processHandlers := includeProcess || !anyInclude

			mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
			resolved, mergeErr := mergePort.MergeFiles(modules, mapping)
			opts := syncfileconfiguration.SyncOptions{Overwrite: overwrite, Reset: reset}

			processCmd := processhandlers.ProcessHandlersCommand{
				Handlers:     mergedHandlers,
				Declarations: mergedDeclarations,
				DryRun:       dryRun,
			}

			if processHandlers {
				pre := processPort.ProcessPreFileHandlers(processCmd)
				reportPort.ReportProcess(strings.Join(repos, ", "), pre.Decisions, nil)
			}
			if processUser {
				reportPort.ReportSync(strings.Join(repos, ", "),
					syncPort.SyncUserFiles(resolved.UserFiles, opts), mergeErr)
			}
			if processRoot {
				reportPort.ReportSyncRoot(strings.Join(repos, ", "),
					syncPort.SyncRootFiles(resolved.RootFiles, opts), mergeErr)
			}
			if processHandlers {
				post := processPort.ProcessPostFileHandlers(processCmd)
				reportPort.ReportProcess(strings.Join(repos, ", "), post.Decisions, nil)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", defaultOutputRoot, "root directory used for file mapping; defaults to /")
	cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeProcess, "include-process", false, "process process handlers when any include flag is set")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "recreate existing files or wrong links at the target")
	cmd.Flags().BoolVar(&reset, "reset", false, "recreate the link even when it already points to the declared file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print process handler commands instead of executing them")

	return cmd
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, v := range values {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				result = append(result, p)
			}
		}
	}
	return result
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeList([]string{value})
}
```

### `cmd/report/report.go` — full replacement
```go
package report

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	processhandlers "syncit/port/processHandlers"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	diffPort difffileconfiguration.DiffFileConfigurationUseCase,
	processPort processhandlers.ProcessHandlersUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName string
	var merged, diff bool
	var outputHome, outputRoot string
	var includeUser, includeRoot, includeProcess bool

	defaultRepositories := splitCommaSeparated(os.Getenv("SYNCIT_REPO"))
	defaultHostName := strings.TrimSpace(os.Getenv("SYNCIT_HOST"))
	if defaultHostName == "" {
		if hostname, err := os.Hostname(); err == nil {
			defaultHostName = hostname
		}
	}
	defaultOutputHome := strings.TrimSpace(os.Getenv("HOME"))
	if defaultOutputHome == "" {
		if home, err := os.UserHomeDir(); err == nil {
			defaultOutputHome = home
		}
	}
	defaultOutputRoot := "/"

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Print repository declaration, merged files, diff or process list",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := normalizeList(repositories)
			if len(repos) == 0 {
				return errors.New("repository is required: use --repo or SYNCIT_REPO")
			}
			host := strings.TrimSpace(hostName)
			if host == "" {
				return errors.New("host is required: use --host or SYNCIT_HOST")
			}

			if merged || diff || includeProcess {
				modules := make([]repositorydeclaration.RepositoryDeclarationModule, 0)
				mergedHandlers := map[string]string{}
				for _, repo := range repos {
					result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
						RepositoryPath: repo,
						HostName:       host,
					})
					if err != nil {
						return fmt.Errorf("parse repository %q: %w", repo, err)
					}
					modules = append(modules, result.Modules...)

					handlers, err := declarationPort.ParseHandlers(repo)
					if err != nil {
						return fmt.Errorf("parse handlers %q: %w", repo, err)
					}
					for k, v := range handlers {
						mergedHandlers[k] = v
					}
				}

				mergedDeclarations := map[string][]string{}
				for _, m := range modules {
					for k, vals := range m.Declarations {
						mergedDeclarations[k] = append(mergedDeclarations[k], vals...)
					}
				}

				anyInclude := includeUser || includeRoot || includeProcess
				processUser := includeUser || !anyInclude
				processRoot := includeRoot || !anyInclude

				mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
				resolved, mergeErr := mergePort.MergeFiles(modules, mapping)

				if merged {
					filtered := mergefileconfiguration.ResolvedFileConfiguration{}
					if processUser {
						filtered.UserFiles = resolved.UserFiles
					}
					if processRoot {
						filtered.RootFiles = resolved.RootFiles
					}
					reportPort.ReportMergedRepo(strings.Join(repos, ", "), filtered, mergeErr)
				}

				if diff {
					if processUser {
						reportPort.ReportDiff(strings.Join(repos, ", "), diffPort.CalculateDiffForUserFiles(resolved.UserFiles), mergeErr)
					}
					if processRoot {
						reportPort.ReportDiffRoot(strings.Join(repos, ", "), diffPort.CalculateDiffForRootFiles(resolved.RootFiles), mergeErr)
					}
				}

				if includeProcess {
					processCmd := processhandlers.ProcessHandlersCommand{
						Handlers:     mergedHandlers,
						Declarations: mergedDeclarations,
						DryRun:       true,
					}
					pre := processPort.ProcessPreFileHandlers(processCmd)
					reportPort.ReportProcess(strings.Join(repos, ", "), pre.Decisions, nil)
					post := processPort.ProcessPostFileHandlers(processCmd)
					reportPort.ReportProcess(strings.Join(repos, ", "), post.Decisions, nil)
				}
				return nil
			}

			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				reportPort.ReportRepo(repo, result.Modules)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().BoolVar(&merged, "merged", false, "merge file configuration and report resolved files and errors")
	cmd.Flags().BoolVar(&diff, "diff", false, "calculate and report diff for resolved files")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for merged file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", defaultOutputRoot, "root directory used for merged file mapping; defaults to /")
	cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")
	cmd.Flags().BoolVar(&includeProcess, "include-process", false, "report process handler commands (dry-run) when any include flag is set")

	return cmd
}

func normalizeList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

func splitCommaSeparated(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeList([]string{value})
}
```

### `main.go` — wiring additions
```go
import (
	// ...
	processrunneradapter "syncit/adapter/processRunner"
	processhandlers "syncit/coremodule/processHandlers"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()
	fileSystemPort := filesystemadapter.New()
	processRunnerPort := processrunneradapter.New()

	mergeUseCase := mergefileconfiguration.New()
	diffUseCase := difffileconfiguration.New(fileSystemPort)
	syncUseCase := syncfileconfiguration.New(fileSystemPort)
	processUseCase := processhandlers.New(processRunnerPort)

	rootCmd := &cobra.Command{ /* unchanged */ }
	rootCmd.AddCommand(reportcmd.New(declarationPort, mergeUseCase, diffUseCase, processUseCase, reportPort))
	rootCmd.AddCommand(synccmd.New(declarationPort, mergeUseCase, syncUseCase, processUseCase, reportPort))

	// ... unchanged
}
```

---

## Behaviour summary

| Flag state | user files | root files | process handlers |
|---|---|---|---|
| neither | ✓ | ✓ | ✓ |
| `--include-user` | ✓ | ✗ | ✗ |
| `--include-root` | ✗ | ✓ | ✗ |
| `--include-process` | ✗ | ✗ | ✓ |
| any combination | per flag | per flag | per flag |

- **`sync`** runs: `pacman` → `aur` → user links → root copies → `systemd` → `usersystemd`.
- **`sync --dry-run`** prints handler commands (file ops unchanged).
- **`report --include-process`** prints a dry-run table of would-be handler commands (values column shows the unique list).
- Placeholders resolve from aggregated `Declarations` values across all repos/modules, deduplicated and sorted for determinism.
- `handlers.yaml` (missing → empty) is read per repo; later repos override earlier for the same tool.
- No `diff` support for process handlers (per spec); file diff is unchanged.
- Core `coremodule/processHandlers` has no `os`/`exec` imports — only the `processRunner` port; adapter is the sole OS caller.
