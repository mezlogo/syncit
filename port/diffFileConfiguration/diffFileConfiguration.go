package difffileconfiguration

import mergefileconfiguration "syncit/port/mergeFileConfiguration"

// This module is a core domain logic for calculating in memory file pathes

// This is the most important part of whole application
// Builds for each file system file and declared file configuration result what to do
type DiffFileConfigurationUseCase interface {
	// CalculateDiffForUserFiles builds a decision for every declared user-space file.
	CalculateDiffForUserFiles(userConfigurationFiles []mergefileconfiguration.ResolvedFile) []DiffFileDecision

	// CalculateDiffForRootFiles builds a decision for every declared root-space file.
	// Equality uses size first, then byte comparison.
	CalculateDiffForRootFiles(rootConfigurationFiles []mergefileconfiguration.ResolvedFile) []DiffFileDecision
}

// DiffFileDecisionCode is the decision code for a single resolved file.
type DiffFileDecisionCode string

const (
	// TARGET_FILE_EQUAL_TO_DECLARED for user space files means it LINKS to declared file
	DiffFileDecisionCodeTargetFileEqualToDeclared DiffFileDecisionCode = "TARGET_FILE_EQUAL_TO_DECLARED"
	// TARGET_FILE_NOT_EQUAL_TO_DECLARED means it could be either a link to other file
	// or just a file
	DiffFileDecisionCodeTargetFileNotEqualToDeclared DiffFileDecisionCode = "TARGET_FILE_NOT_EQUAL_TO_DECLARED"
	// TARGET_FILE_NOT_FOUND means the target file does not exist on the host
	DiffFileDecisionCodeTargetFileNotFound DiffFileDecisionCode = "TARGET_FILE_NOT_FOUND"
	// TARGET_FILE_IS_DIRECTORY means the target path exists but is a directory
	DiffFileDecisionCodeTargetFileIsDirectory DiffFileDecisionCode = "TARGET_FILE_IS_DIRECTORY"
)

// DiffFileDecision is the decision produced for a single resolved file.
type DiffFileDecision struct {
	// ResolvedFile is the declared file the decision was calculated for.
	ResolvedFile mergefileconfiguration.ResolvedFile
	// Code is the calculated decision code.
	Code DiffFileDecisionCode
}
