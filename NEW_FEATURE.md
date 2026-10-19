# Implementation Plan

## 1. Port: FileSystem — add root operations

**`port/fileSystem/fileSystem.go`** (add to interface):

```go
// EnsureRootDirRecursive creates the directory tree using elevated privileges when required.
EnsureRootDirRecursive(absoluteDirPath string) error

// CopyFileToRoot copies sourceAbsoluteFilePath to targetAbsoluteFilePath using elevated privileges when required.
CopyFileToRoot(sourceAbsoluteFilePath, targetAbsoluteFilePath string) error

// RemoveRootFile removes a file at absoluteFilePath using elevated privileges when required.
RemoveRootFile(absoluteFilePath string) error
```

## 2. Adapter: FileSystem — sudo with fallback

**`adapter/fileSystem/fileSystem.go`** (additions):

```go
import (
    "os/exec"
    "strconv"
    "strings"
)

// GetFileSizeInBytesForRootFile — try as user, escalate on permission error
func (a *Adapter) GetFileSizeInBytesForRootFile(absoluteFilePath string) (int64, error) {
    info, err := os.Stat(absoluteFilePath)
    if err == nil {
        return info.Size(), nil
    }
    if !os.IsPermission(err) {
        return 0, err
    }
    out, err := runRootCommandOutput("stat", "-c", "%s", absoluteFilePath)
    if err != nil {
        return 0, err
    }
    return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

func (a *Adapter) ReadRawFileContent(absoluteFilePath string) ([]byte, error) {
    data, err := os.ReadFile(absoluteFilePath)
    if err == nil {
        return data, nil
    }
    if !os.IsPermission(err) {
        return nil, err
    }
    return runRootCommandOutput("cat", absoluteFilePath)
}

func (a *Adapter) EnsureRootDirRecursive(absoluteDirPath string) error {
    return runRootCommand("mkdir", "-p", absoluteDirPath)
}

func (a *Adapter) CopyFileToRoot(sourceAbsoluteFilePath, targetAbsoluteFilePath string) error {
    return runRootCommand("cp", sourceAbsoluteFilePath, targetAbsoluteFilePath)
}

func (a *Adapter) RemoveRootFile(absoluteFilePath string) error {
    return runRootCommand("rm", "-f", absoluteFilePath)
}

func runRootCommand(name string, args ...string) error {
    cmd := buildRootCommand(name, args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func runRootCommandOutput(name string, args ...string) ([]byte, error) {
    cmd := buildRootCommand(name, args...)
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr
    return cmd.Output()
}

func buildRootCommand(name string, args ...string) *exec.Cmd {
    if os.Geteuid() == 0 {
        return exec.Command(name, args...)
    }
    return exec.Command("sudo", append([]string{name}, args...)...)
}
```

## 3. Port: Diff — root diff method

**`port/diffFileConfiguration/diffFileConfiguration.go`** (add to interface):

```go
// CalculateDiffForRootFiles builds a decision for every declared root-space file.
// Equality uses size first, then byte comparison.
CalculateDiffForRootFiles(rootConfigurationFiles []mergefileconfiguration.ResolvedFile) []DiffFileDecision
```

## 4. Core: Diff — implement root diff

**`coremodule/diffFileConfiguration/diffFileConfiguration.go`** (additions, `bytes` import):

```go
func (u *UseCase) CalculateDiffForRootFiles(rootConfigurationFiles []mergefileconfiguration.ResolvedFile) []port.DiffFileDecision {
    decisions := make([]port.DiffFileDecision, len(rootConfigurationFiles))
    if len(rootConfigurationFiles) == 0 {
        return decisions
    }

    targetPaths := make([]string, len(rootConfigurationFiles))
    for i, file := range rootConfigurationFiles {
        targetPaths[i] = file.AbsoluteHostFilePath
    }

    stats := u.fileSystem.CalculateStatForRootFiles(targetPaths)
    statByPath := make(map[string]filesystem.FileSystemStat, len(stats))
    for _, stat := range stats {
        statByPath[stat.AbsoluteFilePath] = stat
    }

    for i, file := range rootConfigurationFiles {
        decision := port.DiffFileDecision{ResolvedFile: file}
        stat, ok := statByPath[file.AbsoluteHostFilePath]
        if !ok || stat.Code == filesystem.FileSystemStatCodeFileNotExist {
            decision.Code = port.DiffFileDecisionCodeTargetFileNotFound
            decisions[i] = decision
            continue
        }
        switch stat.Code {
        case filesystem.FileSystemStatCodeDirectory:
            decision.Code = port.DiffFileDecisionCodeTargetFileIsDirectory
        case filesystem.FileSystemStatCodeFile:
            equal, err := u.rootFileEqualsDeclared(file.AbsoluteHostFilePath, file.AbsoluteModuleFilePath)
            if err != nil || !equal {
                decision.Code = port.DiffFileDecisionCodeTargetFileNotEqualToDeclared
            } else {
                decision.Code = port.DiffFileDecisionCodeTargetFileEqualToDeclared
            }
        default:
            decision.Code = port.DiffFileDecisionCodeTargetFileNotEqualToDeclared
        }
        decisions[i] = decision
    }
    return decisions
}

func (u *UseCase) rootFileEqualsDeclared(targetPath, declaredPath string) (bool, error) {
    targetSize, err := u.fileSystem.GetFileSizeInBytesForRootFile(targetPath)
    if err != nil {
        return false, err
    }
    declaredSize, err := u.fileSystem.GetFileSizeInBytesForRootFile(declaredPath)
    if err != nil {
        return false, err
    }
    if targetSize != declaredSize {
        return false, nil
    }
    targetContent, err := u.fileSystem.ReadRawFileContent(targetPath)
    if err != nil {
        return false, err
    }
    declaredContent, err := u.fileSystem.ReadRawFileContent(declaredPath)
    if err != nil {
        return false, err
    }
    return bytes.Equal(targetContent, declaredContent), nil
}
```

## 5. Port: Sync — root sync method & new code

**`port/syncFileConfiguration/syncFileConfiguration.go`** (additions):

```go
// SyncRootFiles applies every resolved root file as a copied file using sudo when required.
// Operation continues on per-file errors; each outcome is reported in the returned decisions.
SyncRootFiles(files []mergefileconfiguration.ResolvedFile, options SyncOptions) []SyncFileDecision
```

Add code:
```go
SyncFileDecisionCodeCopied SyncFileDecisionCode = "COPIED"
```

## 6. Core: Sync — implement root sync

**`coremodule/syncFileConfiguration/syncFileConfiguration.go`** (additions, `bytes` import):

```go
func (u *UseCase) SyncRootFiles(files []mergefileconfiguration.ResolvedFile, options port.SyncOptions) []port.SyncFileDecision {
    decisions := make([]port.SyncFileDecision, len(files))
    for i, file := range files {
        decisions[i] = u.syncOneRootFile(file, options)
    }
    return decisions
}

func (u *UseCase) syncOneRootFile(file mergefileconfiguration.ResolvedFile, options port.SyncOptions) port.SyncFileDecision {
    decision := port.SyncFileDecision{ResolvedFile: file}
    stat := u.statRootFile(file.AbsoluteHostFilePath)

    switch stat.Code {
    case filesystem.FileSystemStatCodeFileNotExist:
        if err := u.createRootCopy(file); err != nil {
            return fail(decision, err)
        }
        decision.Code = port.SyncFileDecisionCodeCopied
        return decision
    case filesystem.FileSystemStatCodeDirectory:
        decision.Code = port.SyncFileDecisionCodeSkippedDirectory
        return decision
    case filesystem.FileSystemStatCodeFile:
        equal, err := u.rootFileEqualsDeclared(file.AbsoluteHostFilePath, file.AbsoluteModuleFilePath)
        if err != nil {
            return fail(decision, err)
        }
        if equal && !options.Reset {
            decision.Code = port.SyncFileDecisionCodeSkippedUpToDate
            return decision
        }
        if equal && options.Reset {
            return u.recreateRootCopy(file, decision)
        }
        if !options.Overwrite {
            decision.Code = port.SyncFileDecisionCodeSkippedExisting
            return decision
        }
        return u.recreateRootCopy(file, decision)
    }
    return fail(decision, fmt.Errorf("unknown root stat code: %s", stat.Code))
}

func (u *UseCase) recreateRootCopy(file mergefileconfiguration.ResolvedFile, decision port.SyncFileDecision) port.SyncFileDecision {
    if err := u.fileSystem.RemoveRootFile(file.AbsoluteHostFilePath); err != nil {
        return fail(decision, err)
    }
    if err := u.createRootCopy(file); err != nil {
        return fail(decision, err)
    }
    decision.Code = port.SyncFileDecisionCodeRecreated
    return decision
}

func (u *UseCase) createRootCopy(file mergefileconfiguration.ResolvedFile) error {
    if err := u.fileSystem.EnsureRootDirRecursive(path.Dir(file.AbsoluteHostFilePath)); err != nil {
        return err
    }
    return u.fileSystem.CopyFileToRoot(file.AbsoluteModuleFilePath, file.AbsoluteHostFilePath)
}

func (u *UseCase) statRootFile(p string) filesystem.FileSystemStat {
    stats := u.fileSystem.CalculateStatForRootFiles([]string{p})
    if len(stats) == 0 {
        return filesystem.FileSystemStat{AbsoluteFilePath: p, Code: filesystem.FileSystemStatCodeFileNotExist}
    }
    return stats[0]
}

func (u *UseCase) rootFileEqualsDeclared(targetPath, declaredPath string) (bool, error) {
    targetSize, err := u.fileSystem.GetFileSizeInBytesForRootFile(targetPath)
    if err != nil {
        return false, err
    }
    declaredSize, err := u.fileSystem.GetFileSizeInBytesForRootFile(declaredPath)
    if err != nil {
        return false, err
    }
    if targetSize != declaredSize {
        return false, nil
    }
    targetContent, err := u.fileSystem.ReadRawFileContent(targetPath)
    if err != nil {
        return false, err
    }
    declaredContent, err := u.fileSystem.ReadRawFileContent(declaredPath)
    if err != nil {
        return false, err
    }
    return bytes.Equal(targetContent, declaredContent), nil
}
```

## 7. Port: Report — root report methods

**`port/repoReport/repoReport.go`** (add to interface):

```go
// ReportDiffRoot reports diff decisions for root-space files with colorized decision codes.
ReportDiffRoot(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error)

// ReportSyncRoot reports sync decisions for root-space files with colorized decision codes.
ReportSyncRoot(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error)
```

## 8. Adapter: Report — implement root report methods

**`adapter/repoReport/repoReport.go`** (additions + update `colorizeSyncCode`):

```go
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
```

Update `colorizeSyncCode`:
```go
case syncfileconfiguration.SyncFileDecisionCodeLinked,
    syncfileconfiguration.SyncFileDecisionCodeCopied,
    syncfileconfiguration.SyncFileDecisionCodeRecreated:
    return colorGreen + string(code) + colorReset
```

## 9. CLI: `report` — include flags & root diff

**`cmd/report/report.go`** (additions/change):

```go
var includeUser, includeRoot bool
// ... in flags:
cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")
```

In `RunE` for `merged || diff` branch:

```go
processUser := includeUser || !includeRoot
processRoot := includeRoot || !includeUser

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
```

## 10. CLI: `sync` — include flags & root sync

**`cmd/sync/sync.go`** (replace body):

```go
var includeUser, includeRoot bool
var outputHome, outputRoot string

defaultOutputRoot := "/"

// flags:
cmd.Flags().StringVar(&outputRoot, "output-root", defaultOutputRoot, "root directory used for file mapping; defaults to /")
cmd.Flags().BoolVar(&includeUser, "include-user", false, "process user file configuration when any include flag is set")
cmd.Flags().BoolVar(&includeRoot, "include-root", false, "process root file configuration when any include flag is set")

// RunE:
processUser := includeUser || !includeRoot
processRoot := includeRoot || !includeUser

resolved, mergeErr := mergePort.MergeFiles(modules, mapping)
opts := syncfileconfiguration.SyncOptions{Overwrite: overwrite, Reset: reset}

if processUser {
    reportPort.ReportSync(strings.Join(repos, ", "),
        syncPort.SyncUserFiles(resolved.UserFiles, opts), mergeErr)
}
if processRoot {
    reportPort.ReportSyncRoot(strings.Join(repos, ", "),
        syncPort.SyncRootFiles(resolved.RootFiles, opts), mergeErr)
}
return nil
```

## 11. Test fake — extend `fakeFS`

**`coremodule/syncFileConfiguration/syncFileConfiguration_test.go`** (additions):

```go
func (f *fakeFS) EnsureRootDirRecursive(string) error          { return nil }
func (f *fakeFS) CopyFileToRoot(source, target string) error   { f.copiedLinks[target] = source; return nil }
func (f *fakeFS) RemoveRootFile(p string) error                { f.removedFiles = append(f.removedFiles, p); return nil }
```

(and add `rootSizes map[string]int64`, `rootContents map[string][]byte`, plus implementations for `GetFileSizeInBytesForRootFile`/`ReadRawFileContent` returning from those maps when present.)

---

**Summary of behavior:**

| Flag state | User processed | Root processed |
|---|---|---|
| neither | ✓ | ✓ |
| `--include-user` | ✓ | ✗ |
| `--include-root` | ✗ | ✓ |
| both | ✓ | ✓ |

Root sync uses `sudo mkdir -p` + `sudo cp` (interactive stdin wired), comparison is size-first-then-bytes, and all root reads try unprivileged `os` calls first, escalating to `sudo` only on `os.IsPermission`. Root diff reuses existing diff codes (`TARGET_FILE_EQUAL_TO_DECLARED` = content matches for root).
