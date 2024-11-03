package helpers

import (
	"path/filepath"
	"strings"
)

func IsAllowedExtension(filename string, allowedExts []string) bool {
	for _, ext := range allowedExts {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	return false
}

func IsExcludedDir(pth string, excludedDirs []string) bool {
	normalizedPath := filepath.Clean(pth)
	pathComponents := strings.Split(normalizedPath, string(filepath.Separator))

	for _, exclude := range excludedDirs {
		excludePath := filepath.Clean(exclude)
		excludeComponents := strings.Split(excludePath, string(filepath.Separator))

		for _, component := range pathComponents {
			if component == excludeComponents[0] {
				return true
			}
		}
	}

	return false
}
