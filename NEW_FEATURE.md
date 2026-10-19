# Sync Command — First Milestone (User Config Only)

Interpretation used for flags (safe-by-default):
- **Default**: create link if target missing, otherwise skip.
- **`--overwrite`**: recreate target when it exists but is not an up‑to‑date link (wrong link, regular file).
- **`--reset`**: recreate the link even when it already points to the declared file (force fresh).
- **Directories** at target are *never* removed → reported as `SKIPPED_DIRECTORY`.

---

## 1. `port/fileSystem/fileSystem.go` (extend)

```go
package filesystem

type FileSystemPort interface {
	CalculateStatForUserFiles(userAbsouteFilePaths []string) []FileSystemStat
	CalculateStatForRootFiles(rootAbsouteFilePaths []string) []FileSystemStat
	GetLinkForUserFile(absoluteFilePath string) (string, error)
	GetFileSizeInBytesForRootFile(absoluteFilePath string) (int64, error)
	ReadRawFileContent(absoluteFilePath string) ([]byte, error)

	// EnsureDirRecursive creates the directory and any missing parents.
	EnsureDirRecursive(absoluteDirPath string) error

	// CreateUserSymbolicLink creates a symlink at absoluteLinkPath pointing to absoluteTargetPath.
	CreateUserSymbolicLink(absoluteTargetPath, absoluteLinkPath string) error

	// RemoveUserFile removes a file or symlink at absoluteFilePath. Refuses to remove directories.
	RemoveUserFile(absoluteFilePath string) error
}

type FileSystemStatCode string
const (
	FileSystemStatCodeFileNotExist FileSystemStatCode = "FILE_NOT_EXIST"
	FileSystemStatCodeLink         FileSystemStatCode = "LINK"
	FileSystemStatCodeFile         FileSystemStatCode = "FILE"
	FileSystemStatCodeDirectory    FileSystemStatCode = "DIRECTORY"
)
type FileSystemStat struct {
	AbsoluteFilePath string
	Code             FileSystemStatCode
}
```

## 2. `adapter/fileSystem/fileSystem.go` (append)

```go
import (
	"fmt"
	"os"

	port "syncit/port/fileSystem"
)

func (a *Adapter) EnsureDirRecursive(absoluteDirPath string) error {
	return os.MkdirAll(absoluteDirPath, 0o755)
}

func (a *Adapter) CreateUserSymbolicLink(absoluteTargetPath, absoluteLinkPath string) error {
	return os.Symlink(absoluteTargetPath, absoluteLinkPath)
}

func (a *Adapter) RemoveUserFile(absoluteFilePath string) error {
	info, err := os.Lstat(absoluteFilePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to remove directory: %s", absoluteFilePath)
	}
	return os.Remove(absoluteFilePath)
}
```

## 3. `port/syncFileConfiguration/syncFileConfiguration.go` (new)

```go
package syncfileconfiguration

import mergefileconfiguration "syncit/port/mergeFileConfiguration"

// SyncFileConfigurationUseCase applies declared user-space file configuration to the real filesystem.
type SyncFileConfigurationUseCase interface {
	SyncUserFiles(files []mergefileconfiguration.ResolvedFile, options SyncOptions) []SyncFileDecision
}

type SyncOptions struct {
	// Overwrite replaces existing non-link files or links pointing to a different target.
	Overwrite bool
	// Reset recreates the link even when it already points to the declared module file.
	Reset bool
}

type SyncFileDecisionCode string

const (
	SyncFileDecisionCodeLinked           SyncFileDecisionCode = "LINKED"
	SyncFileDecisionCodeRecreated        SyncFileDecisionCode = "RECREATED"
	SyncFileDecisionCodeSkippedUpToDate  SyncFileDecisionCode = "SKIPPED_UP_TO_DATE"
	SyncFileDecisionCodeSkippedExisting  SyncFileDecisionCode = "SKIPPED_EXISTING"
	SyncFileDecisionCodeSkippedDirectory SyncFileDecisionCode = "SKIPPED_DIRECTORY"
	SyncFileDecisionCodeFailed           SyncFileDecisionCode = "FAILED"
)

type SyncFileDecision struct {
	ResolvedFile mergefileconfiguration.ResolvedFile
	Code         SyncFileDecisionCode
	Err          error
}
```

## 4. `coremodule/syncFileConfiguration/syncFileConfiguration.go` (new)

```go
package syncfileconfiguration

import (
	"fmt"
	"path"

	filesystem "syncit/port/fileSystem"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	port "syncit/port/syncFileConfiguration"
)

type UseCase struct {
	fileSystem filesystem.FileSystemPort
}

func New(fileSystem filesystem.FileSystemPort) *UseCase {
	return &UseCase{fileSystem: fileSystem}
}

var _ port.SyncFileConfigurationUseCase = (*UseCase)(nil)

// SyncUserFiles applies every resolved user file as a symlink into the host filesystem.
// Operation continues on per-file errors; each outcome is reported in the returned decisions.
func (u *UseCase) SyncUserFiles(files []mergefileconfiguration.ResolvedFile, options port.SyncOptions) []port.SyncFileDecision {
	decisions := make([]port.SyncFileDecision, len(files))
	for i, file := range files {
		decisions[i] = u.syncOneFile(file, options)
	}
	return decisions
}

func (u *UseCase) syncOneFile(file mergefileconfiguration.ResolvedFile, options port.SyncOptions) port.SyncFileDecision {
	decision := port.SyncFileDecision{ResolvedFile: file}
	stat := u.statFile(file.AbsoluteHostFilePath)

	switch stat.Code {
	case filesystem.FileSystemStatCodeFileNotExist:
		if err := u.createLink(file); err != nil {
			return fail(decision, err)
		}
		decision.Code = port.SyncFileDecisionCodeLinked
		return decision

	case filesystem.FileSystemStatCodeDirectory:
		decision.Code = port.SyncFileDecisionCodeSkippedDirectory
		return decision

	case filesystem.FileSystemStatCodeLink:
		if u.linkPointsToDeclared(file.AbsoluteHostFilePath, file.AbsoluteModuleFilePath) {
			if !options.Reset {
				decision.Code = port.SyncFileDecisionCodeSkippedUpToDate
				return decision
			}
			return u.recreate(file, decision)
		}
		if !options.Overwrite {
			decision.Code = port.SyncFileDecisionCodeSkippedExisting
			return decision
		}
		return u.recreate(file, decision)

	case filesystem.FileSystemStatCodeFile:
		if !options.Overwrite {
			decision.Code = port.SyncFileDecisionCodeSkippedExisting
			return decision
		}
		return u.recreate(file, decision)
	}

	return fail(decision, fmt.Errorf("unknown file stat code: %s", stat.Code))
}

func (u *UseCase) recreate(file mergefileconfiguration.ResolvedFile, decision port.SyncFileDecision) port.SyncFileDecision {
	if err := u.fileSystem.RemoveUserFile(file.AbsoluteHostFilePath); err != nil {
		return fail(decision, err)
	}
	if err := u.createLink(file); err != nil {
		return fail(decision, err)
	}
	decision.Code = port.SyncFileDecisionCodeRecreated
	return decision
}

func (u *UseCase) createLink(file mergefileconfiguration.ResolvedFile) error {
	if err := u.fileSystem.EnsureDirRecursive(path.Dir(file.AbsoluteHostFilePath)); err != nil {
		return err
	}
	return u.fileSystem.CreateUserSymbolicLink(file.AbsoluteModuleFilePath, file.AbsoluteHostFilePath)
}

func (u *UseCase) statFile(p string) filesystem.FileSystemStat {
	stats := u.fileSystem.CalculateStatForUserFiles([]string{p})
	if len(stats) == 0 {
		return filesystem.FileSystemStat{AbsoluteFilePath: p, Code: filesystem.FileSystemStatCodeFileNotExist}
	}
	return stats[0]
}

func (u *UseCase) linkPointsToDeclared(linkPath, declaredPath string) bool {
	target, err := u.fileSystem.GetLinkForUserFile(linkPath)
	if err != nil || target == "" {
		return false
	}
	resolved := target
	if !path.IsAbs(resolved) {
		resolved = path.Join(path.Dir(linkPath), resolved)
	}
	return path.Clean(resolved) == path.Clean(declaredPath)
}

func fail(decision port.SyncFileDecision, err error) port.SyncFileDecision {
	decision.Code = port.SyncFileDecisionCodeFailed
	decision.Err = err
	return decision
}
```

## 5. `port/repoReport/repoReport.go` (extend)

```go
import (
	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"
	syncfileconfiguration "syncit/port/syncFileConfiguration"
)

type RepoReportUseCase interface {
	ReportRepo(repoPath string, modules []repoao.RepositoryDeclarationModule)
	ReportMergedRepo(repoPath string, resolved mergefileconfiguration.ResolvedFileConfiguration, mergeErr error)
	ReportDiff(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error)
	ReportSync(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error)
}
```

## 6. `adapter/repoReport/repoReport.go` (append)

```go
import syncfileconfiguration "syncit/port/syncFileConfiguration"

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
	case syncfileconfiguration.SyncFileDecisionCodeLinked, syncfileconfiguration.SyncFileDecisionCodeRecreated:
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
```

## 7. `cmd/sync/sync.go` (new)

```go
package sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	repositorydeclaration "syncit/port/repoAO"
	reporeport "syncit/port/repoReport"
	syncfileconfiguration "syncit/port/syncFileConfiguration"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func New(
	declarationPort repositorydeclaration.RepositoryDeclarationPort,
	mergePort mergefileconfiguration.MergeFileConfigurationUseCase,
	syncPort syncfileconfiguration.SyncFileConfigurationUseCase,
	reportPort reporeport.RepoReportUseCase,
) *cobra.Command {
	var repositories []string
	var hostName, outputHome, outputRoot string
	var overwrite, reset bool

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

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize user file configuration (links only)",
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
			for _, repo := range repos {
				result, err := declarationPort.ParseRepository(repositorydeclaration.RepositoryDeclarationCommand{
					RepositoryPath: repo,
					HostName:       host,
				})
				if err != nil {
					return fmt.Errorf("parse repository %q: %w", repo, err)
				}
				modules = append(modules, result.Modules...)
			}

			mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping(outputRoot, outputHome)
			resolved, mergeErr := mergePort.MergeFiles(modules, mapping)

			decisions := syncPort.SyncUserFiles(resolved.UserFiles, syncfileconfiguration.SyncOptions{
				Overwrite: overwrite,
				Reset:     reset,
			})

			reportPort.ReportSync(strings.Join(repos, ", "), decisions, mergeErr)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&repositories, "repo", defaultRepositories, "repository directories separated by comma; defaults to $SYNCIT_REPO")
	cmd.Flags().StringVar(&hostName, "host", defaultHostName, "host declaration name; defaults to $SYNCIT_HOST or os hostname")
	cmd.Flags().StringVar(&outputHome, "output-home", defaultOutputHome, "home directory used for file mapping; defaults to $HOME")
	cmd.Flags().StringVar(&outputRoot, "output-root", "/", "root directory used for file mapping; defaults to /")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "recreate existing files or wrong links at the target")
	cmd.Flags().BoolVar(&reset, "reset", false, "recreate the link even when it already points to the declared file")

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

## 8. `main.go` (update)

```go
import (
	// existing + new:
	synccmd "syncit/cmd/sync"
	syncfileconfiguration "syncit/coremodule/syncFileConfiguration"
)

func main() {
	declarationPort := repositorydeclarationadapter.New()
	reportPort := reporeportadapter.New()
	fileSystemPort := filesystemadapter.New()

	mergeUseCase := mergefileconfiguration.New()
	diffUseCase := difffileconfiguration.New(fileSystemPort)
	syncUseCase := syncfileconfiguration.New(fileSystemPort)

	rootCmd := &cobra.Command{ /* unchanged */ }
	rootCmd.AddCommand(reportcmd.New(declarationPort, mergeUseCase, diffUseCase, reportPort))
	rootCmd.AddCommand(synccmd.New(declarationPort, mergeUseCase, syncUseCase, reportPort))

	// unchanged execute/error handling
}
```

## 9. `coremodule/syncFileConfiguration/syncFileConfiguration_test.go` (new, table-driven)

```go
package syncfileconfiguration

import (
	"errors"
	"testing"

	filesystem "syncit/port/fileSystem"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	port "syncit/port/syncFileConfiguration"
)

type fakeFS struct {
	stats map[string]filesystem.FileSystemStat
	links map[string]string

	createdLinks map[string]string
	removedFiles []string

	errOnCreate error
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		stats:        map[string]filesystem.FileSystemStat{},
		links:        map[string]string{},
		createdLinks: map[string]string{},
	}
}

func (f *fakeFS) CalculateStatForUserFiles(paths []string) []filesystem.FileSystemStat {
	out := make([]filesystem.FileSystemStat, 0, len(paths))
	for _, p := range paths {
		if s, ok := f.stats[p]; ok {
			out = append(out, s)
		} else {
			out = append(out, filesystem.FileSystemStat{AbsoluteFilePath: p, Code: filesystem.FileSystemStatCodeFileNotExist})
		}
	}
	return out
}
func (f *fakeFS) CalculateStatForRootFiles([]string) []filesystem.FileSystemStat { return nil }
func (f *fakeFS) GetLinkForUserFile(p string) (string, error) {
	if t, ok := f.links[p]; ok {
		return t, nil
	}
	return "", errors.New("not a link")
}
func (f *fakeFS) GetFileSizeInBytesForRootFile(string) (int64, error) { return 0, nil }
func (f *fakeFS) ReadRawFileContent(string) ([]byte, error)           { return nil, nil }

func (f *fakeFS) EnsureDirRecursive(string) error { return nil }
func (f *fakeFS) CreateUserSymbolicLink(target, linkPath string) error {
	if f.errOnCreate != nil {
		return f.errOnCreate
	}
	f.createdLinks[linkPath] = target
	return nil
}
func (f *fakeFS) RemoveUserFile(p string) error {
	f.removedFiles = append(f.removedFiles, p)
	return nil
}

var _ filesystem.FileSystemPort = (*fakeFS)(nil)

func TestUseCase_SyncUserFiles(t *testing.T) {
	target := "/home/u/.config/git/config"
	declared := "/repo/modules/a/config/git/config"
	mkFile := func(code filesystem.FileSystemStatCode) mergefileconfiguration.ResolvedFile {
		return mergefileconfiguration.ResolvedFile{AbsoluteModuleFilePath: declared, AbsoluteHostFilePath: target}
	}

	tests := []struct {
		name     string
		file     mergefileconfiguration.ResolvedFile
		options  port.SyncOptions
		setup    func(*fakeFS)
		wantCode port.SyncFileDecisionCode
		wantErr  bool
	}{
		{
			name: "creates link when target missing",
			file: mkFile(filesystem.FileSystemStatCodeFileNotExist),
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFileNotExist}
			},
			wantCode: port.SyncFileDecisionCodeLinked,
		},
		{
			name: "skips up to date link",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = declared
			},
			wantCode: port.SyncFileDecisionCodeSkippedUpToDate,
		},
		{
			name:     "recreates up to date link with reset",
			options:  port.SyncOptions{Reset: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = declared
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name: "skips existing regular file without overwrite",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFile}
			},
			wantCode: port.SyncFileDecisionCodeSkippedExisting,
		},
		{
			name:     "recreates existing regular file with overwrite",
			options:  port.SyncOptions{Overwrite: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFile}
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name: "skips wrong link without overwrite",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = "/elsewhere"
			},
			wantCode: port.SyncFileDecisionCodeSkippedExisting,
		},
		{
			name:     "recreates wrong link with overwrite",
			options:  port.SyncOptions{Overwrite: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = "/elsewhere"
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name:    "never replaces a directory",
			options: port.SyncOptions{Overwrite: true, Reset: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeDirectory}
			},
			wantCode: port.SyncFileDecisionCodeSkippedDirectory,
		},
		{
			name: "fails on create link error",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFileNotExist}
				f.errOnCreate = errors.New("boom")
			},
			wantCode: port.SyncFileDecisionCodeFailed,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeFS()
			if tt.setup != nil {
				tt.setup(f)
			}
			file := tt.file
			if file.AbsoluteHostFilePath == "" {
				file = mkFile(filesystem.FileSystemStatCodeFileNotExist)
			}
			got := New(f).SyncUserFiles([]mergefileconfiguration.ResolvedFile{file}, tt.options)
			if len(got) != 1 {
				t.Fatalf("decisions length: got %d, want 1", len(got))
			}
			if got[0].Code != tt.wantCode {
				t.Errorf("code: got %q, want %q", got[0].Code, tt.wantCode)
			}
			if (got[0].Err != nil) != tt.wantErr {
				t.Errorf("error status: got %v, wantErr %v", got[0].Err, tt.wantErr)
			}
		})
	}
}
```

---

## Notes

- Ports stay isolated: `coremodule/syncFileConfiguration` only imports `port/` packages; `adapter/fileSystem` is the only place touching `os.Symlink` / `os.MkdirAll` / `os.Remove`.
- Cross-module deps flow through `port/syncFileConfiguration` and `port/fileSystem`; no `adapter` imports from `cmd` or `coremodule`.
- `--overwrite` and `--reset` are intentionally orthogonal: `--overwrite` handles *wrong* target types, `--reset` refreshes an *already-correct* link.
- Directories are never removed, guarding against accidental data loss.
- Tests follow the table-driven convention from `TESTING.md` with a manual fake for the FS port (used because no gomock generated mocks exist in the repo yet; migrate to `go.uber.org/mock` later per TESTING.md).
