# Open questions: which git operations go-gitutil provides for fixtures

Written 2026-11-03 from a survey of go-gitutil and gomion's gommod/gitutils. The requirements come from gomion's E-2546 brainstorm (see this task's analysis). The questions stay open for Mike to answer; nothing here is decided except the "Decided without asking" list at the end.

Terms:

- **creation operation**: a git operation that runs where no repository exists yet: `init` and `clone`.
- **identity**: the author and committer name and email a commit records.
- **Repo**: go-gitutil's existing type, returned by `OpenRepo(ctx, dir)`, which runs `git rev-parse --show-toplevel` and fails if `dir` is not already a repository.
- **Streamer**: go-gitutil's existing type holding Stdout and Stderr writers, with `Commit`, `StageModuleFiles` and `UnstageAll` methods that run git with its output wired to those writers.

## What the survey found

- go-gitutil and gomion's gommod/gitutils are the same package with small drift: four files differ (file_status.go, ignored_files.go, info_store.go, repo.go), go-gitutil additionally has worktree.go (AddWorktree, AddDetachedWorktree, RemoveWorktree, ListWorktrees), gomion additionally has the worktree test. The E-2546 plan's claim that go-gitutil lacks Commit, StageModuleFiles and UnstageAll was wrong; they are in operations.go in both copies.
- Missing from go-gitutil for the fixture requirements: `init` (none), an exported `clone` (cached_worktree.go runs `clone --local --no-checkout` internally), `add` with `--all --force`, a commit that takes an identity and `--allow-empty` (Streamer.Commit takes only a message and uses the machine's git config).
- go-gitutil runs git three different ways: `GitRunner.runGit` (buffers output, wraps errors with stderr), `Repo.runGit`, and raw `exec.Command` in OpenRepo and Streamer. Streamer methods take no context; every Repo method does.
- gomion calls the package-level `StageModuleFiles` and `UnstageAll` wrappers, which buffer Streamer's output rather than stream it, so nothing in gomion relies on streaming today.

## Q1. Where do creation operations live, given Repo requires an existing repository?

**A. Package-level functions that return a Repo**: `InitRepo(ctx, dir, InitArgs) (*Repo, error)` and `Clone(ctx, src, dst, CloneArgs) (*Repo, error)`, the way `OpenRepo` already works.
- Pros: a Repo never exists for a directory that is not a repository; the caller gets a Repo for the add and commit that follow.
- Cons: none.

**B. Methods on a Repo built by hand**: `(&Repo{Root: dir}).Init(ctx, args)`.
- Pros: every operation is a method.
- Cons: a Repo value exists whose Branch and Remote fields are unset, which every other method assumes OpenRepo filled in.

**C. A separate builder type** for fixtures that chains init, add and commit.
- Pros: a fixture reads as one expression.
- Cons: a second object to learn for three operations, and go-fsfix already is that builder.

Recommendation: A.

## Q2. How does a commit get its identity?

The fixture needs the identity fixed so a fixture is identical on every machine; Mike: constants by default, overridable by the test.

**A. Per call**: `CommitArgs{Message, Identity, AllowEmpty}` on `Repo.Commit`, identity passed through the `GIT_AUTHOR_*` and `GIT_COMMITTER_*` environment of that one git process.
- Pros: nothing persists; two commits in one test can carry different identities.
- Cons: a commit made by the code under test (gomion's own commit flow running inside the fixture) does not get the fixed identity, so a test asserting on author output still depends on the machine's git config.

**B. Per Repo**: an `Identity` field on Repo, applied to every write operation (commit, annotated tag).
- Pros: set once per fixture.
- Cons: two Repo values opened on the same directory behave differently; the identity is invisible on disk, so git run any other way (the code under test, a shell) does not see it.

**C. In the repository's local config at init**: `InitArgs{Branch, Identity}` writes `user.name` and `user.email` into the new repository's `.git/config`; Commit takes no identity.
- Pros: every later git invocation in that repository uses it, including the code under test and a developer inspecting the fixture by hand; nothing to thread through go-fsfix's API beyond InitArgs.
- Cons: a test of behaviour that reads or writes repository config sees the fixture's entries; clone does not copy local config, so a cloned fixture needs its own identity set.

Recommendation: C, with A's per-call field added only when a test needs two authors (none does today).

## Q3. What becomes of Streamer once Repo has Commit?

**A. Keep Streamer; add Repo.Commit beside it.**
- Pros: gomion's callers compile unchanged.
- Cons: two commit implementations in one package, one taking context and an identity and one taking neither; this is the duplication E-2546 set out to remove, now inside one package.

**B. Fold Streamer's three operations into Repo methods that return their output; delete Streamer.**
- Pros: one implementation and one style (context, Repo receiver, buffered output with stderr in the error).
- Cons: output arrives when the operation finishes rather than as git writes it; no caller streams today, so the cost is only to a future one.

**C. Delete Streamer, give Repo optional output writers that write operations honour when set.**
- Pros: one implementation, streaming still possible.
- Cons: read operations ignore the writers, so the field's effect depends on which method is called.

Recommendation: B.

## Q4. How do the three ways of running git converge?

**A. Leave them; new operations use Repo.runGit.**
- Pros: smallest change.
- Cons: creation operations have no Repo yet, so they need GitRunner or a fourth path; error wrapping and context handling keep differing by operation.

**B. One package-level run function** taking context, directory, environment, stdin and arguments, called by GitRunner, Repo and the creation operations; raw exec.Command disappears from OpenRepo and Streamer.
- Pros: one place for error wrapping with stderr, context cancellation and environment; the creation operations fit without a Repo.
- Cons: touches every operation file in a package whose own tests cover only diff parsing, so regressions in the untouched behaviour would go unnoticed until gomion's tests run.

Recommendation: B.

## Q5. Where do the fixture's default branch and identity constants live?

**A. In go-fsfix.** go-gitutil's InitArgs zero values mean "git's own defaults", nothing baked in.
- Pros: a production caller that forgets identity gets the machine's git config, the same as running git by hand.
- Cons: every package that wants deterministic fixtures without go-fsfix declares its own constants.

**B. In go-gitutil**, as exported defaults applied when InitArgs fields are empty.
- Pros: one declaration shared by go-fsfix and anyone else.
- Cons: a production InitRepo that forgets identity silently commits as the fixture identity.

Recommendation: A.

## Q6. Do go-gitutil's own tests use go-fsfix repo fixtures?

A go-gitutil `test/` module importing go-fsfix, which imports go-gitutil, is legal (the test module is separate) but the question is whether it is wise.

**A. Yes.**
- Pros: go-gitutil dogfoods the fixture its operations exist for.
- Cons: a new operation cannot be tested through go-fsfix until go-fsfix has been built against it, so go-gitutil's test module must `replace` go-fsfix with the sibling checkout, and a defect in either package fails the other's tests.

**B. No; go-gitutil's tests call its own InitRepo, Add and Commit.**
- Pros: the package's tests depend on nothing outside it.
- Cons: a test of InitRepo relies on InitRepo to build its fixture, so a defect in it can hide.

Recommendation: B.

## Decided without asking

- `Repo.Add(ctx, AddArgs{Paths, All, Force})` is the single add operation; the fixture passes All and Force so ignore rules cannot drop fixture files.
- `Clone` is exported with `CloneArgs{Local, NoCheckout, Branch, Bare}` and cached_worktree.go calls it instead of its own clone.
- `CommitArgs` carries `AllowEmpty`, for the origin repository in gomion's worktree test.
- `InitArgs.Branch` becomes `git init -b <branch>`; empty means git's default. (`-b` needs git 2.28, from 2020.)
- Every operation takes a context and a directory and never assumes the process cwd.
- Real git via the CLI; no mocking (already in this task's context).
- go-fsfix calls these operations and shells out to git nowhere itself (E-2546).
- gomion's `gommod/gitutils` is deleted and gomion imports go-gitutil in E-2556; this task does not move it.
