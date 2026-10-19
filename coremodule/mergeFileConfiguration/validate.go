package mergefileconfiguration

import (
	"fmt"
	"path"
	"strings"
)

func normalizeRelativeFilePath(relativeFilePath string) (cleaned string, topLevel string, rest string, err error) {
	if relativeFilePath == "" {
		return "", "", "", fmt.Errorf("relative file path is empty")
	}
	if strings.HasPrefix(relativeFilePath, "/") {
		return "", "", "", fmt.Errorf("relative file path %q must not be absolute", relativeFilePath)
	}

	cleaned = path.Clean(relativeFilePath)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned != relativeFilePath {
		return "", "", "", fmt.Errorf("invalid relative file path %q", relativeFilePath)
	}

	parts := strings.Split(cleaned, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("relative file path %q must include a mapped directory and a file", relativeFilePath)
	}

	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", "", "", fmt.Errorf("invalid relative file path %q", relativeFilePath)
		}
	}

	return cleaned, parts[0], strings.Join(parts[1:], "/"), nil
}
