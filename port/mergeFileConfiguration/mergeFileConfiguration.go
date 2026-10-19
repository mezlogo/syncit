package mergefileconfiguration

import (
	"errors"
	"fmt"

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

// Error codes returned (wrapped) by MergeFiles.
var (
	ErrMultipleDeclarations                    = errors.New("MULTIPLE_DECLARATIONS")
	ErrOneDeclaredDirOtherDeclaredFileConflict = errors.New("ONE_DECLARED_DIR_OTHER_DECLARED_FILE_CONFLICT")
)

// MultipleDeclarationsError reports the same resolved target declared more than once.
type MultipleDeclarationsError struct {
	// OverwriteAbsoluteFilePath is the later declaration that would overwrite the existing declaration.
	OverwriteAbsoluteFilePath string
	// AlreadyDeclaredAbsoluteFilePath is the declaration that already owns the resolved target.
	AlreadyDeclaredAbsoluteFilePath string
	// ResolvedTargetAbsoluteFilePath is the absolute host path both declarations resolve to.
	ResolvedTargetAbsoluteFilePath string
}

func (e *MultipleDeclarationsError) Error() string {
	return fmt.Sprintf(
		"%s: overwrite=%q alreadyDeclared=%q resolvedTarget=%q",
		ErrMultipleDeclarations,
		e.OverwriteAbsoluteFilePath,
		e.AlreadyDeclaredAbsoluteFilePath,
		e.ResolvedTargetAbsoluteFilePath,
	)
}

func (e *MultipleDeclarationsError) Unwrap() error {
	return ErrMultipleDeclarations
}

// DeclaredDirFileConflictError reports one declaration resolving to a directory and another to a file.
type DeclaredDirFileConflictError struct {
	// OverwriteAbsoluteFilePath is the declaration that conflicts with the existing declaration.
	OverwriteAbsoluteFilePath string
	// AlreadyDeclaredAbsoluteFilePath is the declaration that already owns the conflicting path.
	AlreadyDeclaredAbsoluteFilePath string
	// ResolvedTargetAbsoluteFilePath is the absolute host path where the directory/file conflict occurs.
	ResolvedTargetAbsoluteFilePath string
}

func (e *DeclaredDirFileConflictError) Error() string {
	return fmt.Sprintf(
		"%s: overwrite=%q alreadyDeclared=%q resolvedTarget=%q",
		ErrOneDeclaredDirOtherDeclaredFileConflict,
		e.OverwriteAbsoluteFilePath,
		e.AlreadyDeclaredAbsoluteFilePath,
		e.ResolvedTargetAbsoluteFilePath,
	)
}

func (e *DeclaredDirFileConflictError) Unwrap() error {
	return ErrOneDeclaredDirOtherDeclaredFileConflict
}
