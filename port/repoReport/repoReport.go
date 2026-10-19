package repoReport

import "syncit/port/repoAO"

// This port isolate pterm or any other beutiful output library AND stdout from core and cmd

type RepoReportUseCase interface {

	// Just report modules from repoAO
	ReportRepo(repoPath string, modules []repoAO.RepositoryDeclarationModule)
}
