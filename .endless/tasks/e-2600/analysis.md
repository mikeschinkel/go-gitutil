From E-2548's outcome:
- Rename package gitutils -> gitutil.
- NewRepo(dir, RepoArgs) constructs without I/O; (*Repo).Open(ctx, OpenArgs) replaces OpenRepo (removed, no alias). RepoArgs{StdIO dt.StdIO, Identity}; OpenArgs empty for now.
- Repo methods Init(ctx, InitArgs{Branch, Identity}), Clone(ctx, src, CloneArgs{Local, NoCheckout, Branch, Bare, Identity}), Add(ctx, AddArgs{Paths, Exclude, All, Force}) with Exclude as ':(exclude)<dir>/' pathspecs, Commit(ctx, CommitArgs{Message, Identity, AllowEmpty}). Package-level Open/Init/Clone construct a Repo and call the method.
- Identity: local config at init/clone, GIT_AUTHOR_*/GIT_COMMITTER_* env at commit; unset means git resolves. Optional strict boolean refuses machine fallback; off by default. No baked-in defaults.
- One run function (ctx, dir, env, stdin, dt.StdIO, args) behind GitRunner, Repo and creation ops; parsed-output ops always capture, human-output ops stream to StdIO when set. Streamer and the package-level Commit/StageModuleFiles/UnstageAll wrappers deleted. StageModuleFiles removed (Go-module knowledge goes to go-goutil); UnstageAll and StageFileContent stay as Repo methods.
- cached_worktree.go uses the exported Clone.
- GitRef and RemoteName move to go-dt's Git-prefixed types once E-2599 lands.

# Requirements from go-fsfix (E-2568's plan, relayed 2026-11-12)

go-fsfix never execs git itself, so each of these must be possible through go-gitutil. Items marked NEW go beyond what E-2548 settled and are for Mike to confirm when planning this task.

1. Extra environment for every git process: go-fsfix sets GIT_CONFIG_GLOBAL=/dev/null and GIT_CONFIG_NOSYSTEM=1 so the machine's config is ignored. Fits the run function's env parameter; expose it on RepoArgs (per-Repo env) so every operation inherits it.
2. Init on a given branch, typed dt.GitBranchName (E-2599).
3. NEW: set repo-local config keys after init: commit.gpgsign=false, core.hooksPath="" and core.autocrlf=false. Needs a Repo.SetConfig(ctx, key, value) operation, or InitArgs.Config map. Identity-in-local-config (above) is the same mechanism.
4. Forced add of everything with advice.addEmbeddedRepo=false, so a nested repo becomes a gitlink without advice output. Add with All+Force exists; the advice flag is a -c argument or a config key.
5. NEW: Commit with separate author and committer (dt.Mailbox, E-2599) and a fixed author and committer date (time.Time), so commit hashes are deterministic across runs. CommitArgs grows Author, Committer, Date; Identity may become these two Mailboxes.
6. Clone a local repo into a sibling directory and check out a given branch. Clone with Branch exists.
7. Add a linked worktree on a given branch, creating the branch if needed. AddWorktree exists in worktree.go; confirm it takes -b.
8. NEW read-side: current branch (exists), log or commit count, status (exists), HEAD's hash (RevParse exists), a tree entry's mode via ls-tree (to see a gitlink), and repo-local config values via Repo.GetConfig.

No reset or checkout: a fixture gets clean git state by being recreated (41-95 ms for 10-1,000 files on git 2.52).
