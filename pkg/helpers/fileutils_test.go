package helpers

import (
	"testing"
)

func TestIsAllowedExtension(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		allowedExts []string
		shouldAllow bool
		description string
	}{
		{
			name:        "basic shell script",
			filename:    "deploy.sh",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: true,
			description: "Should allow .sh files when .sh is in allowed extensions",
		},
		{
			name:        "basic makefile",
			filename:    "Makefile",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: true,
			description: "Should allow Makefile when Makefile is in allowed extensions",
		},
		{
			name:        "makefile with prefix",
			filename:    "project.Makefile",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: true,
			description: "Should allow files ending with Makefile",
		},
		{
			name:        "case sensitive shell script",
			filename:    "script.SH",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: false,
			description: "Should be case sensitive for extensions",
		},
		{
			name:        "dot prefixed file",
			filename:    ".deploy.sh",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: true,
			description: "Should allow hidden files with valid extensions",
		},
		{
			name:        "disallowed extension",
			filename:    "script.py",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: false,
			description: "Should not allow files with extensions not in the list",
		},
		{
			name:        "no extension",
			filename:    "scriptfile",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: false,
			description: "Should not allow files without extensions",
		},
		{
			name:        "empty filename",
			filename:    "",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: false,
			description: "Should handle empty filenames gracefully",
		},
		{
			name:        "empty extensions list",
			filename:    "script.sh",
			allowedExts: []string{},
			shouldAllow: false,
			description: "Should not allow any files when extensions list is empty",
		},
		{
			name:        "nil extensions list",
			filename:    "script.sh",
			allowedExts: nil,
			shouldAllow: false,
			description: "Should handle nil extensions list gracefully",
		},
		{
			name:        "path with directories",
			filename:    "src/scripts/deploy.sh",
			allowedExts: []string{".sh", "Makefile"},
			shouldAllow: true,
			description: "Should check extension regardless of path depth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedExtension(tt.filename, tt.allowedExts)
			if got != tt.shouldAllow {
				t.Errorf("\nTest: %s\nDescription: %s\nFilename: %q\nExpected: %v\nGot: %v",
					tt.name, tt.description, tt.filename, tt.shouldAllow, got)
			}
		})
	}
}

func TestIsExcludedDir(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		excludedDirs  []string
		shouldExclude bool
		description   string
	}{
		{
			name:          "basic excluded directory",
			path:          "node_modules/package/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: true,
			description:   "Should exclude files in node_modules directory",
		},
		{
			name:          "basic non-excluded directory",
			path:          "src/package/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: false,
			description:   "Should not exclude files in non-excluded directories",
		},
		{
			name:          "nested excluded directory",
			path:          "project/vendor/lib/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: true,
			description:   "Should exclude files in nested excluded directories",
		},
		{
			name:          "directory with similar name",
			path:          "my_node_modules_test/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: false,
			description:   "Should not exclude directories that merely contain excluded names",
		},
		{
			name:          "partial directory match",
			path:          "vendors/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: false,
			description:   "Should not exclude directories that partially match excluded names",
		},
		{
			name:          "empty excluded dirs",
			path:          "vendor/file.sh",
			excludedDirs:  []string{},
			shouldExclude: false,
			description:   "Should not exclude any directories when exclusion list is empty",
		},
		{
			name:          "nil excluded dirs",
			path:          "vendor/file.sh",
			excludedDirs:  nil,
			shouldExclude: false,
			description:   "Should handle nil exclusion list gracefully",
		},
		{
			name:          "empty path",
			path:          "",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: false,
			description:   "Should handle empty paths gracefully",
		},
		{
			name:          "dot-prefixed excluded directory",
			path:          ".vendor/file.sh",
			excludedDirs:  []string{"vendor"},
			shouldExclude: false,
			description:   "Should not exclude when excluded dir name is part of another directory name",
		},
		{
			name:          "multiple nested excluded directories",
			path:          "node_modules/vendor/file.sh",
			excludedDirs:  []string{"node_modules", "vendor"},
			shouldExclude: true,
			description:   "Should exclude if path contains any excluded directory",
		},
		{
			name:          "case sensitive directory check",
			path:          "NODE_MODULES/file.sh",
			excludedDirs:  []string{"node_modules"},
			shouldExclude: false,
			description:   "Should be case sensitive when checking directory names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsExcludedDir(tt.path, tt.excludedDirs)
			if got != tt.shouldExclude {
				t.Errorf("\nTest: %s\nDescription: %s\nPath: %q\nExpected: %v\nGot: %v",
					tt.name, tt.description, tt.path, tt.shouldExclude, got)
			}
		})
	}
}
