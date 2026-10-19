package mergefileconfiguration

import (
	"syncit/port/repoAO"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

// This module is a core domain logic for merging in memory file pathes

type MergeFileConfigurationUseCase interface {
	// Takes as input all declared modules
	// Always returns a list of unique files with Last Write Wins conflict resolution stategy
	// In this case we treat error like additional signal for better call site decition making: ignore, try to fix or abort
	// Merge relative files by
	MergeFiles(modules []repoAO.RepositoryDeclarationModule, fileConfigurationMapping toolsconfiguraion.ResolvedFileConfigurationMapping) (ResolvedFileConfiguration, error)
}

type ResolvedFile struct {
	// This is a path to file inside repository
	absoluteModuleFilePath string
	// This is a path to real file on computer
	absoluteHostFilePath string
}

type ResolvedFileConfiguration struct {
	// home related files
	userFiles []ResolvedFile
	// root related files
	rootFiles []ResolvedFile
}

/*
TODO: write here error hierarchy with example in comments and following fields:
- overwriteAbsoluteFilePath: this is WHY conflict is possible
- alreadyDeclaredAbsoluteFilePath:
- resolvedTargetAbsoluteFilePath

declared files WITHOUT fileConfigurationMapping
*/
