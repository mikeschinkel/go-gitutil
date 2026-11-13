package gitutil

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// GetStagedFiles returns the list of files currently staged in the index.
// It does not need Open.
func (r *Repo) GetStagedFiles(ctx context.Context) (files []dt.RelFilepath, err error) {
	var out string
	var lines []string

	// Get list of staged files using --cached --name-only
	out, err = r.runGit(ctx, "diff", "--cached", "--name-only")
	if err != nil {
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, dt.RelFilepath(line))
	}

end:
	return files, err
}

// ExportStagedArgs contains arguments for ExportStagedFiles
type ExportStagedArgs struct {
	RepoArgs
	DestDir      dt.DirPath
	ModuleRelDir dt.PathSegments // Optional: filter to files within this module directory
}

// ExportStagedFiles exports the staged files of the repository in dir. See
// (*Repo).ExportStagedFiles.
func ExportStagedFiles(ctx context.Context, dir dt.DirPath, args *ExportStagedArgs) error {
	return NewRepo(dir, nil).ExportStagedFiles(ctx, args)
}

// ExportStagedFiles exports staged file contents to a destination directory
// using `git show :path` to get the staged version of each file.
//
// This is critical for pre-commit analysis: we need to analyze what WILL be
// committed (staged version) not what's in the working directory (which may
// have unstaged changes).
//
// Edge cases handled:
// - Deleted files: git show fails, we skip them (expected)
// - New files: works correctly (they're in index)
// - Renamed files: appears as delete + add (both handled correctly)
// - Partial staging: returns staged version only (correct)
// - Binary files: copied as-is
//
// It does not need Open.
func (r *Repo) ExportStagedFiles(ctx context.Context, args *ExportStagedArgs) (err error) {
	var files []dt.RelFilepath
	var modulePrefix string
	var repo *Repo

	if args == nil {
		args = &ExportStagedArgs{}
	}
	if args.DestDir == "" {
		err = fmt.Errorf("destination directory cannot be empty")
		goto end
	}

	// The call's RepoArgs apply to every git process this export runs.
	repo = r.withArgs(&args.RepoArgs)

	// Get list of staged files
	files, err = repo.GetStagedFiles(ctx)
	if err != nil {
		goto end
	}

	// Set up module path prefix for filtering if specified
	if !args.ModuleRelDir.IsRoot() {
		modulePrefix = string(args.ModuleRelDir) + "/"
	}

	// Export each staged file
	for _, file := range files {
		var shouldExport bool
		var fileStr string
		var destPath dt.Filepath
		var content []byte

		fileStr = string(file)

		// Filter to module directory if specified
		if modulePrefix != "" {
			shouldExport = strings.HasPrefix(fileStr, modulePrefix)
			if !shouldExport {
				continue
			}
		}

		// Use `git show :path` to get staged version
		// The :path syntax means "the version in the index"
		content, err = repo.gitShowStaged(ctx, fileStr)
		if err != nil {
			// File might be deleted (staged for deletion)
			// This is expected - skip it and continue
			err = nil
			continue
		}

		// Build destination path preserving directory structure
		destPath = dt.FilepathJoin(args.DestDir, dt.RelFilepath(fileStr))

		// Create parent directory if it doesn't exist
		err = destPath.Dir().MkdirAll(0o755)
		if err != nil {
			goto end
		}

		// Write file content
		err = os.WriteFile(string(destPath), content, 0o644)
		if err != nil {
			goto end
		}
	}

end:
	return err
}

// gitShowStaged uses `git show :path` to get the staged version of a file
func (r *Repo) gitShowStaged(ctx context.Context, relPath string) (content []byte, err error) {
	var cmd = fmt.Sprintf(":%s", relPath)
	var out string

	out, err = r.runGit(ctx, "show", cmd)
	if err != nil {
		goto end
	}

	content = []byte(out)

end:
	return content, err
}

// GetStagedDiff returns the diff of staged changes. It does not need Open.
func (r *Repo) GetStagedDiff(ctx context.Context) (diff string, err error) {
	diff, err = r.runGit(ctx, "diff", "--cached")
	return diff, err
}

// FindBaselineTag finds the most recent reachable semver tag to use as a baseline
// for comparison. Uses git describe for O(1) lookup instead of iterating all tags.
// It does not need Open.
func (r *Repo) FindBaselineTag(ctx context.Context, moduleRelDir dt.PathSegments) (tag dt.GitTagName, err error) {
	var matchPattern string
	var prefix string
	var out string

	if moduleRelDir.IsRoot() {
		matchPattern = "v[0-9]*"
	} else {
		prefix = string(moduleRelDir)
		matchPattern = prefix + "/v[0-9]*"
	}

	// Single subprocess: git describe --tags --match <pattern> --abbrev=0 HEAD
	out, err = r.runGit(ctx, "describe", "--tags", "--match", matchPattern, "--abbrev=0", "HEAD")
	if err != nil {
		// TODO — This should ONLY return ErrNoSemverTags when the error indicates that, NOT FOR ALL ERRORS
		err = ErrNoSemverTags
		goto end
	}
	tag = dt.GitTagName(strings.TrimSpace(out))

end:
	return tag, err
}
