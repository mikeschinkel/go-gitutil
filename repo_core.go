package gitutil

import (
	"context"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// Repo is a git repository and the settings its git processes run with.
//
// Build one with NewRepo, which runs no git. Methods that need only the
// directory work at once; methods that read the current branch or its
// upstream need Open first, and say so in their doc comments.
type Repo struct {
	// Root is the directory git runs in. NewRepo sets it to the directory it
	// is given; Open replaces it with the repository's top level.
	Root dt.DirPath
	// Branch is the current branch, set by Open. Empty when HEAD is
	// detached.
	Branch dt.GitBranchName
	// Remote is the remote and branch the current branch tracks, set by
	// Open. Zero when it tracks none.
	Remote GitRemote

	args   RepoArgs
	opened bool
}

// NewRepo returns a Repo for dir whose git processes run with args, which
// may be nil. It runs no git.
func NewRepo(dir dt.DirPath, args *RepoArgs) *Repo {
	r := &Repo{Root: dir}
	if args != nil {
		r.args = *args
		r.args.Env = append([]string{}, args.Env...)
	}
	return r
}

// OpenArgs holds the arguments of Open.
type OpenArgs struct {
	RepoArgs
}

// Open returns the opened Repo for the repository dir is in, built with
// args' RepoArgs, so the Repo keeps them. args may be nil.
func Open(ctx context.Context, dir dt.DirPath, args *OpenArgs) (r *Repo, err error) {
	if args == nil {
		args = &OpenArgs{}
	}
	r = NewRepo(dir, &args.RepoArgs)
	err = r.Open(ctx, nil)
	if err != nil {
		r = nil
	}
	return r, err
}

// Open finds the top level of the repository r.Root is in and makes it
// r.Root, then reads the current branch and the remote it tracks into
// r.Branch and r.Remote. args may be nil.
func (r *Repo) Open(ctx context.Context, args *OpenArgs) (err error) {
	var out string
	var root dt.DirPath
	var call *RepoArgs
	var dir = r.Root

	if args != nil {
		call = &args.RepoArgs
	}

	out, err = r.run(ctx, call, runArgs{
		Command: []string{"rev-parse", "--show-toplevel"},
	})
	if err != nil {
		err = NewErr(ErrNotGitRepo, err)
		goto end
	}
	root = dt.DirPath(strings.TrimSpace(out))
	if root == "" {
		err = NewErr(ErrNotGitRepo)
		goto end
	}
	r.Root = root

	r.Branch, err = r.currentBranch(ctx, call)
	if err != nil {
		goto end
	}

	r.Remote, err = r.currentRemote(ctx, call, r.Branch)
	if err != nil {
		goto end
	}
	r.opened = true

end:
	if err != nil {
		err = WithErr(err, "dir", dir)
	}
	return err
}

// Run runs git with gitArgs and the Repo's settings in r.Root, and returns
// its output with trailing line endings trimmed. It is for git commands
// go-gitutil has no method for. Output is captured, not streamed to StdIO.
// It does not need Open.
func (r *Repo) Run(ctx context.Context, gitArgs ...string) (out string, err error) {
	return r.runGit(ctx, gitArgs...)
}

// RevParse returns the object ID rev names. It does not need Open.
func (r *Repo) RevParse(ctx context.Context, rev dt.GitRevision) (id dt.GitObjectID, err error) {
	var out string
	out, err = r.runGit(ctx, "rev-parse", "--verify", "--end-of-options", string(rev))
	if err != nil {
		goto end
	}
	id = dt.GitObjectID(strings.TrimSpace(out))
end:
	return id, err
}

// CurrentBranch returns the name of the branch HEAD points to, or an empty
// name when HEAD is detached. It reads git, not r.Branch, so it does not need
// Open. It works in a repository with no commits yet.
func (r *Repo) CurrentBranch(ctx context.Context) (dt.GitBranchName, error) {
	return r.currentBranch(ctx, nil)
}

// requireOpen returns ErrRepoNotOpen unless Open has succeeded on r.
func (r *Repo) requireOpen() (err error) {
	if !r.opened {
		err = NewErr(ErrRepoNotOpen, "dir", r.Root)
	}
	return err
}

// withArgs returns a copy of r whose settings are r's combined with call's,
// for a method that runs several git processes with a call's RepoArgs.
func (r *Repo) withArgs(call *RepoArgs) *Repo {
	c := *r
	c.args = r.args.combine(call)
	return &c
}

// run runs one git process with the Repo's settings combined with call's,
// in r.Root unless args.Dir is set.
func (r *Repo) run(ctx context.Context, call *RepoArgs, args runArgs) (string, error) {
	if args.Dir == "" {
		args.Dir = r.Root
	}
	return run(ctx, r.args.combine(call), args)
}

// runGit runs git with gitArgs and the Repo's settings in r.Root, capturing
// its output.
func (r *Repo) runGit(ctx context.Context, gitArgs ...string) (string, error) {
	return r.run(ctx, nil, runArgs{Command: gitArgs})
}

// streamGit runs git with gitArgs and the Repo's settings in r.Root,
// streaming its output to StdIO when it is set, for commands that print for
// a person.
func (r *Repo) streamGit(ctx context.Context, gitArgs ...string) (string, error) {
	return r.run(ctx, nil, runArgs{Command: gitArgs, Stream: true})
}

// currentBranch returns the branch HEAD points to, or an empty name when
// HEAD is detached.
func (r *Repo) currentBranch(ctx context.Context, call *RepoArgs) (branch dt.GitBranchName, err error) {
	var out string

	out, err = r.run(ctx, call, runArgs{
		Command: []string{"symbolic-ref", "--quiet", "--short", "HEAD"},
	})
	if exitCode(err) == 1 {
		// Detached HEAD.
		err = nil
		goto end
	}
	if err != nil {
		goto end
	}
	branch = dt.GitBranchName(strings.TrimSpace(out))
end:
	return branch, err
}

// currentRemote returns the remote and remote branch branch tracks, read from
// git config branch.<branch>.remote and branch.<branch>.merge, or a zero
// GitRemote when it tracks none: a branch that was never pushed, or a
// detached HEAD (an empty branch). A missing upstream is not an error.
func (r *Repo) currentRemote(ctx context.Context, call *RepoArgs, branch dt.GitBranchName) (remote GitRemote, err error) {
	var name, merge string
	var remoteBranch dt.GitBranchName

	if branch == "" {
		goto end
	}
	name, err = r.getConfig(ctx, call, "branch."+string(branch)+".remote")
	if err != nil {
		err = ignoreConfigNotSet(err)
		goto end
	}
	merge, err = r.getConfig(ctx, call, "branch."+string(branch)+".merge")
	if err != nil {
		err = ignoreConfigNotSet(err)
		goto end
	}
	remoteBranch, err = dt.GitRef(merge).ToBranchName()
	if err != nil {
		goto end
	}
	remote = GitRemote{
		Name:   dt.GitRemoteName(name),
		Branch: remoteBranch,
	}
end:
	return remote, err
}
