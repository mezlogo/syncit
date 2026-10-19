package repoReport

import (
	difffileconfiguration "syncit/port/diffFileConfiguration"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
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
}
