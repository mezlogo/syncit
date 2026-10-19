package difffileconfiguration

import (
	"bytes"
	"path"

	port "syncit/port/diffFileConfiguration"
	filesystem "syncit/port/fileSystem"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
)

// UseCase calculates diff decisions for declared user-space files.
type UseCase struct {
	fileSystem filesystem.FileSystemPort
}

// New creates a diff file configuration use case backed by the file system port.
func New(fileSystem filesystem.FileSystemPort) *UseCase {
	return &UseCase{fileSystem: fileSystem}
}

var _ port.DiffFileConfigurationUseCase = (*UseCase)(nil)

// CalculateDiffForUserFiles builds a decision for every declared user-space file.
func (u *UseCase) CalculateDiffForUserFiles(userConfigurationFiles []mergefileconfiguration.ResolvedFile) []port.DiffFileDecision {
	decisions := make([]port.DiffFileDecision, len(userConfigurationFiles))
	if len(userConfigurationFiles) == 0 {
		return decisions
	}

	targetPaths := make([]string, len(userConfigurationFiles))
	for i, file := range userConfigurationFiles {
		targetPaths[i] = file.AbsoluteHostFilePath
	}

	stats := u.fileSystem.CalculateStatForUserFiles(targetPaths)
	statByPath := make(map[string]filesystem.FileSystemStat, len(stats))
	for _, stat := range stats {
		statByPath[stat.AbsoluteFilePath] = stat
	}

	for i, file := range userConfigurationFiles {
		decision := port.DiffFileDecision{ResolvedFile: file}

		stat, ok := statByPath[file.AbsoluteHostFilePath]
		if !ok {
			decision.Code = port.DiffFileDecisionCodeTargetFileNotFound
			decisions[i] = decision
			continue
		}

		switch stat.Code {
		case filesystem.FileSystemStatCodeFileNotExist:
			decision.Code = port.DiffFileDecisionCodeTargetFileNotFound
		case filesystem.FileSystemStatCodeDirectory:
			decision.Code = port.DiffFileDecisionCodeTargetFileIsDirectory
		case filesystem.FileSystemStatCodeLink:
			if u.linkPointsToDeclared(file.AbsoluteHostFilePath, file.AbsoluteModuleFilePath) {
				decision.Code = port.DiffFileDecisionCodeTargetFileEqualToDeclared
			} else {
				decision.Code = port.DiffFileDecisionCodeTargetFileNotEqualToDeclared
			}
		default:
			decision.Code = port.DiffFileDecisionCodeTargetFileNotEqualToDeclared
		}

		decisions[i] = decision
	}

	return decisions
}

func (u *UseCase) linkPointsToDeclared(linkPath string, declaredPath string) bool {
	linkTarget, err := u.fileSystem.GetLinkForUserFile(linkPath)
	if err != nil || linkTarget == "" {
		return false
	}

	resolvedTarget := linkTarget
	if !path.IsAbs(resolvedTarget) {
		resolvedTarget = path.Join(path.Dir(linkPath), resolvedTarget)
	}

	return path.Clean(resolvedTarget) == path.Clean(declaredPath)
}

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
