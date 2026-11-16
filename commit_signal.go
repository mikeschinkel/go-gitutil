package gitutil

import "strings"

// CommitSignal is what one commit message says about the change it makes,
// ordered breaking > feat > fix > none.
type CommitSignal string

const (
	// CommitSignalNone means the message signals nothing.
	CommitSignalNone CommitSignal = ""
	// CommitSignalFix means the message signals a fix.
	CommitSignalFix CommitSignal = "fix"
	// CommitSignalFeat means the message signals a feature.
	CommitSignalFeat CommitSignal = "feat"
	// CommitSignalBreaking means the message signals a breaking change.
	CommitSignalBreaking CommitSignal = "breaking"
)

// rank orders signals so the highest wins.
func (s CommitSignal) rank() (rank int) {
	switch s {
	case CommitSignalBreaking:
		rank = 3
	case CommitSignalFeat:
		rank = 2
	case CommitSignalFix:
		rank = 1
	}
	return rank
}

// CommitClassifier maps one commit message to its signal. A caller holding
// only a subject passes an empty body.
type CommitClassifier func(subject, body string) CommitSignal

// breakingChangeFooters are the tokens that start a breaking-change footer
// line. The Conventional Commits spec requires them in uppercase.
var breakingChangeFooters = []string{
	"BREAKING CHANGE: ",
	"BREAKING-CHANGE: ",
}

// ConventionalCommitSignal is the default CommitClassifier, following the
// Conventional Commits spec. It returns CommitSignalBreaking when the subject
// has a ! before the colon or a body line starts with "BREAKING CHANGE: " or
// "BREAKING-CHANGE: ", otherwise CommitSignalFeat for a feat type,
// CommitSignalFix for a fix type, and CommitSignalNone for anything else.
// Types match case-insensitively.
func ConventionalCommitSignal(subject, body string) (signal CommitSignal) {
	cc, ok := ParseConventionalCommit(subject)
	switch {
	case ok && cc.Breaking, hasBreakingChangeFooter(body):
		signal = CommitSignalBreaking
	case !ok:
		signal = CommitSignalNone
	case strings.EqualFold(cc.Type, conventionalTypeFeat):
		signal = CommitSignalFeat
	case strings.EqualFold(cc.Type, conventionalTypeFix):
		signal = CommitSignalFix
	}
	return signal
}

// hasBreakingChangeFooter reports whether a line of body starts with a
// breaking-change footer token.
func hasBreakingChangeFooter(body string) (has bool) {
	for line := range strings.Lines(body) {
		for _, token := range breakingChangeFooters {
			if strings.HasPrefix(line, token) {
				has = true
				goto end
			}
		}
	}
end:
	return has
}

// CommitSignals summarizes the signals of a set of commits.
type CommitSignals struct {
	// Highest is the highest signal of any commit.
	Highest CommitSignal
	// Breaking counts commits that signal a breaking change.
	Breaking int
	// Feat counts commits that signal a feature.
	Feat int
	// Fix counts commits that signal a fix.
	Fix int
	// Total counts every commit, whatever it signals.
	Total int
}

// SummarizeCommitSignals classifies each entry's subject and body with
// classify and counts the results. A nil classify means
// ConventionalCommitSignal.
func SummarizeCommitSignals(entries []CommitLogEntry, classify CommitClassifier) (signals CommitSignals) {
	if classify == nil {
		classify = ConventionalCommitSignal
	}
	signals.Total = len(entries)
	for _, entry := range entries {
		signal := classify(entry.Subject, entry.Body)
		switch signal {
		case CommitSignalBreaking:
			signals.Breaking++
		case CommitSignalFeat:
			signals.Feat++
		case CommitSignalFix:
			signals.Fix++
		}
		if signal.rank() > signals.Highest.rank() {
			signals.Highest = signal
		}
	}
	return signals
}
