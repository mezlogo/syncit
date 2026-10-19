package filesystem

// This module is a prot for isolating fs IO

type FileSystemPort interface {
	// This method works with user space, not sudo or root required
	StatUserSpace(userAbsouteFilePaths []string) map[string]StatResult
}

/*
TODO: wirte a hierarchy like sruct for StatResult
- FILE_NOT_EXIST
- LINK_TO(target)
- FILE
- DIRECTORY
*/
