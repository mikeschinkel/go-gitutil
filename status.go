package gitutils

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

// StatusArgs contains optional arguments for Status() and StatusCounts()
type StatusArgs struct {
	Path          dt.PathSegments   // Optional path to scope status to
	ExcludePaths  []dt.PathSegments // Subpaths to exclude from counts (e.g. submodule dirs)
	HumanReadable bool              // If true, return human-readable format; if false, use --porcelain
}

// Status returns git status output (package-level function)
// By default returns machine-readable format (--porcelain)
// If args.HumanReadable is true, returns human-readable format
// If args.Path is provided, status is scoped to that path
func Status(ctx context.Context, root dt.DirPath, args *StatusArgs) (output string, err error) {
	var cmdArgs []string

	cmdArgs = []string{"status"}

	// Add --porcelain unless HumanReadable is true
	if args == nil || !args.HumanReadable {
		cmdArgs = append(cmdArgs, "--porcelain")
	}

	// Add path if provided
	if args != nil && args.Path != "" {
		cmdArgs = append(cmdArgs, string(args.Path))
	}

	output, err = newGitRunner(root).runGit(ctx, runGitArgs{Command: cmdArgs})
	return output, err
}
