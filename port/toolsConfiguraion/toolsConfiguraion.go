package toolsconfiguraion

import "strings"

// This is a simple tools extension mechanism

type ResolvedFileConfigurationMapping struct {
	// maps dir name from module to real path as root dir
	// schema: $repo/modules/$module/$configuration
	// example: $repo/modules/terminal/config
	// map: { "config": "$HOME/.config/" }
	// this values are always dirs with absolute path - starts with "/"
	// and cos it dirs - ends with "/"
	fileConfigurations map[string]string
}

// Default value used in application
// Must handle arguments with both notation: with and without ending slash
func NewResolvedFileConfigurationMapping(root string, home string) ResolvedFileConfigurationMapping {
	root = ensureTrailingSlash(root)
	home = ensureTrailingSlash(home)

	return ResolvedFileConfigurationMapping{
		fileConfigurations: map[string]string{
			"config": home + ".config/",
			"local":  home + ".local/",
			"ssh":    home + ".ssh/",
			"home":   home,
			"root":   root,
			"etc":    root + "etc/",
		},
	}
}

// Lookup returns the resolved host directory for a module top-level configuration directory.
func (m ResolvedFileConfigurationMapping) Lookup(configuration string) (string, bool) {
	value, ok := m.fileConfigurations[configuration]
	return value, ok
}

func ensureTrailingSlash(value string) string {
	if value == "" || strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}
