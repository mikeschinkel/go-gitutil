# E-2600 plan: reshape go-gitutil's Repo

go-gitutil has no importers and no tags, so every change below is made in
place, with no aliases or deprecated forms. Its first release is v0.1.0.

## Package and dependencies

- Rename the package `gitutils` to `gitutil`. The module path stays
  `github.com/mikeschinkel/go-gitutil`.
- Require go-dt v0.7.2. Bring every `go.mod` to `go 1.27.0` and lint clean
  with golangci-lint v2.15.0.
- Delete go-gitutil's own `GitRef` and `RemoteName`. Use go-dt's git types
  as E-2600's analysis maps them: branch names are `dt.GitBranchName`,
  revisions `dt.GitRevision`, tags `dt.GitTagName`, remotes
  `dt.GitRemoteName`, and `RevParse` returns `dt.GitObjectID`.
- Replace `currentRemote`'s split of `@{upstream}` on '/' with
  `git config branch.<branch>.remote` and `branch.<branch>.merge`.

## One run function

Every git process runs through one unexported function taking the
directory, environment, stdin, a `dt.StdIO` and the git arguments.
`GitRunner.runGit`, `OpenRepo`'s raw `exec.Command` and `Streamer` all go
through it or go away. Commands whose output go-gitutil parses always
capture it. Commands that print for a person stream to `StdIO` when it is
set.

`Repo.Run(ctx, gitArgs ...string) (string, error)` exports that function
for git commands go-gitutil has no method for, with the Repo's settings.

## RepoArgs

`RepoArgs` holds the settings that last for a Repo's whole life:

- `StdIO dt.StdIO`. Nil means output is captured and not shown.
- `Env []string` of `KEY=VALUE`, applied after the base environment.
- `GitPath dt.Filepath`. Empty means `git` is found on `PATH`.
- `Strict bool`. Isolated mode (below).

`NewRepo(dir dt.DirPath, args *RepoArgs) *Repo` builds a Repo without
running git. `(*Repo).Open(ctx, *OpenArgs)` replaces `OpenRepo`, which is
removed.

### Per-call RepoArgs (P1)

Every operation that exists both as a Repo method and as a package-level
function has one Args type, which embeds `RepoArgs` and is the last
parameter. When a method gets a non-zero embedded `RepoArgs`, it combines
it with the Repo's settings for that one call:

- `StdIO`: the call's value replaces the Repo's when it is non-nil.
- `Env`: the call's entries come after the Repo's, so a repeated key takes
  the call's value.
- `GitPath`: the call's value replaces the Repo's when it is non-empty.
- `Strict`: true turns isolated mode on for the call. A call cannot turn a
  strict Repo's isolation off.

`Open`, `Init` and `Clone` as package-level functions build their Repo
from the embedded `RepoArgs`, so the returned Repo keeps those settings.
Every other package-level function is `NewRepo(dir, nil)` followed by the
method.

### Isolated mode (Strict)

With `Strict`, every git process gets an environment built from nothing,
with none of the parent process's variables:

- `HOME` set to the path of an empty directory that go-gitutil creates and
  removes. Never an empty or unset `HOME`: tools such as ssh then fall back
  to the home directory in the user database;
- `GIT_CONFIG_GLOBAL=/dev/null` and `GIT_CONFIG_NOSYSTEM=1`;
- `GIT_ALLOW_PROTOCOL=file`, so only local remotes work;
- `GIT_TERMINAL_PROMPT=0`;
- then `RepoArgs.Env`.

Each process also gets `-c user.useConfigOnly=true`, so git never guesses
an identity from the user name and host name. Strict requires `GitPath`
and fails before running git when it is empty. `PATH` is not passed: git
run by full path needs none.

## Creating repos

- `(*Repo).Init(ctx, *InitArgs)` and `gitutil.Init(ctx, dir, *InitArgs)
  (*Repo, error)`. `InitArgs{RepoArgs; Branch dt.GitBranchName; Identity
  *dt.Mailbox; Config map[string]string}`. go-gitutil applies `Config`
  right after `git init` through the code `SetConfig` uses.
- `(*Repo).Clone(ctx, src, *CloneArgs)` and `gitutil.Clone(ctx, src, dir,
  *CloneArgs) (*Repo, error)`. `CloneArgs{RepoArgs; Local, NoCheckout,
  Bare bool; Branch dt.GitBranchName; Identity *dt.Mailbox; Config
  map[string]string}`. `Config` is passed as `git clone -c key=value`, so
  it applies during the clone's fetch and is written into the new repo.
- `Identity` writes `user.name` and `user.email` into the repo's local
  config. Config keys are applied in sorted order.
- `cached_worktree.go`'s `cloneLocalRepo` uses `Clone`.

## Config

`(*Repo).SetConfig(ctx, key, value string) error` and
`(*Repo).GetConfig(ctx, key string) (string, error)` read and write the
repo's local config.

## Add and Commit

- `(*Repo).Add(ctx, *AddArgs)`. `AddArgs{Paths []dt.RelFilepath; Exclude
  []dt.RelDirPath; All, Force bool}`. Each `Exclude` entry becomes a
  `:(exclude)<dir>/` pathspec.
- `(*Repo).Commit(ctx, *CommitArgs)`. `CommitArgs{Message string; Author,
  Committer *dt.Mailbox; AuthorDate, CommitterDate *time.Time;
  AllowEmpty bool}`, passed as `GIT_AUTHOR_*` and `GIT_COMMITTER_*`
  environment variables:
  - nil `Author`: nothing is passed, so git resolves it;
  - nil `Committer`: uses `Author`;
  - nil `AuthorDate`: git uses the current time;
  - nil `CommitterDate`: uses `AuthorDate`.
- go-gitutil refuses an identity (`Identity`, `Author`, `Committer`) whose
  `DisplayName` is empty, before running git.
- `Streamer`, the package-level `Commit`, `StageModuleFiles` and
  `UnstageAll` are deleted. `UnstageAll` and `StageFileContent` stay as
  Repo methods. `FindNestedModules` stays until E-2608 moves it.

## Methods without Open (W2)

Methods that need only the directory run on a Repo that has not been
opened. Methods that read the current branch or its upstream (the Repo's
`Branch` or `Remote`) return an error on an unopened Repo. Each method's
doc comment says whether it needs `Open`.

The package-level `Status`, `StageFiles`, `StageFileContent`,
`IgnoredFiles`, `LsFilesRaw`, `UnstageFiles`, `ExportStagedFiles`,
`FindRepoRoot` and `InfoDir` become Repo methods, and each package-level
form becomes a wrapper over its method. Each takes one Args type that
embeds `RepoArgs`, last. `AppendToGitignore` and `AppendToGitExclude`
run no git and stay as they are.

## Worktrees

`AddWorktreeArgs{Path dt.DirPath; Branch dt.GitBranchName; Base
dt.GitRevision; Force, ReuseBranch bool}`. With `ReuseBranch` set and the
branch existing, `AddWorktree` checks it out. Otherwise it creates the
branch with `-b`, as today.

## Read side

- `(*Repo).CommitCount(ctx, dt.GitRevision) (int, error)`.
- `(*Repo).TreeEntry(ctx, dt.GitRevision, dt.RelFilepath) (TreeEntry,
  error)`. `TreeEntry` holds the mode, object type, `dt.GitObjectID` and
  path. A gitlink has mode `160000`.

## Tests

The `test/` module covers each item above against real git repos created
in temporary directories, including a strict-mode test that sets a hostile
environment (`HOME` with a `.gitconfig`, `GIT_DIR`, `GIT_AUTHOR_NAME`) and
shows none of it reaches git.
