package mergefileconfiguration

import "syncit/port/repoAO"

// This module is a core domain logic for merging in memory file pathes

type MergeFileConfigurationUseCase interface {
	// Takes as input all declared modules
	// Always returns a list of unique files with Last Write Wins conflict resolution stategy
	MergeFiles(modules []repoAO.RepositoryDeclarationModule)
}
