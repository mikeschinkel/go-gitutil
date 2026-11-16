package test

import (
	"testing"

	"github.com/mikeschinkel/go-gitutil"
)

func TestParseConventionalCommit(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    gitutil.ConventionalCommit
		wantOK  bool
	}{
		{
			name:    "type only",
			subject: "fix: y",
			want:    gitutil.ConventionalCommit{Type: "fix", Description: "y"},
			wantOK:  true,
		},
		{
			name:    "scope",
			subject: "feat(api): add x",
			want:    gitutil.ConventionalCommit{Type: "feat", Scope: "api", Description: "add x"},
			wantOK:  true,
		},
		{
			name:    "bang",
			subject: "feat!: drop x",
			want:    gitutil.ConventionalCommit{Type: "feat", Breaking: true, Description: "drop x"},
			wantOK:  true,
		},
		{
			name:    "scope and bang",
			subject: "refactor(core)!: rework",
			want:    gitutil.ConventionalCommit{Type: "refactor", Scope: "core", Breaking: true, Description: "rework"},
			wantOK:  true,
		},
		{
			name:    "type keeps its case",
			subject: "Feat: x",
			want:    gitutil.ConventionalCommit{Type: "Feat", Description: "x"},
			wantOK:  true,
		},
		{
			name:    "type not in the table",
			subject: "wip-2: x",
			want:    gitutil.ConventionalCommit{Type: "wip-2", Description: "x"},
			wantOK:  true,
		},
		{
			name:    "empty description",
			subject: "feat: ",
			want:    gitutil.ConventionalCommit{Type: "feat"},
			wantOK:  true,
		},
		{
			name:    "merge subject",
			subject: "Merge branch 'x'",
			want:    gitutil.ConventionalCommit{Description: "Merge branch 'x'"},
		},
		{
			name:    "no space after colon",
			subject: "feat:x",
			want:    gitutil.ConventionalCommit{Description: "feat:x"},
		},
		{
			name:    "empty scope",
			subject: "feat(): x",
			want:    gitutil.ConventionalCommit{Description: "feat(): x"},
		},
		{
			name:    "unclosed scope",
			subject: "feat(api: x",
			want:    gitutil.ConventionalCommit{Description: "feat(api: x"},
		},
		{
			name:    "two words before colon",
			subject: "add feat: x",
			want:    gitutil.ConventionalCommit{Description: "add feat: x"},
		},
		{
			name:    "empty subject",
			subject: "",
			want:    gitutil.ConventionalCommit{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := gitutil.ParseConventionalCommit(tt.subject)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if got.String() != tt.subject {
				t.Errorf("String() = %q, want %q", got.String(), tt.subject)
			}
		})
	}
}

func TestConventionalCommitPrefixLen(t *testing.T) {
	tests := []struct {
		subject string
		want    int
	}{
		{"feat(api)!: x", len("feat(api)!: ")},
		{"fix: y", len("fix: ")},
		{"no type here", 0},
	}
	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			cc, _ := gitutil.ParseConventionalCommit(tt.subject)
			if got := cc.PrefixLen(); got != tt.want {
				t.Errorf("PrefixLen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConventionalCommitStringComposes(t *testing.T) {
	cc := gitutil.ConventionalCommit{Type: "feat", Scope: "api", Description: "x"}
	if got, want := cc.String(), "feat(api): x"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	cc = gitutil.ConventionalCommit{Description: "x"}
	if got, want := cc.String(), "x"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestConventionalCommitTypeNames(t *testing.T) {
	names := gitutil.ConventionalCommitTypeNames()
	if len(names) == 0 || names[0] != "none" {
		t.Fatalf("names = %v, want \"none\" first", names)
	}
	names[0] = "changed"
	if gitutil.ConventionalCommitTypeNames()[0] != "none" {
		t.Error("ConventionalCommitTypeNames returned the package's own slice")
	}
}

func TestConventionalCommitSignal(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		body    string
		want    gitutil.CommitSignal
	}{
		{"feat", "feat: x", "", gitutil.CommitSignalFeat},
		{"feat with scope", "feat(api): x", "", gitutil.CommitSignalFeat},
		{"feat bang", "feat!: x", "", gitutil.CommitSignalBreaking},
		{"refactor scope bang", "refactor(x)!: y", "", gitutil.CommitSignalBreaking},
		{"fix uppercase", "Fix: x", "", gitutil.CommitSignalFix},
		{"feat capitalized", "Feat: x", "", gitutil.CommitSignalFeat},
		{"fix", "fix: x", "", gitutil.CommitSignalFix},
		{"docs", "docs: x", "", gitutil.CommitSignalNone},
		{"no type", "update readme", "", gitutil.CommitSignalNone},
		{"footer", "chore: x", "Some detail.\n\nBREAKING CHANGE: removes Y", gitutil.CommitSignalBreaking},
		{"hyphen footer", "fix: x", "BREAKING-CHANGE: removes Y", gitutil.CommitSignalBreaking},
		{"footer on untyped subject", "update x", "BREAKING CHANGE: removes Y", gitutil.CommitSignalBreaking},
		{"phrase mid-line", "fix: x", "This is not a BREAKING CHANGE: really.", gitutil.CommitSignalFix},
		{"phrase lowercase", "feat: x", "breaking change: removes Y", gitutil.CommitSignalFeat},
		{"phrase without colon", "feat: x", "BREAKING CHANGE removes Y", gitutil.CommitSignalFeat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gitutil.ConventionalCommitSignal(tt.subject, tt.body); got != tt.want {
				t.Errorf("ConventionalCommitSignal(%q, %q) = %q, want %q", tt.subject, tt.body, got, tt.want)
			}
		})
	}
}

func TestSummarizeCommitSignals(t *testing.T) {
	entry := func(subject, body string) gitutil.CommitLogEntry {
		return gitutil.CommitLogEntry{Subject: subject, Body: body}
	}
	tests := []struct {
		name     string
		entries  []gitutil.CommitLogEntry
		classify gitutil.CommitClassifier
		want     gitutil.CommitSignals
	}{
		{
			name: "no entries",
			want: gitutil.CommitSignals{},
		},
		{
			name:    "fix and none",
			entries: []gitutil.CommitLogEntry{entry("fix: a", ""), entry("docs: b", "")},
			want:    gitutil.CommitSignals{Highest: gitutil.CommitSignalFix, Fix: 1, Total: 2},
		},
		{
			name: "mixed",
			entries: []gitutil.CommitLogEntry{
				entry("fix: a", ""),
				entry("feat: b", ""),
				entry("chore: c", "BREAKING CHANGE: d"),
				entry("feat(x): e", ""),
				entry("Merge branch 'y'", ""),
			},
			want: gitutil.CommitSignals{Highest: gitutil.CommitSignalBreaking, Breaking: 1, Feat: 2, Fix: 1, Total: 5},
		},
		{
			name: "only none",
			entries: []gitutil.CommitLogEntry{
				entry("update", ""),
			},
			want: gitutil.CommitSignals{Highest: gitutil.CommitSignalNone, Total: 1},
		},
		{
			name: "custom classifier",
			entries: []gitutil.CommitLogEntry{
				entry("[feature] a", ""),
				entry("[bugfix] b", ""),
				entry("feat: c", ""),
			},
			classify: func(subject, _ string) gitutil.CommitSignal {
				switch {
				case len(subject) > 9 && subject[:9] == "[feature]":
					return gitutil.CommitSignalFeat
				case len(subject) > 8 && subject[:8] == "[bugfix]":
					return gitutil.CommitSignalFix
				}
				return gitutil.CommitSignalNone
			},
			want: gitutil.CommitSignals{Highest: gitutil.CommitSignalFeat, Feat: 1, Fix: 1, Total: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitutil.SummarizeCommitSignals(tt.entries, tt.classify)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
