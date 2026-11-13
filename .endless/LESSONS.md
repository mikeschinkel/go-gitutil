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

### [2026-11-12] In an Endless worktree, go.mod ../ replaces don't resolve; drop them or the verify tidy row fails
Worktrees sit at <repo>/.endless/worktrees/e-NNNN, so a go.mod replace like ../go-dt points at .endless/worktrees/go-dt, which does not exist. endless task verify always runs a built-in 'go mod tidy -diff' under GOWORK=off, so any module with ../ replaces fails verify from a worktree. Also, an untracked go.work in the worktree makes verify refuse as uncommitted; go.work must be gitignored. In E-2611 (go-gitutil) the fix was requiring published versions with no replace (the go-pkgs rule anyway) and gitignoring go.work/go.work.sum.
- **Project**: go-gitutil

### [2026-11-13] Mike's Args convention: one Args, last parameter
Planning go-gitutil E-2600, I recommended adding RepoArgs as an extra parameter to the package-level Init/Clone/Open (Init(ctx, dir, RepoArgs, InitArgs)). Mike rejected it: across his packages a function takes at most one Args struct and it is the last parameter. When a function needs settings from two Args structs, embed one in the other (InitArgs embeds RepoArgs) instead of adding a second Args parameter.
- **Project**: go-gitutil

### [2026-11-13] Check whether a default removes a con before listing it
Planning go-gitutil E-2600 Q1, I listed 'a caller wanting one identity sets it twice' as a con of separate Author/Committer fields. Mike pointed out that a missing Committer can default to Author (and CommitterDate to AuthorDate), so the con disappears. Before listing a con, check whether a sensible default or zero-value rule removes it. Prefer zero values (dt.Mailbox with empty Email, time.Time.IsZero) to pointers for detecting an unset field.
- **Project**: go-gitutil

### [2026-11-13] Use nil pointers, not zero-value rules, to mark an optional Args field as unset
While planning E-2600 (go-gitutil CommitArgs), I proposed that a zero Committer/CommitterDate means 'copy Author/AuthorDate', relying on an empty dt.Mailbox.Email and time.Time.IsZero(). Mike rejected that: zero-value rules are fragile, the meaning of 'unset' differs per type and has to be learned field by field, so developers and AI agents misread it. A nil pointer means 'not given' to everyone. So, when an Args field's absence triggers a default, make it a pointer and default on nil.
- **Project**: go-gitutil

### [2026-11-13] Follow-up tasks pose the question, not my chat conclusion
Filing a go-dt follow-up after concluding in chat that no remote-tracking branch type was needed, I titled it 'Document why no remote-tracking branch type exists'. Mike reframed it as 'Consider adding a remote-tracking branch type'. A conclusion reached in a chat is reasoning for the task's analysis, not a decision the task exists to record; the task should pose the open question so whoever plans it weighs both sides, with the chat reasoning as a starting point.
- **Project**: go-gitutil

### [2026-11-13] For go-gitutil, isolated means no machine state at all, not just no git config files
While planning E-2600's strict mode, I offered isolating only git's global and system config files (GIT_CONFIG_GLOBAL=/dev/null, GIT_CONFIG_NOSYSTEM=1), then listed SSH keys, askpass env vars and the user's .netrc file as auth methods that would 'still work'. Mike pointed out that each of those is the machine leaking into a test that is meant to be isolated. Isolation is the goal; network auth is not. So build the process environment from scratch (env -i style): HOME set to an empty dir, GIT_ALLOW_PROTOCOL=file, no inherited GIT_* or SSH_* vars. Don't propose adding a secrets store to make isolated mode reach network remotes; that defeats the goal.
- **Project**: go-gitutil

### [2026-11-13] The nil-pointer rule is for struct-typed Args fields, not scalars
Scopes the earlier lesson 'Use nil pointers, not zero-value rules, to mark an optional Args field as unset'. While planning E-2600 I applied it to MinVersion/MaxVersion and made them pointers. Mike said I over-indexed: he meant struct-typed fields (like a dt.Mailbox), where which property decides that the struct counts as 'not passed' is unclear. A scalar whose zero value is obvious and well known (an empty string, 0) keeps using the zero value to mean unset.
- **Project**: go-gitutil
