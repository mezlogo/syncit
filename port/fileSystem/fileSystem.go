package filesystem

// This module is a prot for isolating fs IO

type FileSystemPort interface {
	// This method works with user space, not sudo or root required
	StatUserSpace(userAbsouteFilePaths []string) []FileSystemStat
}

type FileSystemStat struct {
	absoluteFilePath string
	// TODO: declare code: FILE_NOT_EXIST, LINK_TO, FILE, DIRECTORY
}
