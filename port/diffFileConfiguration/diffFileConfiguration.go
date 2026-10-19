package difffileconfiguration

// This module is a core domain logic for calculating in memory file pathes

// This is the most important part of whole application
// Builds for each file system file and declared file configuration result what to do
type DiffFileConfigurationUseCase interface {
	CalculateDiffForUserFiles()
}

/*
TODO: create a structures for diff like:

*/
