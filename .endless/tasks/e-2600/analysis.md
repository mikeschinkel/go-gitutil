From E-2548's outcome:
- Rename package gitutils -> gitutil.
- NewRepo(dir, RepoArgs) constructs without I/O; (*Repo).Open(ctx, OpenArgs) replaces OpenRepo (removed, no alias). RepoArgs{StdIO dt.StdIO, Identity}; OpenArgs empty for now.
- Repo methods Init(ctx, InitArgs{Branch, Identity}), Clone(ctx, src, CloneArgs{Local, NoCheckout, Branch, Bare, Identity}), Add(ctx, AddArgs{Paths, Exclude, All, Force}) with Exclude as ':(exclude)<dir>/' pathspecs, Commit(ctx, CommitArgs{Message, Identity, AllowEmpty}). Package-level Open/Init/Clone construct a Repo and call the method.
- Identity: local config at init/clone, GIT_AUTHOR_*/GIT_COMMITTER_* env at commit; unset means git resolves. Optional strict boolean refuses machine fallback; off by default. No baked-in defaults.
- One run function (ctx, dir, env, stdin, dt.StdIO, args) behind GitRunner, Repo and creation ops; parsed-output ops always capture, human-output ops stream to StdIO when set. Streamer and the package-level Commit/StageModuleFiles/UnstageAll wrappers deleted. StageModuleFiles removed (Go-module knowledge goes to go-goutil); UnstageAll and StageFileContent stay as Repo methods.
- cached_worktree.go uses the exported Clone.
- Type mapping onto go-dt (E-2599, landed 2026-11-13): go-dt defines GitRef (full refname only), GitBranchName, GitTagName, GitRemoteName, GitRevision, GitObjectID, GitShortObjectID, plus the union GitRevisionTypes. Correction to the premise relayed from E-2568: every use of go-gitutil's GitRef today is a branch name (Repo.Branch, currentBranch, CurrentBranch, Worktree.Branch, GitRemote.Branch), so those map to dt.GitBranchName, not dt.GitRevision. The untyped string ref parameters (RevParse, isAncestor, DiffNameStatus, DiffStatRange, DiffNumStat, CommitDiff, CommitFileDiff, AddDetachedWorktree, CommitsOnBranch's branch, CommitsSinceTag/CommitsBetweenTags' tags) become dt.GitRevision, dt.GitBranchName or dt.GitTagName as each means. RemoteName maps to dt.GitRemoteName; InitArgs.Branch and CloneArgs.Branch are dt.GitBranchName; CreateTagArgs.TagName is dt.GitTagName; RevParse returns dt.GitObjectID. go-dt has no type for the joined origin/main form; currentRemote's split of `rev-parse --abbrev-ref @{upstream}` output on the first '/' is ambiguous when a remote or branch name contains '/'; replace it with `git config branch.<branch>.remote` (a GitRemoteName) and `git config branch.<branch>.merge` (a GitRef under refs/heads/, ToBranchName gives the GitBranchName), which need no split. Adopt dt ErrKV methods from the ErrKV task once it lands.

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

# E-2600 decisions (planning with Mike, 2026-11-13)

The analysis above these decisions (E-2548's outcome, the go-dt type
mapping and go-fsfix's requirements) still stands except where a decision
here changes it.

- **Q1 Commit identity and dates: B.** Separate `Author`, `Committer`,
  `AuthorDate` and `CommitterDate`, each a pointer. Nil `Committer` uses
  `Author`, and nil `CommitterDate` uses `AuthorDate`. Mike chose nil
  pointers over zero-value rules for struct-typed fields: nil means "not
  given" to every developer, while which property makes a struct count as
  unset has to be learned type by type.
- **Q2 Strict mode: D3, isolated.** Strict builds each git process's
  environment from nothing. Rejected: setting only `GIT_CONFIG_GLOBAL` and
  `GIT_CONFIG_NOSYSTEM` (D1, D2). SSH keys through the agent, askpass
  variables and `.netrc` would still reach git, and git would still guess
  an identity from the host name. Tested on Apple Git 2.50.1: under the
  isolated environment, https and ssh clones fail with "transport not
  allowed", a commit with no identity fails, a `file://` clone works, and
  `git config --list` is empty. The cost is that a strict Repo cannot reach
  network remotes, which is what isolation is for. Network credentials for
  deployment are E-2618's subject (go-secrets).
- **GitPath.** Strict requires an explicit git binary. One machine can
  have Apple's git (2.50.1) and Homebrew's (2.52.0), and the launcher in
  the system bin directory forwards to whichever git xcode-select names.
  All three were tested with no `PATH` and work. Choosing a binary by
  version constraint is E-2616's subject (go-binaries); go-gitutil only
  takes the path.
- **Q3 RepoArgs: B, embedded.** Mike's convention is one Args parameter,
  last. Options taking `RepoArgs` as a separate parameter broke it.
- **Q4 Config: A and B.** `SetConfig`/`GetConfig` methods, plus `Config`
  maps on `InitArgs` and `CloneArgs`. Clone's map goes on the clone command
  line (`-c`) so settings such as a credential helper apply during the
  fetch.
- **Q5 Existing worktree branch: A.** `ReuseBranch` states the intent; the
  rejected options either create a branch from a typo silently or reset an
  existing branch with `-B`.
- **Q6 Read side: C.** Typed `CommitCount` and `TreeEntry` for go-fsfix's
  known needs, plus exported `Run` for the rest.
- **Q7 Package-level functions: wrappers over methods, W2.** One
  implementation; zero `RepoArgs` means the current environment. W2 lets
  directory-only methods run without `Open`, so a wrapper call costs one
  git process instead of about four.
- **Q9 RepoArgs on both a Repo and a call: P1.** One Args type per
  operation, embedding `RepoArgs`; a call's non-zero `RepoArgs` combines
  with the Repo's under the rules in the plan.
- **RepoArgs has no Identity.** E-2548 put `Identity` on `RepoArgs`. With
  `RepoArgs` embedded in `InitArgs` and `CloneArgs`, which have their own
  `Identity`, the outer field would hide the embedded one. Identity lives
  on the operation Args.
