package gitutil

import (
	"context"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// IgnoredFilesArgs holds the arguments of IgnoredFiles.
type IgnoredFilesArgs struct {
	RepoArgs
}

// IgnoredFiles returns the ignored files in dir. See (*Repo).IgnoredFiles.
func IgnoredFiles(ctx context.Context, dir dt.DirPath, args *IgnoredFilesArgs) ([]dt.RelFilepath, error) {
	return NewRepo(dir, nil).IgnoredFiles(ctx, args)
}

// IgnoredFiles returns repo-root-relative file paths that should be treated as
// ignored regardless of staging/tracking state.
//
// It unions:
//  1. ignored + untracked: git ls-files -o -i --exclude-standard
//  2. tracked-but-ignored: git ls-files -c -i --exclude-standard
//
// Notes:
// - This includes ignore rules from .gitignore, .git/info/exclude, and the user's global excludes file.
// - Git ignore rules do not apply to tracked files, but (2) finds tracked files that *match* ignore rules.
//
// It does not need Open. args may be nil.
func (r *Repo) IgnoredFiles(ctx context.Context, args *IgnoredFilesArgs) (files []dt.RelFilepath, err error) {
	var oFiles, cFiles []string
	var ls LsFilesRawArgs

	if args != nil {
		ls.RepoArgs = args.RepoArgs
	}

	ls.Args = []string{"-o", "-i", "--exclude-standard"}
	oFiles, err = r.LsFilesRaw(ctx, &ls)
	if err != nil {
		goto end
	}

	ls.Args = []string{"-c", "-i", "--exclude-standard"}
	cFiles, err = r.LsFilesRaw(ctx, &ls)
	if err != nil {
		goto end
	}
	files = make([]dt.RelFilepath, 0, len(oFiles)+len(cFiles))
	for _, xFiles := range [][]string{oFiles, cFiles} {
		for _, file := range xFiles {
			files = append(files, dt.RelFilepath(file))
		}
	}
end:
	return files, err
}

// LsFilesRawArgs holds the arguments of LsFilesRaw.
type LsFilesRawArgs struct {
	RepoArgs
	// Args are passed to git ls-files -z after -z.
	Args []string
}

// LsFilesRaw runs git ls-files in dir. See (*Repo).LsFilesRaw.
func LsFilesRaw(ctx context.Context, dir dt.DirPath, args *LsFilesRawArgs) ([]string, error) {
	return NewRepo(dir, nil).LsFilesRaw(ctx, args)
}

// LsFilesRaw runs git ls-files -z with args.Args and returns the paths it
// lists. It does not need Open. args may be nil.
func (r *Repo) LsFilesRaw(ctx context.Context, args *LsFilesRawArgs) (paths []string, err error) {
	var out string

	if args == nil {
		args = &LsFilesRawArgs{}
	}
	out, err = r.run(ctx, &args.RepoArgs, runArgs{
		NULSeparated: true,
		Command:      append([]string{"ls-files", "-z"}, args.Args...),
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
	var gitignorePath = dt.FilepathJoin(repoRoot, ".gitignore")

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
	var infoDir dt.DirPath

	infoDir, err = InfoDir(context.Background(), repoRoot, nil)
	if err != nil {
		err = NewErr(ErrAppendToGitExclude, err)
		goto end
	}
	excludePath = dt.FilepathJoin(infoDir, "exclude")
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
	defer func() {
		err = CombineErrs([]error{err, f.Close()})
	}()

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
