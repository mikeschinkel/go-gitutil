# Requirements from gomion (E-2546, 2026-11-03)

Settled with Mike in the gomion brainstorm. Design is this task's; these are what callers need.

## Direction of extraction (Mike, 2026-11-03)
gomion's gommod/gitutils moves into go-gitutil, not the other way round. gomion's package duplicates go-gitutil almost file for file and additionally has Commit, StageModuleFiles, UnstageAll, InfoDir/InfoStore; those become go-gitutil's. The fixture operations below are the first pieces to move.

## Operations go-fsfix needs (for RepoFixture, E-2547)
- Init a repository in a directory with a given default branch name.
- Add all files, forced (ignore rules must not drop fixture files).
- Commit with a given message and an explicit author/committer identity, so the result does not depend on the machine's git config.

## Operations gomion tests call directly
- Init with branch, empty commit (`--allow-empty`), clone to a directory, `worktree add -b <branch> <dir>`: gomion's gitutils worktree test builds origin, clone and linked worktree by hand today. gomion's worktree handling wants clone and worktree-add in non-test code as well.
- Stage and commit a set of files (gomion's commit flow, already in gommod/gitutils).

## Constraints
- Real git via the CLI; mocking git is ruled out (misses edge cases such as permissions).
- Operations take a directory and run there; they must not assume the process cwd.
