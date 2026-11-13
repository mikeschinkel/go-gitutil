package gitutil

import (
	"context"

	"github.com/mikeschinkel/go-dt"
)

// InitArgs holds the arguments of Init.
type InitArgs struct {
	RepoArgs

	// Branch names the initial branch. Empty lets git choose it.
	Branch dt.GitBranchName

	// Identity, when set, writes user.name and user.email into the new
	// repository's local config. Its DisplayName must not be empty.
	Identity *dt.Mailbox

	// Config holds keys written into the new repository's local config,
	// in sorted order, right after git init. Identity's keys take
	// precedence over the same keys here.
	Config map[string]string
}

// Init creates a git repository in dir, which git creates if it does not
// exist, and returns a Repo for it built with args' RepoArgs, so the Repo
// keeps them. The Repo is not opened. args may be nil.
func Init(ctx context.Context, dir dt.DirPath, args *InitArgs) (r *Repo, err error) {
	if args == nil {
		args = &InitArgs{}
	}
	r = NewRepo(dir, &args.RepoArgs)
	err = r.Init(ctx, &InitArgs{
		Branch:   args.Branch,
		Identity: args.Identity,
		Config:   args.Config,
	})
	if err != nil {
		r = nil
	}
	return r, err
}

// Init creates a git repository in r.Root, which git creates if it does not
// exist, then writes args' Identity and Config into its local config. It
// does not need Open, and does not open r. args may be nil.
func (r *Repo) Init(ctx context.Context, args *InitArgs) (err error) {
	var cmd []string
	var config map[string]string

	if args == nil {
		args = &InitArgs{}
	}
	err = checkIdentity("Identity", args.Identity)
	if err != nil {
		goto end
	}

	cmd = []string{"init"}
	if args.Branch != "" {
		cmd = append(cmd, "--initial-branch="+string(args.Branch))
	}
	cmd = append(cmd, "--", string(r.Root))
	// git init creates r.Root, so it runs in the current directory.
	_, err = r.run(ctx, &args.RepoArgs, runArgs{
		Dir:     ".",
		Command: cmd,
		Stream:  true,
	})
	if err != nil {
		goto end
	}

	config = newRepoConfig(args.Config, args.Identity)
	for _, key := range sortedKeys(config) {
		err = r.setConfig(ctx, &args.RepoArgs, key, config[key])
		if err != nil {
			goto end
		}
	}
end:
	if err != nil {
		err = NewErr(ErrInit, "dir", r.Root, err)
	}
	return err
}

// CloneArgs holds the arguments of Clone.
type CloneArgs struct {
	RepoArgs

	// Local passes --local: a clone of a local repository hardlinks its
	// objects.
	Local bool
	// NoCheckout passes --no-checkout.
	NoCheckout bool
	// Bare passes --bare.
	Bare bool

	// Branch names the branch to check out. Empty means the source's HEAD.
	Branch dt.GitBranchName

	// Identity, when set, writes user.name and user.email into the new
	// repository's local config. Its DisplayName must not be empty.
	Identity *dt.Mailbox

	// Config holds keys passed as git clone -c key=value, in sorted order,
	// so they apply during the clone's fetch and are written into the new
	// repository's local config. Identity's keys take precedence over the
	// same keys here.
	Config map[string]string
}

// Clone clones src, a path or URL as git clone takes it, into dir and
// returns a Repo for dir built with args' RepoArgs, so the Repo keeps them.
// The Repo is not opened. args may be nil.
func Clone(ctx context.Context, src string, dir dt.DirPath, args *CloneArgs) (r *Repo, err error) {
	var call CloneArgs

	if args == nil {
		args = &CloneArgs{}
	}
	r = NewRepo(dir, &args.RepoArgs)
	call = *args
	call.RepoArgs = RepoArgs{}
	err = r.Clone(ctx, src, &call)
	if err != nil {
		r = nil
	}
	return r, err
}

// Clone clones src, a path or URL as git clone takes it, into r.Root. It
// does not need Open, and does not open r. args may be nil.
func (r *Repo) Clone(ctx context.Context, src string, args *CloneArgs) (err error) {
	var cmd []string
	var config map[string]string

	if args == nil {
		args = &CloneArgs{}
	}
	err = checkIdentity("Identity", args.Identity)
	if err != nil {
		goto end
	}

	cmd = []string{"clone"}
	if args.Local {
		cmd = append(cmd, "--local")
	}
	if args.NoCheckout {
		cmd = append(cmd, "--no-checkout")
	}
	if args.Bare {
		cmd = append(cmd, "--bare")
	}
	if args.Branch != "" {
		cmd = append(cmd, "--branch="+string(args.Branch))
	}
	config = newRepoConfig(args.Config, args.Identity)
	for _, key := range sortedKeys(config) {
		cmd = append(cmd, "--config="+key+"="+config[key])
	}
	cmd = append(cmd, "--", src, string(r.Root))

	// git clone creates r.Root, so it runs in the current directory.
	_, err = r.run(ctx, &args.RepoArgs, runArgs{
		Dir:     ".",
		Command: cmd,
		Stream:  true,
	})
end:
	if err != nil {
		err = NewErr(ErrClone, "source", src, "dir", r.Root, err)
	}
	return err
}
