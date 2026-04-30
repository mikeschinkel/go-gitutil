package gitutils

import (
	"github.com/mikeschinkel/go-dt"
)

// CreateDirPathFilter returns a filter for files within a repo but constrainted to
// just one directory in that repo. The filter returns true if the file is within
// the specified directory moduleRelPath should be relative to the git
// repo root (e.g., "gommod" or ".")
func CreateDirPathFilter(dirPath dt.RelDirPath) (ff FileFilter) {
	switch {
	case dirPath == "":
		dirPath = "."
		fallthrough

	case dirPath == ".":
		// If path is current directory, all files match
		ff = func(file dt.RelFilepath) bool {
			return true
		}

	default:
		// Ensure module path has trailing separator for prefix matching
		var prefix = dirPath
		if !prefix.HasSuffix("/") {
			prefix += "/"
		}

		ff = func(file dt.RelFilepath) bool {
			// Check if file path starts with module directory
			return file.HasPrefix(prefix)
		}
	}

	return ff
}

// CreateModuleFilter returns a filter that includes:
// 1. Files within the specified module directory
// 2. Files at the repo root (not in any subdirectory)
// This ensures repo-level files like README.md are included along with module files.
func CreateModuleFilter(dirPath dt.RelDirPath) (ff FileFilter) {
	switch {
	case dirPath == "" || dirPath == ".":
		// If path is repo root, all files match
		return func(file dt.RelFilepath) bool {
			return true
		}

	default:
		// Ensure module path has trailing separator for prefix matching
		var prefix = dirPath
		if !prefix.HasSuffix("/") {
			prefix += "/"
		}

		return func(file dt.RelFilepath) bool {
			// Include files in the module directory
			if file.HasPrefix(prefix) {
				return true
			}
			// Include files at repo root (no "/" in path)
			pathStr := string(file)
			for i := 0; i < len(pathStr); i++ {
				if pathStr[i] == '/' {
					return false // File is in a subdirectory
				}
			}
			return true // File is at repo root
		}
	}
}
