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
	stats := make([]port.FileSystemStat, 0, len(userAbsouteFilePaths))
	for _, absoluteFilePath := range userAbsouteFilePaths {
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
