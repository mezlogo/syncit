package model

// ==> INSIDE FLOW

// Represents request for parsing repository
// All repository MUST has following format:
// $repositoryRoot/hosts/$HOSTNAME.yaml - Each host is a file with declare modules and supported providers
// $repositoryRoot/modules/$MODULENAME/module.yaml - Each module is a dir with possible empty file `module.yaml`
type RepositoryParsingCommand struct {
	// An absolute path to repository with expected file structure
	RepositoryPath string

	// This host name used for determine which $HOSTNAME.yaml to process.
	HostName string
}

// Represents result for
type RepositoryParseingResult struct {
	DeclaredHosts   []string
	DeclaredModules []string
}
