package toolsconfiguraion

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
func NewResolvedFileConfigurationMapping(root string, home string) {
	/*
	 * TODO: Use only string manipulation append "/" to and if no for root and home.
	 * Then create mapping:
	 * "config": home + ".config/"
	 * "local": home + ".local/"
	 * "ssh": home + ".ssh/"
	 * "home": home
	 * "root": root
	 * "etc": root + "etc/"
	 */
}
