package filesystem

import (
	"os"

	port "syncit/port/fileSystem"
)

// Adapter implements the filesystem port using the local OS filesystem.
type Adapter struct{}

// New creates a filesystem adapter.
func New() *Adapter {
	return &Adapter{}
}

var _ port.FileSystemPort = (*Adapter)(nil)

// StatUserSpace stats each absolute path without requiring elevated privileges.
func (a *Adapter) StatUserSpace(userAbsouteFilePaths []string) []port.FileSystemStat {
	return a.CalculateStatForUserFiles(userAbsouteFilePaths)
}

// CalculateStatForUserFiles stats each absolute user-space path.
func (a *Adapter) CalculateStatForUserFiles(userAbsouteFilePaths []string) []port.FileSystemStat {
	return statPaths(userAbsouteFilePaths)
}

// CalculateStatForRootFiles stats each absolute root-space path.
func (a *Adapter) CalculateStatForRootFiles(rootAbsouteFilePaths []string) []port.FileSystemStat {
	return statPaths(rootAbsouteFilePaths)
}

func statPaths(absoluteFilePaths []string) []port.FileSystemStat {
	stats := make([]port.FileSystemStat, 0, len(absoluteFilePaths))
	for _, absoluteFilePath := range absoluteFilePaths {
		stat := port.FileSystemStat{AbsoluteFilePath: absoluteFilePath}
		info, err := os.Lstat(absoluteFilePath)
		if err != nil {
			stat.Code = port.FileSystemStatCodeFileNotExist
			stats = append(stats, stat)
			continue
		}

		switch {
		case info.Mode()&os.ModeSymlink != 0:
			stat.Code = port.FileSystemStatCodeLink
		case info.IsDir():
			stat.Code = port.FileSystemStatCodeDirectory
		default:
			stat.Code = port.FileSystemStatCodeFile
		}
		stats = append(stats, stat)
	}
	return stats
}

// GetLinkForUserFile returns the link target for a user-space symlink.
func (a *Adapter) GetLinkForUserFile(absoluteFilePath string) (string, error) {
	return os.Readlink(absoluteFilePath)
}

// GetFileSizeInBytesForRootFile returns the size in bytes for a root-space file.
func (a *Adapter) GetFileSizeInBytesForRootFile(absoluteFilePath string) (int64, error) {
	info, err := os.Stat(absoluteFilePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ReadRawFileContent returns the raw content of a file.
func (a *Adapter) ReadRawFileContent(absoluteFilePath string) ([]byte, error) {
	return os.ReadFile(absoluteFilePath)
}
