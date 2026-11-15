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
