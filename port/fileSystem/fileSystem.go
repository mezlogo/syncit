package filesystem

// This module is a prot for isolating fs IO

type FileSystemPort interface {
	// This method works with user space, not sudo or root required
	CalculateStatForUserFiles(userAbsouteFilePaths []string) []FileSystemStat

	// port for simple os call
	GetLinkForUserFile(absoluteFilePath string) (string, error)

	// get file size in bytes for eqaul test
	GetFileSizeInBytesForUserFile(absoluteFilePath string) (int64, error)
}

type FileSystemStatCode string

const (
	FileSystemStatCodeFileNotExist FileSystemStatCode = "FILE_NOT_EXIST"
	FileSystemStatCodeLink         FileSystemStatCode = "LINK"
	FileSystemStatCodeFile         FileSystemStatCode = "FILE"
	FileSystemStatCodeDirectory    FileSystemStatCode = "DIRECTORY"
)

type FileSystemStat struct {
	// AbsoluteFilePath is the absolute path that was statted.
	AbsoluteFilePath string

	Code FileSystemStatCode
}
