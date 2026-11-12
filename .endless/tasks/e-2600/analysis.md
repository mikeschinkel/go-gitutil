From E-2548's outcome:
- Rename package gitutils -> gitutil.
- NewRepo(dir, RepoArgs) constructs without I/O; (*Repo).Open(ctx, OpenArgs) replaces OpenRepo (removed, no alias). RepoArgs{StdIO dt.StdIO, Identity}; OpenArgs empty for now.
- Repo methods Init(ctx, InitArgs{Branch, Identity}), Clone(ctx, src, CloneArgs{Local, NoCheckout, Branch, Bare, Identity}), Add(ctx, AddArgs{Paths, Exclude, All, Force}) with Exclude as ':(exclude)<dir>/' pathspecs, Commit(ctx, CommitArgs{Message, Identity, AllowEmpty}). Package-level Open/Init/Clone construct a Repo and call the method.
- Identity: local config at init/clone, GIT_AUTHOR_*/GIT_COMMITTER_* env at commit; unset means git resolves. Optional strict boolean refuses machine fallback; off by default. No baked-in defaults.
- One run function (ctx, dir, env, stdin, dt.StdIO, args) behind GitRunner, Repo and creation ops; parsed-output ops always capture, human-output ops stream to StdIO when set. Streamer and the package-level Commit/StageModuleFiles/UnstageAll wrappers deleted. StageModuleFiles removed (Go-module knowledge goes to go-goutil); UnstageAll and StageFileContent stay as Repo methods.
- cached_worktree.go uses the exported Clone.
- GitRef and RemoteName move to go-dt's Git-prefixed types once E-2599 lands.
