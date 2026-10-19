package syncfileconfiguration

import (
	"errors"
	"testing"

	filesystem "syncit/port/fileSystem"
	mergefileconfiguration "syncit/port/mergeFileConfiguration"
	port "syncit/port/syncFileConfiguration"
)

type fakeFS struct {
	stats        map[string]filesystem.FileSystemStat
	links        map[string]string
	rootSizes    map[string]int64
	rootContents map[string][]byte
	copiedLinks  map[string]string

	createdLinks map[string]string
	removedFiles []string

	errOnCreate error
}

func newFakeFS() *fakeFS {
	return &fakeFS{
		stats:        map[string]filesystem.FileSystemStat{},
		links:        map[string]string{},
		rootSizes:    map[string]int64{},
		rootContents: map[string][]byte{},
		copiedLinks:  map[string]string{},

		createdLinks: map[string]string{},
	}
}

func (f *fakeFS) CalculateStatForUserFiles(paths []string) []filesystem.FileSystemStat {
	out := make([]filesystem.FileSystemStat, 0, len(paths))
	for _, p := range paths {
		if s, ok := f.stats[p]; ok {
			out = append(out, s)
		} else {
			out = append(out, filesystem.FileSystemStat{AbsoluteFilePath: p, Code: filesystem.FileSystemStatCodeFileNotExist})
		}
	}
	return out
}
func (f *fakeFS) CalculateStatForRootFiles([]string) []filesystem.FileSystemStat { return nil }
func (f *fakeFS) GetLinkForUserFile(p string) (string, error) {
	if t, ok := f.links[p]; ok {
		return t, nil
	}
	return "", errors.New("not a link")
}
func (f *fakeFS) GetFileSizeInBytesForRootFile(p string) (int64, error) {
	if size, ok := f.rootSizes[p]; ok {
		return size, nil
	}
	return 0, nil
}
func (f *fakeFS) ReadRawFileContent(p string) ([]byte, error) {
	if content, ok := f.rootContents[p]; ok {
		return content, nil
	}
	return nil, nil
}

func (f *fakeFS) EnsureDirRecursive(string) error { return nil }
func (f *fakeFS) CreateUserSymbolicLink(target, linkPath string) error {
	if f.errOnCreate != nil {
		return f.errOnCreate
	}
	f.createdLinks[linkPath] = target
	return nil
}
func (f *fakeFS) RemoveUserFile(p string) error {
	f.removedFiles = append(f.removedFiles, p)
	return nil
}

func (f *fakeFS) EnsureRootDirRecursive(string) error { return nil }
func (f *fakeFS) CopyFileToRoot(source, target string) error {
	f.copiedLinks[target] = source
	return nil
}
func (f *fakeFS) RemoveRootFile(p string) error {
	f.removedFiles = append(f.removedFiles, p)
	return nil
}

var _ filesystem.FileSystemPort = (*fakeFS)(nil)

func TestUseCase_SyncUserFiles(t *testing.T) {
	target := "/home/u/.config/git/config"
	declared := "/repo/modules/a/config/git/config"
	mkFile := func(code filesystem.FileSystemStatCode) mergefileconfiguration.ResolvedFile {
		return mergefileconfiguration.ResolvedFile{AbsoluteModuleFilePath: declared, AbsoluteHostFilePath: target}
	}

	tests := []struct {
		name     string
		file     mergefileconfiguration.ResolvedFile
		options  port.SyncOptions
		setup    func(*fakeFS)
		wantCode port.SyncFileDecisionCode
		wantErr  bool
	}{
		{
			name: "creates link when target missing",
			file: mkFile(filesystem.FileSystemStatCodeFileNotExist),
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFileNotExist}
			},
			wantCode: port.SyncFileDecisionCodeLinked,
		},
		{
			name: "skips up to date link",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = declared
			},
			wantCode: port.SyncFileDecisionCodeSkippedUpToDate,
		},
		{
			name:    "recreates up to date link with reset",
			options: port.SyncOptions{Reset: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = declared
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name: "skips existing regular file without overwrite",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFile}
			},
			wantCode: port.SyncFileDecisionCodeSkippedExisting,
		},
		{
			name:    "recreates existing regular file with overwrite",
			options: port.SyncOptions{Overwrite: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFile}
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name: "skips wrong link without overwrite",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = "/elsewhere"
			},
			wantCode: port.SyncFileDecisionCodeSkippedExisting,
		},
		{
			name:    "recreates wrong link with overwrite",
			options: port.SyncOptions{Overwrite: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeLink}
				f.links[target] = "/elsewhere"
			},
			wantCode: port.SyncFileDecisionCodeRecreated,
		},
		{
			name:    "never replaces a directory",
			options: port.SyncOptions{Overwrite: true, Reset: true},
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeDirectory}
			},
			wantCode: port.SyncFileDecisionCodeSkippedDirectory,
		},
		{
			name: "fails on create link error",
			setup: func(f *fakeFS) {
				f.stats[target] = filesystem.FileSystemStat{AbsoluteFilePath: target, Code: filesystem.FileSystemStatCodeFileNotExist}
				f.errOnCreate = errors.New("boom")
			},
			wantCode: port.SyncFileDecisionCodeFailed,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeFS()
			if tt.setup != nil {
				tt.setup(f)
			}
			file := tt.file
			if file.AbsoluteHostFilePath == "" {
				file = mkFile(filesystem.FileSystemStatCodeFileNotExist)
			}
			got := New(f).SyncUserFiles([]mergefileconfiguration.ResolvedFile{file}, tt.options)
			if len(got) != 1 {
				t.Fatalf("decisions length: got %d, want 1", len(got))
			}
			if got[0].Code != tt.wantCode {
				t.Errorf("code: got %q, want %q", got[0].Code, tt.wantCode)
			}
			if (got[0].Err != nil) != tt.wantErr {
				t.Errorf("error status: got %v, wantErr %v", got[0].Err, tt.wantErr)
			}
		})
	}
}
