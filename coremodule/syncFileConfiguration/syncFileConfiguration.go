package syncfileconfiguration

import (
	"bytes"
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
