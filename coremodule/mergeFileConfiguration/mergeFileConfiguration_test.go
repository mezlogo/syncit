package mergefileconfiguration

import (
	"errors"
	"reflect"
	"testing"

	mergeport "syncit/port/mergeFileConfiguration"
	repoao "syncit/port/repoAO"
	toolsconfiguraion "syncit/port/toolsConfiguraion"
)

func TestUseCase_MergeFiles(t *testing.T) {
	mapping := toolsconfiguraion.NewResolvedFileConfigurationMapping("/", "/home/user")

	tests := []struct {
		name     string
		modules  []repoao.RepositoryDeclarationModule
		wantUser []mergeport.ResolvedFile
		wantRoot []mergeport.ResolvedFile
		wantErr  error
	}{
		{
			name: "maps user and root files",
			modules: []repoao.RepositoryDeclarationModule{
				{
					ModuleAbsoulutePath: "/repo/modules/base",
					RelativeFilePaths: []string{
						"config/nvim/init.lua",
						"ssh/config",
						"etc/systemd/network/20-wired.network",
						"root/boot/loader.conf",
					},
				},
			},
			wantUser: []mergeport.ResolvedFile{
				{
					AbsoluteModuleFilePath: "/repo/modules/base/config/nvim/init.lua",
					AbsoluteHostFilePath:   "/home/user/.config/nvim/init.lua",
				},
				{
					AbsoluteModuleFilePath: "/repo/modules/base/ssh/config",
					AbsoluteHostFilePath:   "/home/user/.ssh/config",
				},
			},
			wantRoot: []mergeport.ResolvedFile{
				{
					AbsoluteModuleFilePath: "/repo/modules/base/root/boot/loader.conf",
					AbsoluteHostFilePath:   "/boot/loader.conf",
				},
				{
					AbsoluteModuleFilePath: "/repo/modules/base/etc/systemd/network/20-wired.network",
					AbsoluteHostFilePath:   "/etc/systemd/network/20-wired.network",
				},
			},
		},
		{
			name: "last write wins and reports duplicate target",
			modules: []repoao.RepositoryDeclarationModule{
				{
					ModuleName:          "a",
					ModuleAbsoulutePath: "/repo/modules/a",
					RelativeFilePaths:   []string{"config/git/config"},
				},
				{
					ModuleName:          "b",
					ModuleAbsoulutePath: "/repo/modules/b",
					RelativeFilePaths:   []string{"config/git/config"},
				},
			},
			wantUser: []mergeport.ResolvedFile{
				{
					AbsoluteModuleFilePath: "/repo/modules/b/config/git/config",
					AbsoluteHostFilePath:   "/home/user/.config/git/config",
				},
			},
			wantRoot: []mergeport.ResolvedFile{},
			wantErr:  mergeport.ErrMultipleDeclarations,
		},
		{
			name: "reports dir file conflict",
			modules: []repoao.RepositoryDeclarationModule{
				{
					ModuleAbsoulutePath: "/repo/modules/a",
					RelativeFilePaths:   []string{"config/foo"},
				},
				{
					ModuleAbsoulutePath: "/repo/modules/b",
					RelativeFilePaths:   []string{"config/foo/bar"},
				},
			},
			wantUser: []mergeport.ResolvedFile{
				{
					AbsoluteModuleFilePath: "/repo/modules/a/config/foo",
					AbsoluteHostFilePath:   "/home/user/.config/foo",
				},
				{
					AbsoluteModuleFilePath: "/repo/modules/b/config/foo/bar",
					AbsoluteHostFilePath:   "/home/user/.config/foo/bar",
				},
			},
			wantRoot: []mergeport.ResolvedFile{},
			wantErr:  mergeport.ErrOneDeclaredDirOtherDeclaredFileConflict,
		},
		{
			name: "reports missing mapping",
			modules: []repoao.RepositoryDeclarationModule{
				{
					ModuleAbsoulutePath: "/repo/modules/a",
					RelativeFilePaths:   []string{"unknown/file"},
				},
			},
			wantUser: []mergeport.ResolvedFile{},
			wantRoot: []mergeport.ResolvedFile{},
			wantErr:  mergeport.ErrDeclaredFileNotFoundInFileConfigurationMapping,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New().MergeFiles(tt.modules, mapping)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error mismatch: got %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got.UserFiles, tt.wantUser) {
				t.Errorf("user files mismatch:\ngot:  %#v\nwant: %#v", got.UserFiles, tt.wantUser)
			}
			if !reflect.DeepEqual(got.RootFiles, tt.wantRoot) {
				t.Errorf("root files mismatch:\ngot:  %#v\nwant: %#v", got.RootFiles, tt.wantRoot)
			}
		})
	}
}
