package mergefileconfiguration

import (
	"sort"
	"strings"

	mergeport "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

type UseCase struct{}

func New() *UseCase {
	return &UseCase{}
}

var _ mergeport.MergeFileConfigurationUseCase = (*UseCase)(nil)

type resolvedEntry struct {
	file   mergeport.ResolvedFile
	isUser bool
}

// MergeFiles resolves every declared module file to an absolute host target.
// Duplicate host targets use Last Write Wins, while the first detected conflict
// is returned as an additional signal for the caller.
func (u *UseCase) MergeFiles(modules []repoao.RepositoryDeclarationModule, fileConfigurationMapping toolsconfiguraion.ResolvedFileConfigurationMapping) (mergeport.ResolvedFileConfiguration, error) {
	resolvedByTarget := make(map[string]resolvedEntry)
	var firstErr error

	for _, module := range modules {
		for _, relativeFilePath := range module.RelativeFilePaths {
			resolvedFile, isUser, err := resolveFile(module, relativeFilePath, fileConfigurationMapping)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}

			if existing, ok := resolvedByTarget[resolvedFile.AbsoluteHostFilePath]; ok {
				if firstErr == nil {
					firstErr = &mergeport.MultipleDeclarationsError{
						OverwriteAbsoluteFilePath:       resolvedFile.AbsoluteModuleFilePath,
						AlreadyDeclaredAbsoluteFilePath: existing.file.AbsoluteModuleFilePath,
						ResolvedTargetAbsoluteFilePath:  resolvedFile.AbsoluteHostFilePath,
					}
				}
			}

			if existing, conflictPath, ok := findDirFileConflict(resolvedByTarget, resolvedFile.AbsoluteHostFilePath); ok {
				if firstErr == nil {
					firstErr = &mergeport.DeclaredDirFileConflictError{
						OverwriteAbsoluteFilePath:       resolvedFile.AbsoluteModuleFilePath,
						AlreadyDeclaredAbsoluteFilePath: existing.file.AbsoluteModuleFilePath,
						ResolvedTargetAbsoluteFilePath:  conflictPath,
					}
				}
			}

			resolvedByTarget[resolvedFile.AbsoluteHostFilePath] = resolvedEntry{
				file:   resolvedFile,
				isUser: isUser,
			}
		}
	}

	userFiles := make([]mergeport.ResolvedFile, 0, len(resolvedByTarget))
	rootFiles := make([]mergeport.ResolvedFile, 0, len(resolvedByTarget))
	for _, entry := range resolvedByTarget {
		if entry.isUser {
			userFiles = append(userFiles, entry.file)
		} else {
			rootFiles = append(rootFiles, entry.file)
		}
	}

	sortResolvedFiles(userFiles)
	sortResolvedFiles(rootFiles)

	return mergeport.ResolvedFileConfiguration{
		UserFiles: userFiles,
		RootFiles: rootFiles,
	}, firstErr
}

func findDirFileConflict(seen map[string]resolvedEntry, target string) (resolvedEntry, string, bool) {
	for existingTarget, existing := range seen {
		if isPathAncestor(target, existingTarget) {
			return existing, target, true
		}
		if isPathAncestor(existingTarget, target) {
			return existing, existingTarget, true
		}
	}
	return resolvedEntry{}, "", false
}

func isPathAncestor(parent, child string) bool {
	if parent == child {
		return false
	}
	if parent == "/" {
		return strings.HasPrefix(child, "/")
	}
	return strings.HasPrefix(child, parent+"/")
}

func sortResolvedFiles(files []mergeport.ResolvedFile) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].AbsoluteHostFilePath < files[j].AbsoluteHostFilePath
	})
}
