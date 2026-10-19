package syncfileconfiguration

import mergefileconfiguration "syncit/port/mergeFileConfiguration"

// SyncFileConfigurationUseCase applies declared user-space file configuration to the real filesystem.
type SyncFileConfigurationUseCase interface {
	SyncUserFiles(files []mergefileconfiguration.ResolvedFile, options SyncOptions) []SyncFileDecision
}

type SyncOptions struct {
	// Overwrite replaces existing non-link files or links pointing to a different target.
	Overwrite bool
	// Reset recreates the link even when it already points to the declared module file.
	Reset bool
}

type SyncFileDecisionCode string

const (
	SyncFileDecisionCodeLinked           SyncFileDecisionCode = "LINKED"
	SyncFileDecisionCodeRecreated        SyncFileDecisionCode = "RECREATED"
	SyncFileDecisionCodeSkippedUpToDate  SyncFileDecisionCode = "SKIPPED_UP_TO_DATE"
	SyncFileDecisionCodeSkippedExisting  SyncFileDecisionCode = "SKIPPED_EXISTING"
	SyncFileDecisionCodeSkippedDirectory SyncFileDecisionCode = "SKIPPED_DIRECTORY"
	SyncFileDecisionCodeFailed           SyncFileDecisionCode = "FAILED"
)

type SyncFileDecision struct {
	ResolvedFile mergefileconfiguration.ResolvedFile
	Code         SyncFileDecisionCode
	Err          error
}
