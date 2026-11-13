package gitutil

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

// FindRepoRootArgs holds the arguments of FindRepoRoot.
type FindRepoRootArgs struct {
	RepoArgs
}

// FindRepoRoot returns the top level of the repository dir is in.
func FindRepoRoot(ctx context.Context, dir dt.DirPath, args *FindRepoRootArgs) (dt.DirPath, error) {
	return NewRepo(dir, nil).FindRepoRoot(ctx, args)
}

// FindRepoRoot returns the top level of the repository r.Root is in. It does
// not need Open. args may be nil.
func (r *Repo) FindRepoRoot(ctx context.Context, args *FindRepoRootArgs) (root dt.DirPath, err error) {
	var out string
	var call *RepoArgs
	var abs dt.DirPath

	if args != nil {
		call = &args.RepoArgs
	}
	abs, err = r.Root.Abs()
	if err != nil {
		goto end
	}
	out, err = r.run(ctx, call, runArgs{
		Dir:     abs,
		Command: []string{"rev-parse", "--show-toplevel"},
	})
	if err != nil {
		goto end
	}
	root = dt.DirPath(out).TrimSpace()
end:
	return root, err
}
