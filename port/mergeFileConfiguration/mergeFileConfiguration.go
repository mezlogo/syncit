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
	AbsoluteModuleFilePath string
	// This is a path to real file on computer
	AbsoluteHostFilePath string
}

type ResolvedFileConfiguration struct {
	// home related files
	UserFiles []ResolvedFile
	// root related files
	RootFiles []ResolvedFile
}

// Error codes returned (wrapped) by MergeFiles.
var (
	ErrMultipleDeclarations                           = errors.New("MULTIPLE_DECLARATIONS")
	ErrOneDeclaredDirOtherDeclaredFileConflict        = errors.New("ONE_DECLARED_DIR_OTHER_DECLARED_FILE_CONFLICT")
	ErrDeclaredFileNotFoundInFileConfigurationMapping = errors.New("DECLARED_FILE_NOT_FOUND_IN_FILE_CONFIGURATION_MAPPING")
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
		"%s:\nFor given target file %s\nAlready declared file: %s\nConflict with: %s",
		ErrMultipleDeclarations,
		e.ResolvedTargetAbsoluteFilePath,
		e.AlreadyDeclaredAbsoluteFilePath,
		e.OverwriteAbsoluteFilePath,
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
		"%s:\nFor given target file %s\nAlready declared file: %s\nConflict with: %s",
		ErrOneDeclaredDirOtherDeclaredFileConflict,
		e.ResolvedTargetAbsoluteFilePath,
		e.AlreadyDeclaredAbsoluteFilePath,
		e.OverwriteAbsoluteFilePath,
	)
}

func (e *DeclaredDirFileConflictError) Unwrap() error {
	return ErrOneDeclaredDirOtherDeclaredFileConflict
}

// DeclaredFileNotFoundInFileConfigurationMappingError reports a module file whose top-level directory
// is not present in the resolved file configuration mapping.
type DeclaredFileNotFoundInFileConfigurationMappingError struct {
	// AbsoluteModuleFilePath is the absolute path to the declared file inside the repository.
	AbsoluteModuleFilePath string
	// RelativeFilePath is the file path relative to the module root.
	RelativeFilePath string
}

func (e *DeclaredFileNotFoundInFileConfigurationMappingError) Error() string {
	return fmt.Sprintf(
		"%s:\nModule file: %s\nRelative file: %s",
		ErrDeclaredFileNotFoundInFileConfigurationMapping,
		e.AbsoluteModuleFilePath,
		e.RelativeFilePath,
	)
}

func (e *DeclaredFileNotFoundInFileConfigurationMappingError) Unwrap() error {
	return ErrDeclaredFileNotFoundInFileConfigurationMapping
}
