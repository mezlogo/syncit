package mergefileconfiguration

import (
	"path"

	mergeport "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func resolveFile(module repoao.RepositoryDeclarationModule, relativeFilePath string, fileConfigurationMapping toolsconfiguraion.ResolvedFileConfigurationMapping) (mergeport.ResolvedFile, bool, error) {
	cleaned, topLevel, rest, err := normalizeRelativeFilePath(relativeFilePath)
	if err != nil {
		return mergeport.ResolvedFile{}, false, err
	}

	targetDir, ok := fileConfigurationMapping.Lookup(topLevel)
	if !ok {
		return mergeport.ResolvedFile{}, false, &mergeport.DeclaredFileNotFoundInFileConfigurationMappingError{
			AbsoluteModuleFilePath: path.Join(module.ModuleAbsoulutePath, cleaned),
			RelativeFilePath:       relativeFilePath,
		}
	}

	return mergeport.ResolvedFile{
		AbsoluteModuleFilePath: path.Join(module.ModuleAbsoulutePath, cleaned),
		AbsoluteHostFilePath:   path.Join(targetDir, rest),
	}, isUserConfiguration(topLevel), nil
}

func isUserConfiguration(topLevel string) bool {
	switch topLevel {
	case "config", "local", "ssh", "home":
		return true
	default:
		return false
	}
}
