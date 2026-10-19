package repositorydeclaration

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	port "syncit/port/repositorydeclaration"
)

/*
Exmaple for expected yaml - all fields MUST be specified.
hosts/$host.yaml
```yaml
modules:
  - basic-linux-lts
  - basic-linux-amdgpu

tools:
  - pacman
  - aur
  - systemd
  - home
  - root

```
*/
type HostDeclarationYaml struct {
	Modules []string `yaml:"modules"`
	Tools   []string `yaml:"tools"`
}

/*
Exmaple for expected yaml - all fields are a map of string -> string[]
modules/$host/module.yaml
```yaml
pacman:
  - fish
  - nvim

aur:
  - vscode
  - leaf

systemd:
  - sytemd-networked.service
  - sytemd-resolved.service

```
*/
type ModuleDeclarationYaml map[string][]string

// Adapter parses repository declarations from the filesystem and YAML.
type Adapter struct{}

// New creates a repository declaration adapter.
func New() *Adapter {
	return &Adapter{}
}

var _ port.RepositoryDeclarationPort = (*Adapter)(nil)

// ParseRepository reads the host declaration, every module declaration, and
// the relative file list for every declared module.
func (a *Adapter) ParseRepository(command port.RepositoryDeclarationCommand) (port.RepositoryDeclarationResult, error) {

	if command.RepositoryPath == "" {
		return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: repository path is empty", port.ErrNoRepositoryFound)
	}
	if command.HostName == "" {
		return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: host name is empty", port.ErrNoHostFileFound)
	}

	repoPath, err := filepath.Abs(command.RepositoryPath)
	if err != nil {
		return port.RepositoryDeclarationResult{}, fmt.Errorf("resolve repository path: %w", err)
	}

	info, err := os.Stat(repoPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s", port.ErrNoRepositoryFound, repoPath)
		}
		return port.RepositoryDeclarationResult{}, fmt.Errorf("stat repository: %w", err)
	}
	if !info.IsDir() {
		return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s is not a directory", port.ErrNoRepositoryFound, repoPath)
	}

	hostFile := filepath.Join(repoPath, "hosts", command.HostName+".yaml")
	hostData, err := os.ReadFile(hostFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s", port.ErrNoHostFileFound, hostFile)
		}
		return port.RepositoryDeclarationResult{}, fmt.Errorf("read host declaration: %w", err)
	}

	var host HostDeclarationYaml
	if err := yaml.Unmarshal(hostData, &host); err != nil {
		return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s: %v", port.ErrBadHostFile, hostFile, err)
	}

	modules := make([]port.RepositoryDeclarationModule, 0, len(host.Modules))
	for _, rawModuleName := range host.Modules {

		moduleName := strings.TrimSpace(rawModuleName)
		if moduleName == "" {
			continue
		}
		if moduleName == "." || moduleName == ".." || moduleName != filepath.Base(moduleName) {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: invalid module name %q", port.ErrBadHostFile, moduleName)
		}

		moduleDir, err := filepath.Abs(filepath.Join(repoPath, "modules", moduleName))
		if err != nil {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("resolve module %q path: %w", moduleName, err)
		}

		moduleInfo, err := os.Stat(moduleDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s", port.ErrDeclaredModuleNotFound, moduleDir)
			}
			return port.RepositoryDeclarationResult{}, fmt.Errorf("stat module %q: %w", moduleName, err)
		}
		if !moduleInfo.IsDir() {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("%w: %s is not a directory", port.ErrDeclaredModuleNotFound, moduleDir)
		}

		declarations, err := readModuleDeclaration(moduleDir)
		if err != nil {
			return port.RepositoryDeclarationResult{}, err
		}

		files, err := collectDeclaredFiles(moduleDir)

		if err != nil {
			return port.RepositoryDeclarationResult{}, fmt.Errorf("collect files for module %q: %w", moduleName, err)
		}

		modules = append(modules, port.RepositoryDeclarationModule{
			ModuleName:          moduleName,
			ModuleAbsoulutePath: moduleDir,
			Declarations:        map[string][]string(declarations),
			RelativeFilePaths:   files,
		})
	}

	return port.RepositoryDeclarationResult{Modules: modules}, nil
}

func readModuleDeclaration(moduleDir string) (ModuleDeclarationYaml, error) {
	moduleFile := filepath.Join(moduleDir, "module.yaml")
	data, err := os.ReadFile(moduleFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", port.ErrModuleDeclarationFileNotFound, moduleFile)
		}
		return nil, fmt.Errorf("read module declaration: %w", err)
	}

	declarations := ModuleDeclarationYaml{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return declarations, nil
	}
	if err := yaml.Unmarshal(data, &declarations); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", port.ErrBadModuleDeclaration, moduleFile, err)
	}
	return declarations, nil
}

func collectDeclaredFiles(moduleDir string) ([]string, error) {

	files := make([]string, 0)
	err := filepath.WalkDir(moduleDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(moduleDir, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(filepath.Clean(relativePath))
		if relativePath == "." || relativePath == "module.yaml" {
			return nil
		}
		if strings.HasPrefix(relativePath, "/") || strings.HasPrefix(relativePath, "../") {
			return nil
		}

		files = append(files, relativePath)

		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
