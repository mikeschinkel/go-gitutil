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
