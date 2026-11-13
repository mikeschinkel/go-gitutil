# Synthesis: what git operations go-gitutil provides for fixtures

Brainstorm with Mike, 2026-11-11 and 2026-11-12, with answers relayed from the
gomion (E-2546) and go-fsfix (E-2547/E-2568) sessions. Everything below is
Mike's choice unless marked "proposed" (my proposal he accepted) or "fact".

**This outcome is our best current understanding, not a specification.** It
records what we knew and chose on 2026-11-12. Implementation (E-2600, E-2602,
E-2568, E-2559) will learn things this discussion could not, and when it does
the design evolves and this outcome is revised. Treat a deviation discovered
during implementation as information, not as a breach.

## The operation set

go-gitutil gains four operations for fixtures and gomion, all on `Repo`:

- `Init(ctx, InitArgs{Branch, Identity})`: `git init -b <branch>`; writes
  `user.name`/`user.email` into the new repo's local config when Identity is set.
- `Clone(ctx, src, CloneArgs{Local, NoCheckout, Branch, Bare, Identity})`:
  exported; cached_worktree.go calls it instead of its private clone. Identity
  is written to the clone's local config, since a clone does not copy the
  source's local config.
- `Add(ctx, AddArgs{Paths, Exclude, All, Force})`: Exclude becomes git pathspec
  magic `:(exclude)<dir>/`, so git filters and no file list is built in Go.
  Force is an ordinary flag, off by default, like git's.
- `Commit(ctx, CommitArgs{Message, Identity, AllowEmpty})`: Identity is passed
  as `GIT_AUTHOR_*`/`GIT_COMMITTER_*` env for that one process.

Nothing else is new. `RevParse` already reads HEAD. Reset and checkout were
proposed by the E-2568 session for RepoFixture.Reconcile and withdrawn: a
fixture gets clean git state by being recreated (Cleanup then Create), which
measured 41 ms for 10 files and 412 ms for 10,000. Reconcile is disk-only and
never runs git.

## The shape of Repo (Q1, answered via E-2546 and here)

- `NewRepo(dir, RepoArgs) *Repo` constructs with no I/O. Mike: "a NewRepo()
  allows ALWAYS getting a Repo value which can be important for testing or
  other use-cases." He accepts that every other method guards against an
  unopened Repo. His further point: `Repo{}` is always constructible in Go, so
  package-level constructors never protected against an uninitialized value.
- `(*Repo).Open(ctx, OpenArgs) error` does what OpenRepo does today. `Open`,
  not `Initialize`: a type with both `Init` (git init) and `Initialize` (read
  state) is a trap, and people look for Open.
- Package-level `gitutil.Open(ctx, dir, OpenArgs)`, `gitutil.Init(ctx, dir,
  InitArgs)` and `gitutil.Clone(ctx, src, dst, CloneArgs)` each construct a
  Repo and call the method, so the implementation lives once. Both styles are
  offered so callers pick what fits.
- `OpenRepo` is removed, no alias.
- `RepoArgs` holds what describes the Repo for its life: `StdIO dt.StdIO` and
  `Identity`. `OpenArgs` holds what is about the act of opening; empty today,
  kept for evolution.

## Identity (Q2)

- Identity is optional on `RepoArgs`, `InitArgs`, `CloneArgs` and
  `CommitArgs`. Set, it is applied (local config at init/clone, env at
  commit). Unset, go-gitutil passes nothing and git resolves it itself (env,
  repo config, global config, then git's own "Please tell me who you are"
  error). go-gitutil does not pre-check for a missing identity: that would
  refuse on every machine whose identity lives in the user-level git config.
- A strict mode that refuses to fall back to the machine's config is an
  opt-in boolean on the args, never the default.
- go-gitutil bakes in no fixture defaults. The constants are go-fsfix's
  (Q5 → go-fsfix), exported and overridable per fixture:

  | Value | Default |
  |---|---|
  | Author/committer name | `FSFix Fixture` |
  | Author/committer email | `fsfix@example.com` (proposed) |
  | Initial commit message | `Initial commit message (fsfix)` |
  | Default branch | `main` |
  | Force add | on |

  Default branch is a real gap today: gomion's test leaves it to the
  machine's `init.defaultBranch`. Force is on in fixtures because the threat
  is the developer's global excludes file (`core.excludesFile`): a global
  ignore of `.endless/` would silently drop fixture files on one machine only.

## Streamer, Repo and how git runs (Q3, Q4)

- One type. `Repo` holds state plus the optional `dt.StdIO` from RepoArgs.
  `Streamer` is deleted along with the package-level `Commit`,
  `StageModuleFiles` and `UnstageAll` wrappers, which only buffered Streamer's
  output. Mike: streaming is for future callers, not nobody; both benefits in
  one type.
- The rule: operations whose output the package parses (status, rev-parse,
  log, diff) always capture, regardless of StdIO. Operations whose output is
  for a human (commit, add, push, clone, checkout) write to StdIO when set and
  otherwise buffer and return it.
- One run function behind GitRunner, Repo and the creation operations,
  taking ctx, dir, env, stdin, `dt.StdIO` and args; raw `exec.Command`
  disappears from OpenRepo and Streamer. (Fact: today there are three paths,
  GitRunner.runGit in run_git.go buffered with stderr wrapped, OpenRepo in repo.go raw with stderr discarded,
  Streamer.Commit in operations.go raw streaming with no ctx.)

## Go-module knowledge leaves go-gitutil

`StageModuleFiles` finds nested `go.mod` files and stages around them. Mike:
"Go module logic should definitely not be in go-gitutil, but instead in
go-goutil." So:

- `FindNestedModules` moves to go-goutil. gomion's `gommod/goutils` becomes
  go-pkgs/go-goutil, package `goutil`; that extraction is a
  gomion task of its own (the package also holds AST diff, API compatibility,
  dependency analysis).
- go-gitutil keeps the git-only pieces: `UnstageAll`, `StageFileContent`
  (hash-object + update-index), and the new `Add` with `Exclude`.
- gomion composes: `goutil.FindNestedModules`, then `repo.UnstageAll`, then
  `repo.Add(AddArgs{Paths: ["."], Exclude: nested})`.

## Naming

- The package renames from `gitutils` to `gitutil`. Mike: standardize on
  singular `util` across go-pkgs, including renaming go-diffutils. go-diffutils
  is not an Endless project; Mike files that where he tracks it.
- Git domain types (branch name, revision, object id) live in go-dt's root
  package with a `Git` prefix (`GitRefName`, `GitRevision`, `GitObjectID`),
  filed as E-2599 in go-dt. go-gitutil's `GitRef` and `RemoteName` move to
  those types in its implementation task.

## Tests (Q6)

- go-gitutil gets a `test/` module per ED-1650; today it has none and its one
  test, diff_parser_test.go, sits beside the code.
- The test module dogfoods go-fsfix repo fixtures. The build-order friction
  (go-gitutil's tests need a go-fsfix built against the new API) is a tooling
  problem, not an architecture one. Existing tasks: gomion E-2561 (command
  managing go.work across directories) and endless E-1192 (go.work generation
  as a standalone tool).

## The negative test cases (formerly "decoy" files)

gomion's repo_setup_test.go fixture places `go.mod` files at
`dtx/testdata/x/go.mod` and `.endless/worktrees/a/go.mod`, with module paths
literally `ignored/testdata` and `ignored/endless`. They are negative test
cases: gomion's layout classifier must not treat them as modules. They must be
committed because gomion reads tracked files only ("An untracked go.mod is
invisible to CI, so it is invisible here too"), which is why the fixture adds
with Force. "Decoy" was the E-2546 session's word; negative test cases is the
clear one.

## Follow-ups filed

- E-2600 (go-gitutil): Reshape Repo with Init, Clone, Add and Commit. Cleans up
  E-2548; blocks E-2568, E-2559 and E-2556; relates to E-2599. Its analysis
  holds the API shape and the identity/StdIO rules above, as task-level
  decisions.
- E-2602 (go-gitutil): Create go-gitutil's test module on go-fsfix fixtures.
  Blocked by E-2600 and E-2568.
- go-goutil extraction, three parts: Mike created and registered the
  go-goutil project on 2026-11-12; E-2608 (go-goutil) extracts gommod/goutils
  into it as package goutil; E-2601 (gomion) then replaces the local copy with
  the import. E-2608 blocks E-2601.
- E-2607 (gomion, research): Survey go-pkgs for plural util names to rename,
  filing a rename task in each project that needs one (go-diffutils is a
  registered Endless project, so its rename task can be filed there directly).
- Decision ED-1662 (gomion): go-pkgs package names use singular util. Four
  decisions first filed in go-gitutil (ED-1658 to ED-1661) were rejected as
  task-level or misplaced; their content lives in E-2600's analysis and here.
- For Mike: revise E-2556's description. go-gitutil already has Commit,
  UnstageAll and InfoStore, and StageModuleFiles is removed rather than carried
  over, so its remaining work is deleting gomion's copy and switching imports;
  the call-site rewrite is E-2601's.
- E-2609 (endless): rename the `--git-hooks none` choice on project init, which
  installs a plain Git hook rather than nothing.
