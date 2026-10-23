package mergefileconfiguration

import (
	"path"
	"strings"

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

	absoluteHostFilePath := path.Join(targetDir, rest)
	homePath, _ := fileConfigurationMapping.Lookup("home")

	return mergeport.ResolvedFile{
		AbsoluteModuleFilePath: path.Join(module.ModuleAbsoulutePath, cleaned),
		AbsoluteHostFilePath:   absoluteHostFilePath,
	}, isUserConfiguration(homePath, absoluteHostFilePath), nil
}

func isUserConfiguration(homePath, filePath string) bool {
	home := path.Clean(homePath)
	file := path.Clean(filePath)

	if home == "" || home == "." {
		return false
	}
	if file == home {
		return true
	}
	return strings.HasPrefix(file, home+"/")
}
