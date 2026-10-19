package filesystem

// This module is a prot for isolating fs IO

type FileSystemPort interface {
	// This method works with user space, not sudo or root required
	StatUserSpace(userAbsouteFilePaths []string) []FileSystemStat
}

type FileSystemStatCode string

const (
	FileSystemStatCodeFileNotExist FileSystemStatCode = "FILE_NOT_EXIST"
	FileSystemStatCodeLink         FileSystemStatCode = "LINK"
	FileSystemStatCodeFile         FileSystemStatCode = "FILE"
	FileSystemStatCodeDirectory    FileSystemStatCode = "DIRECTORY"
)

type FileSystemStat struct {
	absoluteFilePath string
	Code             FileSystemStatCode
}
