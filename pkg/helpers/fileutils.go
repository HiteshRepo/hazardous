package helpers

import "strings"

func IsAllowedExtension(filename string, allowedExts []string) bool {
	for _, ext := range allowedExts {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}

func IsExcludedDir(path string, excludedDirs []string) bool {
	for _, exclude := range excludedDirs {
		if strings.Contains(path, exclude) {
			return true
		}
	}

	return false
}
