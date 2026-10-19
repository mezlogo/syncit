package difffileconfiguration

import mergefileconfiguration "syncit/port/mergeFileConfiguration"

// This module is a core domain logic for calculating in memory file pathes

// This is the most important part of whole application
// Builds for each file system file and declared file configuration result what to do
type DiffFileConfigurationUseCase interface {
	// TARGET_FILE_EQUAL_TO_DECLARED for user space files means it LINKS to declared file
	// TARGET_FILE_NOT_EQUAL_TO_DECLARED means it could be either a link to other file
	//   or just a file
	CalculateDiffForUserFiles(userConfigurationFiles []mergefileconfiguration.ResolvedFile)
}

/*
TODO: declare multicode configuration for decision making:
- TARGET_FILE_EQUAL_TO_DECLARED
- TARGET_FILE_NOT_EQUAL_TO_DECLARED
- TARGET_FILE_NOT_FOUND
- TARGET_FILE_IS_DIRECTORY
*/
