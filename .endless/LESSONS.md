# Lessons Learned

A capture log of corrections, newest last. Entries are appended by
`endless lesson write`, which commits this file on the main checkout as it
writes — so recording a lesson never waits on a task landing, and never
dirties a worktree.

## Format

```
### [YYYY-MM-DD] One-line summary
<the lesson — what went wrong, why, and the rule that replaces it>
- **Project**: <project>
```

---

## Entries

### [2026-11-12] A brainstorm outcome is the current best understanding, not dogma; say so in it and expect it to evolve during implementation
In E-2548 Mike asked for a note that the outcome is what we best understand and may need to evolve during implementation, because he finds that once an outcome is written it gets treated as settled and he has to fight to change a design when implementation reveals a need. Rule: write that caveat into every brainstorm or research outcome, and when implementation finds a reason to deviate, treat the outcome as a starting point to revise, not an authority to defend.
- **Project**: go-gitutil

### [2026-11-12] A choice that shapes one task's implementation is a task decision for its plan or analysis, even when the user made it; file a decision in the project it binds
In E-2548 I filed four decisions; Mike rejected three as task-level (the Repo API shape with optional StdIO, where fixture constants live) or as a platitude (Go-module knowledge not in go-gitutil, 'water is wet'), and moved the fourth (singular util names across go-pkgs) to gomion because a cross-repo convention belongs where the ecosystem is managed, not in the repo where it came up. Rule: before filing, ask whether the statement would still bind if the task were deleted and whether anyone would argue the other side; a design the user chose for one package's API fails the first test and goes in the implementing task's analysis.
- **Project**: go-gitutil

### [2026-11-12] Extracting a package into a new repo is three tasks: create the project first, extract in the new project, replace in the consumer
In E-2548 I filed one gomion task 'Extract gommod/goutils into go-goutil'. Mike split it: create the go-goutil project first, a go-goutil task to extract the package, a gomion task to replace the local copy with the import. Rule: an extraction crosses two projects, so the work lives in each project's own task, and the destination project must exist before its task can be filed.
- **Project**: go-gitutil

### [2026-11-12] A command you hand the user to run must have been checked against its preconditions; project init needs a repo with at least one commit
In E-2548 I gave Mike a one-liner to git init an empty directory and run endless project init. He had to add a README and commit first before project init would run, and told me the instruction was insufficient. Rule: before handing the user a command, check what it requires (an existing commit, a file, a registered project) by reading its help or trying the equivalent where allowed, and include those steps in the command.
- **Project**: go-gitutil
