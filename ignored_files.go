package gitutils

import (
	"context"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// IgnoredFiles returns repo-root-relative file paths that should be treated as
// ignored regardless of regardless of staging/tracking state.
//
// It unions:
//  1. ignored + untracked: git ls-files -o -i --exclude-standard
//  2. tracked-but-ignored: git ls-files -c -i --exclude-standard
//
// Notes:
// - This includes ignore rules from .gitignore, .git/info/exclude, and the user's global excludes file.
// - Git ignore rules do not apply to tracked files, but (2) finds tracked files that *match* ignore rules.
func IgnoredFiles(ctx context.Context, repoDir dt.DirPath) (files []dt.RelFilepath, err error) {
	var oFiles, cFiles []string

	oFiles, err = LsFilesRaw(ctx, repoDir, []string{"-o", "-i", "--exclude-standard"})
	if err != nil {
		goto end
	}

	cFiles, err = LsFilesRaw(ctx, repoDir, []string{"-c", "-i", "--exclude-standard"})
	if err != nil {
		goto end
	}
	files = make([]dt.RelFilepath, len(oFiles)+len(cFiles))
	{
		n := 0
		for _, xFiles := range [][]string{oFiles, cFiles} {
			for _, file := range xFiles {
				files[n] = dt.RelFilepath(file)
				n++
			}
		}
	}
end:
	return files, err
}

func LsFilesRaw(ctx context.Context, repoDir dt.DirPath, args []string) (paths []string, err error) {
	var out string

	out, err = newGitRunner(repoDir).runGit(ctx, runGitArgs{
		NULSeparated: true,
		Command:      append([]string{"ls-files", "-z"}, args...),
	})

	if err != nil {
		goto end
	}

	if out == "" {
		goto end
	}

	paths = strings.Split(strings.TrimRight(out, "\x00"), "\x00")

end:
	return paths, err
}

// AppendToGitignore appends patterns to .gitignore file, checking for duplicates.
// Creates .gitignore if it doesn't exist.
// Patterns should be repo-root-relative paths (e.g., "debug.log", "build/").
func AppendToGitignore(repoRoot dt.DirPath, patterns []string) (err error) {
	var gitignorePath dt.Filepath

	gitignorePath = dt.FilepathJoin(repoRoot, ".gitignore")
	err = appendPatternsToFile(gitignorePath, patterns)
	if err != nil {
		err = NewErr(ErrAppendToGitignore, err,
			"file", gitignorePath,
			"pattern_count", len(patterns),
		)
		goto end
	}

end:
	return err
}

// AppendToGitExclude appends patterns to .git/info/exclude file, checking for duplicates.
// File should already exist (created by git init).
// Patterns should be repo-root-relative paths (e.g., ".idea/", ".vscode/").
func AppendToGitExclude(repoRoot dt.DirPath, patterns []string) (err error) {
	var excludePath dt.Filepath

	excludePath = dt.FilepathJoin(repoRoot, ".git/info/exclude")
	err = appendPatternsToFile(excludePath, patterns)
	if err != nil {
		err = NewErr(ErrAppendToGitExclude, err,
			"file", excludePath,
			"pattern_count", len(patterns),
		)
		goto end
	}

end:
	return err
}

// UnstageFiles removes files from the git index (unstages them) without deleting from working tree.
// Uses `git reset HEAD <file>` for each file.
// This is necessary when adding staged files to .gitignore - they won't be ignored until unstaged.
func UnstageFiles(ctx context.Context, repoRoot dt.DirPath, files []dt.RelFilepath) (err error) {
	if len(files) == 0 {
		goto end
	}

	for _, file := range files {
		_, err = newGitRunner(repoRoot).runGit(ctx, runGitArgs{
			Command: []string{"reset", "HEAD", string(file)},
		})
		if err != nil {
			err = NewErr(ErrUnstageFile, err,
				"file", file,
				"repo_root", repoRoot,
			)
			goto end
		}
	}

end:
	return err
}

// appendPatternsToFile appends patterns to file, avoiding duplicates.
// Creates file if it doesn't exist.
func appendPatternsToFile(filePath dt.Filepath, patterns []string) (err error) {
	var existingContent []byte
	var existingPatterns map[string]bool
	var newPatterns []string
	var f *os.File

	// Read existing content to check for duplicates
	existingContent, err = os.ReadFile(string(filePath))
	if err != nil && !os.IsNotExist(err) {
		goto end
	}

	// Build set of existing patterns (non-comment, non-empty lines)
	existingPatterns = make(map[string]bool)
	if len(existingContent) > 0 {
		lines := strings.Split(string(existingContent), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				existingPatterns[line] = true
			}
		}
	}

	// Filter out patterns that already exist
	newPatterns = make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if !existingPatterns[pattern] {
			newPatterns = append(newPatterns, pattern)
		}
	}

	// Nothing to add if all patterns already exist
	if len(newPatterns) == 0 {
		goto end
	}

	// Open file for append (create if doesn't exist)
	f, err = os.OpenFile(string(filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		goto end
	}
	defer f.Close()

	// Ensure file ends with newline before appending
	if len(existingContent) > 0 && existingContent[len(existingContent)-1] != '\n' {
		_, err = f.WriteString("\n")
		if err != nil {
			goto end
		}
	}

	// Append new patterns (one per line)
	for _, pattern := range newPatterns {
		_, err = f.WriteString(pattern + "\n")
		if err != nil {
			goto end
		}
	}

end:
	return err
}
