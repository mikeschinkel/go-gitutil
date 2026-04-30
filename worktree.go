package gitutils

import (
	"bufio"
	"context"
	"errors"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// Worktree represents one entry from `git worktree list --porcelain`.
type Worktree struct {
	// Path is the worktree's working-tree root.
	Path dt.DirPath
	// Branch is the branch checked out in the worktree (e.g. "refs/heads/main").
	// Empty if the worktree is detached.
	Branch GitRef
	// HEAD is the commit SHA the worktree is currently at.
	HEAD string
	// Bare is true for bare repos.
	Bare bool
	// Detached is true for detached-HEAD worktrees.
	Detached bool
	// Locked is true if the worktree is locked (`git worktree lock`).
	Locked bool
	// LockReason is the reason given to `git worktree lock --reason`, if any.
	LockReason string
	// Prunable is true if `git worktree list --porcelain` reports this
	// worktree as prunable (filesystem missing, etc.).
	Prunable bool
	// PrunableReason is the explanation given by porcelain output.
	PrunableReason string
}

// AddWorktreeArgs provides arguments for AddWorktree.
type AddWorktreeArgs struct {
	// Path is where the new worktree's working tree will live.
	Path dt.DirPath
	// Branch is the new branch name to create (e.g. "task/969-stable-paths").
	// If empty, AddWorktree returns an error (use AddDetachedWorktree for detached HEAD).
	Branch string
	// Base is the ref to branch from (e.g. "main", "HEAD", or a commit SHA).
	Base string
	// Force, when true, allows creating the worktree even if the path already
	// exists or the branch is already checked out elsewhere. Use sparingly.
	Force bool
}

var ErrAddWorktreeBranchRequired = errors.New("AddWorktree: Branch is required (use AddDetachedWorktree for detached HEAD)")

// AddWorktree runs `git worktree add -b <branch> <path> <base>` from the
// repository root. The new worktree is created at args.Path with a fresh
// branch named args.Branch starting from args.Base.
func (r *Repo) AddWorktree(ctx context.Context, args AddWorktreeArgs) (err error) {
	if args.Branch == "" {
		err = ErrAddWorktreeBranchRequired
		goto end
	}

	{
		cmd := []string{"worktree", "add"}
		if args.Force {
			cmd = append(cmd, "--force")
		}
		cmd = append(cmd, "-b", args.Branch, string(args.Path))
		if args.Base != "" {
			cmd = append(cmd, args.Base)
		}
		_, err = r.runGit(ctx, r.Root, cmd...)
	}
end:
	return err
}

// AddDetachedWorktree runs `git worktree add --detach <path> <ref>` from
// the repository root.
func (r *Repo) AddDetachedWorktree(ctx context.Context, path dt.DirPath, ref string) (err error) {
	cmd := []string{"worktree", "add", "--detach", string(path)}
	if ref != "" {
		cmd = append(cmd, ref)
	}
	_, err = r.runGit(ctx, r.Root, cmd...)
	return err
}

// RemoveWorktreeArgs provides arguments for RemoveWorktree.
type RemoveWorktreeArgs struct {
	// Path is the worktree's working-tree root to remove.
	Path dt.DirPath
	// Force, when true, removes the worktree even if it has uncommitted
	// changes or is locked.
	Force bool
}

// RemoveWorktree runs `git worktree remove <path>` from the repository root.
func (r *Repo) RemoveWorktree(ctx context.Context, args RemoveWorktreeArgs) (err error) {
	cmd := []string{"worktree", "remove"}
	if args.Force {
		cmd = append(cmd, "--force")
	}
	cmd = append(cmd, string(args.Path))
	_, err = r.runGit(ctx, r.Root, cmd...)
	return err
}

// ListWorktrees returns all worktrees of this repository, parsed from
// `git worktree list --porcelain`. Includes the main checkout itself.
func (r *Repo) ListWorktrees(ctx context.Context) (worktrees []Worktree, err error) {
	var out string

	out, err = r.runGit(ctx, r.Root, "worktree", "list", "--porcelain")
	if err != nil {
		goto end
	}
	worktrees = parseWorktreePorcelain(out)
end:
	return worktrees, err
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain`.
//
// Format: empty-line-separated stanzas, each with key-value pairs:
//
//	worktree /path/to/checkout
//	HEAD <sha>
//	branch refs/heads/<branch>     (or "detached")
//	bare                            (only for bare repos)
//	locked [<reason>]               (only if locked)
//	prunable [<reason>]             (only if prunable)
func parseWorktreePorcelain(s string) (worktrees []Worktree) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	var current *Worktree
	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}
		if current == nil {
			current = &Worktree{}
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = dt.DirPath(strings.TrimPrefix(line, "worktree "))
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = GitRef(strings.TrimPrefix(line, "branch "))
		case line == "detached":
			current.Detached = true
		case line == "bare":
			current.Bare = true
		case line == "locked":
			current.Locked = true
		case strings.HasPrefix(line, "locked "):
			current.Locked = true
			current.LockReason = strings.TrimPrefix(line, "locked ")
		case line == "prunable":
			current.Prunable = true
		case strings.HasPrefix(line, "prunable "):
			current.Prunable = true
			current.PrunableReason = strings.TrimPrefix(line, "prunable ")
		}
	}
	flush()
	return worktrees
}
