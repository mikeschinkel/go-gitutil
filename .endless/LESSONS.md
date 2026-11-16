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

### [2026-11-13] A worktree with no session is a staged task, not a failed spawn
Investigating E-2551 (underway, worktree present, no session ever bound), I told Mike a spawn had pre-claimed the task and then failed to launch. Mike corrected me: the worktree was created on purpose to hold his main-checkout edit and wait for a later spawn. The ledger pattern (status flipped to underway by a system actor, worktree created the same second, no session) is what a deliberately staged task looks like, not evidence of a failure. When the record fits a deliberate path, say what the record shows and ask, rather than naming a failure.
- **Project**: go-gitutil

### [2026-11-13] Never claim a task to get a worktree; a claim without a session is not a tool
Investigating E-2551 (2026-11-13), Mike found that a planning session had run 'endless task claim --unattended' on four tasks it filed, solely to obtain worktrees for moving his uncommitted main-checkout edits onto branches. Mike: a claim's entire purpose is to associate a session with a task; pre-claiming is like filling a car's tank before shipping it cross-country. The result was tasks underway with no session, which goto cannot reach and spawn silently adopts; he wants that state never to occur. Apply: do not run task claim from a session that will not work the task, and do not pass --unattended; if a worktree is needed for the user's own edits, say that endless has no verb for it yet (brainstorm E-2545) and ask Mike rather than bending claim to do it.
- **Project**: go-gitutil

### [2026-11-13] Adopting edits made outside a task: branch, reset main, delegate to tasks
Discussing E-2545 on 2026-11-13, Mike set the route for edits he makes in a main checkout outside any task: commit them to a holding branch (wip/E-NNNN, never the task branch, which worktree creation would refuse as an orphan carrying real work), reset main, and delegate the rest to project-specific tasks whose spawned sessions bring the edits into their own worktrees. He corrected my 'no worktree' wording: a spawned session always has a worktree; what is never created is a worktree without a session. Apply: never create a worktree up front for such edits; describe the route in those terms.
- **Project**: go-gitutil

### [2026-11-14] Show code when presenting API-shape options
While planning E-2608 (go-goutil extraction) I presented API options for AnalyzeCommitMessages' input type as prose with symbol names. Mike: the options were not clear, he needs more detail and wants to see code, not just symbol names, and he suspected an interface option was missing. When an open question is about the shape of an API, show each option as a short code snippet (signature plus the call site it implies) and include an interface-based option when one is plausible.
- **Project**: go-gitutil

### [2026-11-14] Error-path code must not raise errors; don't silence a deliberate panic to get past it
Context: E-2600 in go-gitutil. go-dt's DirPath.ErrKV() calls ToTilde(), which calls GetUserHomeDir(), which panics unless the program called dt.EnsureUserHomeDir(). I proposed changing GetUserHomeDir() to look up the home directory silently on first use. Mike rejected that: the panic exists on purpose so the user sees that EnsureUserHomeDir() was never called, and quietly making it go away defeats that intent. The root cause is elsewhere: error-handling code should take special effort not to generate errors itself, so ErrKV() should not depend on ToTilde(). It should output the full path, and downstream code (whatever displays the error) should turn $HOME into '~'. Lesson: when a deliberate error or panic stands in the way, don't remove or bypass it. Ask whether the code that triggers it belongs on that path at all, especially on an error path.
- **Project**: go-gitutil

### [2026-11-14] go-nextver is irrelevant; leave it out of research and analysis
Context: planning go-dt E-2623 (ErrKV() returns full paths without ToTilde()). While surveying consumers across Mike's projects, I cited go-nextver's ux() as evidence of an existing pattern and of a broken ErrValue lookup. Mike said go-nextver is an irrelevant project and should be hidden. Lesson: when surveying consumers of Mike's packages, leave go-nextver out of findings and analysis.
- **Project**: go-gitutil

### [2026-11-14] Work in a project always goes in a task in that project; never ask
Context: planning go-dt E-2623, which needs a matching change in go-cliutil's error printing. I asked whether to file a separate go-cliutil task or widen E-2623 to cover both repos. Mike said this is a question I should never ask: work in a project always belongs in a task in that project. Lesson: when a plan needs a change in another project, file (or plan for) a task in that project and link it; do not offer to widen the current task across projects.
- **Project**: go-gitutil

### [2026-11-14] Several needs from one cause are one task
While planning E-2608 (go-goutil extraction) I filed two go-gitutil tasks, E-2624 (commit signal classification) and E-2625 (Repo.Ignored check-ignore batch). Mike pointed me at ED-1550: several symptoms of one cause are one task; file the cause. Both existed for one reason, the extraction leaving gomion two git-shaped needs, so they belonged in one task. Before filing a second task in the same repo for the same originating decision, ask whether it shares the cause, and if so put both in one task.
- **Project**: go-gitutil

### [2026-11-15] go-stash and gru are irrelevant; leave them out of research and analysis
Context: planning go-cliutil E-2626 (shorten home-directory paths in printed errors). While surveying apps that use go-cliutil, I cited go-stash and gru as evidence of how apps call writer.Errorf before cliutil.Initialize(). Mike said both are irrelevant projects and should be hidden, as he said earlier of go-nextver. Lesson: when surveying consumers of Mike's packages, leave go-stash and gru out of findings and analysis.
- **Project**: go-gitutil

### [2026-11-15] Search the full task list before filing
While planning go-gitutil E-2529 I filed gomion E-2627 (repo setup reads command words as dirs) after grepping `endless task list --all`, which shows only 20 rows by default. E-2540 already covered it, and Mike said several sessions have re-reported this bug. Before filing, search with --no-limit (or read every row) in the target project so a duplicate isn't missed.
- **Project**: go-gitutil
