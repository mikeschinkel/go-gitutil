# Commit signals

Shape agreed in the E-2608 planning chat:

```go
type CommitSignal string
const (
	CommitSignalNone     CommitSignal = ""
	CommitSignalFix      CommitSignal = "fix"
	CommitSignalFeat     CommitSignal = "feat"
	CommitSignalBreaking CommitSignal = "breaking"
)
// CommitClassifier maps one commit message to its signal.
type CommitClassifier func(subject, body string) CommitSignal
// ConventionalCommitSignal is the default classifier.
func ConventionalCommitSignal(subject, body string) CommitSignal
type CommitSignals struct {
	Highest  CommitSignal
	Breaking int
	Feat     int
	Fix      int
	Total    int
}
// nil classify means ConventionalCommitSignal.
func SummarizeCommitSignals(entries []CommitLogEntry, classify CommitClassifier) CommitSignals
```

ConventionalCommitSignal should reuse conventional_commit.go's type table rather than hard-code "feat" and "fix" prefixes a second time. The hasBreakingBangPrefix, hasBreakingChangeFooter and hasConventionalPrefix helpers in gomion's commit_signals.go are the reference behaviour; gomion's E-2601 then writes `findingFromCommitSignals(gitutil.CommitSignals) goutil.FindingType` (breaking → FindingBreaking, feat or fix → FindingLikelyCompatible, none with commits → FindingUnknown, no commits → FindingNoChanges). Tests go in go-gitutil's test/ module (E-2602 creates it).

# Ignored-paths check

Shape agreed in the E-2608 planning chat:

```go
// Ignored runs one git check-ignore over paths and returns the subset git ignores.
func (r *Repo) Ignored(ctx context.Context, paths []dt.RelDirPath) ([]dt.RelDirPath, error)

// gomion composes it with goutil:
nested, err := goutil.FindNestedModules(dir, nil)
ignored, err := repo.Ignored(ctx, nested)
nested = slices.DeleteFunc(nested, func(d dt.RelDirPath) bool { return slices.Contains(ignored, d) })
```

Use `check-ignore --stdin -z` with paths NUL-separated on stdin through the one run function (E-2600 gives it stdin), and read the NUL-separated answer. check-ignore exits 1 when no path is ignored, which is not an error here. Whether the parameter is []dt.RelDirPath or a path union covering files too is for the implementer to settle with Mike; the known caller passes directories. Tests in test/ (E-2602).

# Decided while planning (2026-11-15, with Mike, from go-goutil E-2608's session)

## One conventional-commit grammar, existing functions rebuilt on it (Mike: B)
- conventional_commit.go matched exact prefixes ("feat: "), so `feat(api): x` and `feat!: x` parsed as having no type, while gomion's commit-signal helpers understood scope and `!`. Options: (A) add a spec parser and leave the three existing functions, leaving two exported readings that disagree on scoped subjects; (B) add it and rebuild the existing functions on it, so scope and `!` survive parse and compose; (C) port gomion's private helpers, a second private reading. Mike chose B: one grammar, broken before the first release rather than after. Cost accepted: gomion's commit-message editor must carry the scope through an edit, handed to E-2601.

## Decided without asking (Mike could object)
- Breaking-change footer per the spec: a line starting with `BREAKING CHANGE: ` or `BREAKING-CHANGE: `, uppercase only. gomion's `strings.Contains` also fired on prose mentioning the phrase.
- Types match case-insensitively, per the spec.
- The parser accepts any one-word type (the spec allows any noun); the type table only supplies names a UI offers.
- `Ignored` takes `[]dt.RelDirPath` (the known caller passes directories), appends `/` so dir-only patterns match, treats exit 1 as none ignored, and uses git's index-aware default.
- Tests go in the existing test/ module; E-2602 moves them to go-fsfix later.
- `CommitClassifier func(subject, body string)`: a caller holding only a subject passes "".

## Correction to the context's embedded-repo advice item
- The context says `Add` with `Force` should pass `-c advice.addEmbeddedRepo=false`. `Repo.Add` runs through `runGit`, which never streams; git's output is captured and only attached to an error on failure, so the advice reaches no one and E-2568's P11 holds without a change. Dropped; the misleading `AddArgs.Force` comment (it says Force makes a nested repository a gitlink; plain add does that too) is corrected instead.
