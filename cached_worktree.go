package gitutils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

type releaseFunc func() error

type CachedWorktree struct {
	Dir     dt.DirPath
	repoDir dt.DirPath
	release func() error
}

func (cw *CachedWorktree) Close() (err error) {
	if cw == nil {
		goto end
	}
	if cw.release == nil {
		goto end
	}
	err = cw.release()
	cw.release = nil
end:
	return err
}

var ErrNilCachedWorktree = errors.New("cached_worktree is nil")

func (cw *CachedWorktree) Checkout(ctx context.Context, ref string) (err error) {
	var runner GitRunner
	var args runGitArgs

	if cw == nil {
		err = NewErr(ErrNilCachedWorktree)
		goto end
	}
	err = ensureSafeCacheRepoDir(cw.repoDir)
	if err != nil {
		goto end
	}

	runner = newGitRunner(cw.repoDir)
	args = runGitArgs{}

	args.Command = []string{"checkout", "--force", ref}
	_, err = runner.runGit(ctx, args)
	if err != nil {
		goto end
	}
	args.Command = []string{"reset", "--hard", ref}
	_, err = runner.runGit(ctx, args)
	if err != nil {
		goto end
	}
	args.Command = []string{"clean", "-fdx"}
	_, err = runner.runGit(ctx, args)
	if err != nil {
		goto end
	}
end:
	return err
}

func ensureSafeCacheRepoDir(repoDir dt.DirPath) (err error) {
	var baseDir dt.DirPath
	var exists bool

	if repoDir == "" {
		err = fmt.Errorf("empty cache repo dir")
		goto end
	}
	baseDir, err = cacheBaseDir()
	if err != nil {
		goto end
	}
	baseDir, err = baseDir.Abs()
	if err != nil {
		goto end
	}
	repoDir, err = repoDir.Abs()
	if err != nil {
		goto end
	}
	if repoDir != baseDir && !repoDir.HasPrefix(baseDir.EnsureTrailSep()) {
		err = fmt.Errorf("refusing to run in non-cache directory: %s", repoDir)
		goto end
	}
	exists, err = dt.FilepathJoin(repoDir, ".git").Exists()
	if exists {
		err = fmt.Errorf("cache repo missing .git directory: %s", repoDir)
		goto end
	}
	if err != nil {
		err = fmt.Errorf("filesystem error accessing %s", repoDir)
		goto end
	}
end:
	return err
}

func openCachedWorktree(ctx context.Context, r *Repo) (wt *CachedWorktree, err error) {
	var cacheBase dt.DirPath
	var release releaseFunc

	cacheBase, err = cacheBaseDir()
	if err != nil {
		return nil, err
	}

	reposDir := dt.DirPathJoin(cacheBase, "repos")
	repoDir := dt.DirPathJoin(reposDir, repoCacheKey(r.Root))
	lockDir := dt.DirPathJoin3(cacheBase, "locks", repoCacheKey(r.Root)+".lock")

	err = ensureSameFilesystem(r.Root, cacheBase)
	if err != nil {
		goto end
	}

	release, err = acquireLock(lockDir)
	if err != nil {
		goto end
	}

	wt = &CachedWorktree{
		Dir:     repoDir,
		repoDir: repoDir,
		release: release,
	}
	err = reposDir.MkdirAll(0o755)
	if err != nil {
		goto end
	}

	_, err = dt.FilepathJoin(repoDir, DotGitPath).Stat()
	if err != nil {
		if !os.IsNotExist(err) {
			goto end
		}
		err = cloneLocalRepo(ctx, r.Root, repoDir)
		if err != nil {
			goto end
		}
	}

	err = ensureSafeCacheRepoDir(repoDir)
	if err != nil {
		goto end
	}

	err = refreshFromSource(ctx, repoDir, r.Root)
	if err != nil {
		goto end
	}

	_, err = newGitRunner(repoDir).runGit(ctx, runGitArgs{
		Command: []string{"clean", "-fdx"},
	})
	if err != nil {
		goto end
	}

end:
	if err != nil && wt != nil {
		err = CombineErrs([]error{wt.Close(), err})
	}
	return wt, err
}

func refreshFromSource(ctx context.Context, cache, source dt.DirPath) (err error) {
	// Always refresh from the source repo, ensuring we have up-to-date commits and tags.
	// This fetches branches into Remote-tracking refs to avoid updating any checked-out local branch.
	_, err = newGitRunner(cache).runGit(ctx, runGitArgs{
		Command: []string{"fetch", "--prune", "--tags", string(source),
			"+refs/heads/*:refs/remotes/source/*",
			"+HEAD:refs/remotes/source/HEAD",
		},
	})
	return err
}

func acquireLock(lockDir dt.DirPath) (fn releaseFunc, err error) {
	err = lockDir.Dir().MkdirAll(0o755)
	if err != nil {
		goto end
	}
	err = lockDir.MkdirAll(0o700)
	if err != nil {
		if os.IsExist(err) {
			err = fmt.Errorf("another run is in progress (lock exists at %s)", lockDir)
		}
		goto end
	}
	fn = func() error { return lockDir.Remove() }
end:
	return fn, err
}

func cacheBaseDir() (baseDir dt.DirPath, err error) {
	v := strings.TrimSpace(os.Getenv("NEXTVER_CACHE_DIR"))
	if v != "" {
		baseDir, err = dt.ParseDirPath(v)
		goto end
	}
	baseDir, err = dt.UserCacheDir()
	if err != nil {
		goto end
	}
end:
	return baseDir, err
}

// TODO: Can we not just use the module path?!?
func repoCacheKey(repoRoot dt.DirPath) string {
	base := repoRoot.Clean().Base()
	sum := sha256.Sum256([]byte(repoRoot))
	return fmt.Sprintf("%s-%s", sanitize(base), hex.EncodeToString(sum[:8]))
}

func sanitize(ps dt.PathSegment) string {
	s := strings.TrimSpace(string(ps))
	if s == "" {
		return "repo"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func cloneLocalRepo(ctx context.Context, src, dst dt.DirPath) (err error) {
	err = dst.Dir().MkdirAll(0o755)
	if err != nil {
		goto end
	}
	_, err = newGitRunner("").runGit(ctx, runGitArgs{
		Command: []string{"clone", "--local", "--no-checkout", string(src), string(dst)},
	})
end:
	return err
}
