package filesystem

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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

// GetFileSizeInBytesForRootFile — try as user, escalate on permission error
func (a *Adapter) GetFileSizeInBytesForRootFile(absoluteFilePath string) (int64, error) {
	info, err := os.Stat(absoluteFilePath)
	if err == nil {
		return info.Size(), nil
	}
	if !os.IsPermission(err) {
		return 0, err
	}
	out, err := runRootCommandOutput("stat", "-c", "%s", absoluteFilePath)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

func (a *Adapter) ReadRawFileContent(absoluteFilePath string) ([]byte, error) {
	data, err := os.ReadFile(absoluteFilePath)
	if err == nil {
		return data, nil
	}
	if !os.IsPermission(err) {
		return nil, err
	}
	return runRootCommandOutput("cat", absoluteFilePath)
}

func (a *Adapter) EnsureDirRecursive(absoluteDirPath string) error {
	return os.MkdirAll(absoluteDirPath, 0o755)
}

func (a *Adapter) CreateUserSymbolicLink(absoluteTargetPath, absoluteLinkPath string) error {
	return os.Symlink(absoluteTargetPath, absoluteLinkPath)
}

func (a *Adapter) RemoveUserFile(absoluteFilePath string) error {
	info, err := os.Lstat(absoluteFilePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to remove directory: %s", absoluteFilePath)
	}
	return os.Remove(absoluteFilePath)
}

func (a *Adapter) EnsureRootDirRecursive(absoluteDirPath string) error {
	return runRootCommand("mkdir", "-p", absoluteDirPath)
}

func (a *Adapter) CopyFileToRoot(sourceAbsoluteFilePath, targetAbsoluteFilePath string) error {
	return runRootCommand("cp", sourceAbsoluteFilePath, targetAbsoluteFilePath)
}

func (a *Adapter) RemoveRootFile(absoluteFilePath string) error {
	return runRootCommand("rm", "-f", absoluteFilePath)
}

func runRootCommand(name string, args ...string) error {
	cmd := buildRootCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runRootCommandOutput(name string, args ...string) ([]byte, error) {
	cmd := buildRootCommand(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func buildRootCommand(name string, args ...string) *exec.Cmd {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...)
	}
	return exec.Command("sudo", append([]string{name}, args...)...)
}
