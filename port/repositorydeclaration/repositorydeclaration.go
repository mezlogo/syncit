package repositorydeclaration

// Represents request for parsing repository
// All repository MUST has following format:
// $repositoryRoot/hosts/$HOSTNAME.yaml - Each host is a file with declare modules and supported providers
// $repositoryRoot/modules/$MODULENAME/module.yaml - Each module is a dir with possible empty file `module.yaml`
type RepositoryDeclarationCommand struct {
	// An absolute path to repository with expected file structure
	RepositoryPath string

	// This host name used for determine which $HOSTNAME.yaml to process.
	HostName string
}

// Represent result structures
type RepositoryDeclarationResult struct {
	Modules []RepositoryDeclarationModule
}

type RepositoryDeclarationModule struct {
	// Module name for reporting and better error handling
	ModuleName string
	// Absolute path to module
	ModuleAbsoulutePath string
	// What module.yaml declared
	// Dictionary is an architecture decision for make it possible to easy extend functionality
	// Example:
	// ```yaml
	// pacman: ["nvim", "fish"]
	// systemd: ["systemd-resolved.service", "systemd-networkd.service"]
	// ```
	// Must be:
	// { "pacman": ["nvim", "fish"], "systemd": ["systemd-resolved.service", "systemd-networkd.service"] }
	Declarations map[string][]string
	// What files module declared
	// This is a RELATIVE to module pathes
	RelativeToModuleDeclaredFiles []string
}

// This module is a port - it isolates file system and yaml specific dependency from core logic
type RepositoryDeclarationPort interface {
	ParseRepository(command RepositoryDeclarationCommand)
}
