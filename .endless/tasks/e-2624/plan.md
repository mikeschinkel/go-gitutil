# Plan: commit signals, one conventional-commit grammar, an ignored-paths check

## Conventional commits
- New `ParseConventionalCommit(subject string) (ConventionalCommit, bool)` in conventional_commit.go: `type(scope)!: description`, type one word of letters, digits or hyphens (the spec allows any noun), scope optional, `!` optional. `ConventionalCommit{Type, Scope string; Breaking bool; Description string}` with `String()` and `PrefixLen()`.
- `ParseConventionalCommitType`, `ComposeConventionalCommitSubject` and `ConventionalCommitPrefixLen` are removed; their callers use the parser, `String` and `PrefixLen`. The type table keeps the names for `ConventionalCommitTypeNames`, which a UI offers as choices; the parser does not require a known type.
- New commit_signal.go: `CommitSignal`, `CommitClassifier`, `ConventionalCommitSignal` (breaking from `!` or a footer line, then feat, then fix, types compared case-insensitively), `CommitSignals` and `SummarizeCommitSignals`. Footer lines start with `BREAKING CHANGE: ` or `BREAKING-CHANGE: `, uppercase only. gomion's gommod/goutils/commit_signals.go is the reference for intent, not for the footer check.

## Ignored paths
- `(*Repo).Ignored(ctx, paths []dt.RelDirPath) ([]dt.RelDirPath, error)`: one `git check-ignore --stdin -z` with each path plus `/` on stdin, NUL-separated; parse the NUL-separated answer back to the given paths. Exit 1 means none ignored. Index-aware (git's default).

## Clean-up before the first release
- Remove `FindNestedModules` from add_commit.go; go-goutil owns it (E-2608).
- Correct `AddArgs.Force`'s comment.

## Tests
- In the existing test/ module with its current helpers; E-2602 later moves them to go-fsfix fixtures.

## Hand-off to gomion E-2601
- The commit-message editor (gomtui/commit_message_editor.go) moves from ParseConventionalCommitType/Compose/PrefixLen to ParseConventionalCommit and keeps the scope through an edit.
- `findingFromCommitSignals(gitutil.CommitSignals) goutil.FindingType` maps breaking to FindingBreaking, feat or fix to FindingLikelyCompatible, none with commits to FindingUnknown, no commits to FindingNoChanges; ClassifyCommitSubject callers use ConventionalCommitSignal(subject, "").
