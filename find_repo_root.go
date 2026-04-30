package gitutils

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

func FindRepoRoot(ctx context.Context, dp dt.DirPath) (_ dt.DirPath, err error) {
	var out string
	var abs dt.DirPath

	abs, err = dp.Abs()
	if err != nil {
		goto end
	}
	out, err = newGitRunner(abs).runGit(ctx, runGitArgs{
		Command: []string{"rev-parse", "--show-toplevel"},
	})
	if err != nil {
		goto end
	}
	dp = dt.DirPath(out).TrimSpace()
end:
	return dp, err
}
