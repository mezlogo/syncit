package repoReport

import (
	"sort"
	"strings"

	"github.com/pterm/pterm"

	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergeport "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"

	processhandlers "syncit/port/processHandlers"
	syncfileconfiguration "syncit/port/syncFileConfiguration"

	port "syncit/port/repoReport"
)

// Adapter renders repository declarations using pterm.
type Adapter struct{}

// New creates a pterm-backed repository report adapter.
func New() *Adapter {
	return &Adapter{}
}

var _ port.RepoReportUseCase = (*Adapter)(nil)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// ReportRepo prints a repository declaration using pterm tables and sections.
func (a *Adapter) ReportRepo(repoPath string, modules []repoao.RepositoryDeclarationModule) {
	pterm.DefaultSection.WithLevel(1).Println("repository: " + repoPath)

	if len(modules) == 0 {
		pterm.Info.Println("modules: []")
		return
	}

	for _, module := range modules {
		pterm.DefaultSection.WithLevel(2).Println("module: " + module.ModuleName)

		_ = pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
			{"field", "value"},
			{"path", module.ModuleAbsoulutePath},
		}).Render()

		if len(module.Declarations) == 0 {
			pterm.Println("declarations: {}")
		} else {
			keys := make([]string, 0, len(module.Declarations))
			for key := range module.Declarations {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			rows := pterm.TableData{{"declaration", "values"}}
			for _, key := range keys {
				rows = append(rows, []string{key, strings.Join(module.Declarations[key], ", ")})
			}
			_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
		}

		if len(module.RelativeFilePaths) == 0 {
			pterm.Println("files: []")
		} else {
			rows := pterm.TableData{{"file"}}
			for _, file := range module.RelativeFilePaths {
				rows = append(rows, []string{file})
			}
			_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
		}
	}
}

// ReportDeclarations reports aggregated tool declarations across all modules.
func (a *Adapter) ReportDeclarations(repoPath string, declarations map[string][]string) {
	pterm.DefaultSection.WithLevel(1).Println("declarations: " + repoPath)

	if len(declarations) == 0 {
		pterm.Println("{}")
		return
	}

	keys := make([]string, 0, len(declarations))
	for key := range declarations {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := pterm.TableData{{"declaration", "values"}}
	for _, key := range keys {
		rows = append(rows, []string{key, strings.Join(declarations[key], ", ")})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

// ReportMergedRepo prints resolved files and the first merge error.
func (a *Adapter) ReportMergedRepo(repoPath string, resolved mergeport.ResolvedFileConfiguration, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("merged repository: " + repoPath)

	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}

	reportMergedFiles("user files", resolved.UserFiles)
	reportMergedFiles("root files", resolved.RootFiles)
}

// ReportDiff prints colorized diff decisions for resolved user files.
func (a *Adapter) ReportDiff(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("diff repository: " + repoPath)

	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}

	if len(decisions) == 0 {
		pterm.Println("[]")
		return
	}

	rows := pterm.TableData{{"decision", "declared", "target"}}
	for _, decision := range decisions {
		rows = append(rows, []string{
			colorizeDiffCode(decision.Code),
			decision.ResolvedFile.AbsoluteModuleFilePath,
			decision.ResolvedFile.AbsoluteHostFilePath,
		})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

func (a *Adapter) ReportDiffRoot(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("diff repository (root): " + repoPath)
	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}
	if len(decisions) == 0 {
		pterm.Println("[]")
		return
	}
	rows := pterm.TableData{{"decision", "declared", "target"}}
	for _, decision := range decisions {
		rows = append(rows, []string{
			colorizeDiffCode(decision.Code),
			decision.ResolvedFile.AbsoluteModuleFilePath,
			decision.ResolvedFile.AbsoluteHostFilePath,
		})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

func (a *Adapter) ReportSyncRoot(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("sync repository (root): " + repoPath)
	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}
	if len(decisions) == 0 {
		pterm.Println("[]")
		return
	}
	rows := pterm.TableData{{"decision", "declared", "target", "error"}}
	for _, d := range decisions {
		errStr := ""
		if d.Err != nil {
			errStr = d.Err.Error()
		}
		rows = append(rows, []string{
			colorizeSyncCode(d.Code),
			d.ResolvedFile.AbsoluteModuleFilePath,
			d.ResolvedFile.AbsoluteHostFilePath,
			errStr,
		})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

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

func colorizeDiffCode(code difffileconfiguration.DiffFileDecisionCode) string {
	switch code {
	case difffileconfiguration.DiffFileDecisionCodeTargetFileEqualToDeclared:
		return colorGreen + string(code) + colorReset
	case difffileconfiguration.DiffFileDecisionCodeTargetFileNotFound:
		return colorYellow + string(code) + colorReset
	case difffileconfiguration.DiffFileDecisionCodeTargetFileIsDirectory:
		return colorRed + string(code) + colorReset
	default:
		return colorCyan + string(code) + colorReset
	}
}

func reportMergedFiles(title string, files []mergeport.ResolvedFile) {
	pterm.DefaultSection.WithLevel(2).Println(title)
	if len(files) == 0 {
		pterm.Println("[]")
		return
	}
	rows := pterm.TableData{{"source", "target"}}
	for _, file := range files {
		rows = append(rows, []string{file.AbsoluteModuleFilePath, file.AbsoluteHostFilePath})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

// ReportSync prints a colorized table of per-file sync decisions.
func (a *Adapter) ReportSync(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("sync repository: " + repoPath)

	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}
	if len(decisions) == 0 {
		pterm.Println("[]")
		return
	}

	rows := pterm.TableData{{"decision", "declared", "target", "error"}}
	for _, d := range decisions {
		errStr := ""
		if d.Err != nil {
			errStr = d.Err.Error()
		}
		rows = append(rows, []string{
			colorizeSyncCode(d.Code),
			d.ResolvedFile.AbsoluteModuleFilePath,
			d.ResolvedFile.AbsoluteHostFilePath,
			errStr,
		})
	}
	_ = pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
}

func colorizeSyncCode(code syncfileconfiguration.SyncFileDecisionCode) string {
	switch code {
	case syncfileconfiguration.SyncFileDecisionCodeLinked,
		syncfileconfiguration.SyncFileDecisionCodeCopied,
		syncfileconfiguration.SyncFileDecisionCodeRecreated:

		return colorGreen + string(code) + colorReset
	case syncfileconfiguration.SyncFileDecisionCodeSkippedUpToDate:
		return colorCyan + string(code) + colorReset
	case syncfileconfiguration.SyncFileDecisionCodeSkippedExisting, syncfileconfiguration.SyncFileDecisionCodeSkippedDirectory:
		return colorYellow + string(code) + colorReset
	case syncfileconfiguration.SyncFileDecisionCodeFailed:
		return colorRed + string(code) + colorReset
	}
	return string(code)
}
