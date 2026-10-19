package repoReport

import (
	"sort"
	"strings"

	"github.com/pterm/pterm"

	mergeport "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"
	port "syncit/port/repoReport"
)

// Adapter renders repository declarations using pterm.
type Adapter struct{}

// New creates a pterm-backed repository report adapter.
func New() *Adapter {
	return &Adapter{}
}

var _ port.RepoReportUseCase = (*Adapter)(nil)

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

// ReportMergedRepo prints resolved files and the first merge error.
func (a *Adapter) ReportMergedRepo(repoPath string, resolved mergeport.ResolvedFileConfiguration, mergeErr error) {
	pterm.DefaultSection.WithLevel(1).Println("merged repository: " + repoPath)

	if mergeErr != nil {
		pterm.Error.Println(mergeErr)
	}

	reportMergedFiles("user files", resolved.UserFiles)
	reportMergedFiles("root files", resolved.RootFiles)
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
