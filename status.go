package gitutil

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

// StatusArgs contains optional arguments for Status() and StatusCounts()
type StatusArgs struct {
	RepoArgs
	Path          dt.PathSegments   // Optional path to scope status to
	ExcludePaths  []dt.PathSegments // Subpaths to exclude from counts (e.g. submodule dirs)
	HumanReadable bool              // If true, return human-readable format; if false, use --porcelain
}

// Status returns git status output for dir. See (*Repo).Status.
func Status(ctx context.Context, dir dt.DirPath, args *StatusArgs) (output string, err error) {
	return NewRepo(dir, nil).Status(ctx, args)
}

// Status returns git status output.
// By default returns machine-readable format (--porcelain).
// If args.HumanReadable is true, returns human-readable format, which also
// streams to StdIO when it is set.
// If args.Path is provided, status is scoped to that path.
// It does not need Open. args may be nil.
func (r *Repo) Status(ctx context.Context, args *StatusArgs) (output string, err error) {
	var cmd []string

	if args == nil {
		args = &StatusArgs{}
	}
	cmd = []string{"status"}
	if !args.HumanReadable {
		cmd = append(cmd, "--porcelain")
	}
	if args.Path != "" {
		cmd = append(cmd, "--", string(args.Path))
	}

	output, err = r.run(ctx, &args.RepoArgs, runArgs{
		Command: cmd,
		Stream:  args.HumanReadable,
	})
	return output, err
}
