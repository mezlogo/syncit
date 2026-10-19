package repoReport

import (
	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	processhandlers "syncit/port/processHandlers"
	repoao "syncit/port/repoAO"

	syncfileconfiguration "syncit/port/syncFileConfiguration"
)

// This port isolate pterm or any other beutiful output library AND stdout from core and cmd

type RepoReportUseCase interface {

	// Just report modules from repoAO
	ReportRepo(repoPath string, modules []repoao.RepositoryDeclarationModule)

	// ReportMergedRepo reports resolved file configuration and any merge error.
	ReportMergedRepo(repoPath string, resolved mergefileconfiguration.ResolvedFileConfiguration, mergeErr error)

	// ReportDiff reports diff decisions with colorized decision codes.
	ReportDiff(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error)

	// ReportSync reports sync decisions with colorized decision codes.
	ReportSync(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error)

	// ReportDiffRoot reports diff decisions for root-space files with colorized decision codes.
	ReportDiffRoot(repoPath string, decisions []difffileconfiguration.DiffFileDecision, mergeErr error)

	// ReportSyncRoot reports sync decisions for root-space files with colorized decision codes.
	ReportSyncRoot(repoPath string, decisions []syncfileconfiguration.SyncFileDecision, mergeErr error)

	// ReportProcess reports process decisions (dry-run or executed) with colorized codes.
	ReportProcess(repoPath string, decisions []processhandlers.ProcessDecision, mergeErr error)
}
