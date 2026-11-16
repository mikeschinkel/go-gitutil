package gitutil

import "strings"

// Conventional commit type names the package compares against.
const (
	conventionalTypeFeat = "feat"
	conventionalTypeFix  = "fix"
)

// conventionalCommitTypes lists the type names a UI offers as choices, with
// "none" first for a subject that has no type. ParseConventionalCommit does
// not require a type to be in this list.
var conventionalCommitTypes = []string{
	"none",
	conventionalTypeFeat,
	conventionalTypeFix,
	"docs",
	"style",
	"refactor",
	"perf",
	"test",
	"build",
	"ci",
	"chore",
	"revert",
}

// ConventionalCommitTypeNames returns the known conventional commit type
// names, "none" first, for a UI to offer as choices.
func ConventionalCommitTypeNames() []string {
	return append([]string{}, conventionalCommitTypes...)
}

// ConventionalCommit is a commit subject split by the Conventional Commits
// grammar, type(scope)!: description.
type ConventionalCommit struct {
	// Type is the type as written, such as "feat" or "Fix". Compare it
	// case-insensitively. Empty means the subject has no type.
	Type string
	// Scope is the text between the parentheses, or empty when there are none.
	Scope string
	// Breaking reports a ! before the colon.
	Breaking bool
	// Description is everything after the ": ".
	Description string
}

// ParseConventionalCommit splits subject of the form type(scope)!: description
// into its parts. The type is one word of letters, digits and hyphens; the
// scope and the ! are optional; a colon and a space end the prefix. Any type
// is accepted, not only the ones ConventionalCommitTypeNames lists. ok is
// false when subject has no type, and cc then holds subject as Description.
func ParseConventionalCommit(subject string) (cc ConventionalCommit, ok bool) {
	var rest string
	var i int

	cc.Description = subject
	for i < len(subject) && isConventionalTypeByte(subject[i]) {
		i++
	}
	if i == 0 {
		goto end
	}
	rest = subject[i:]
	if strings.HasPrefix(rest, "(") {
		closing := strings.IndexByte(rest, ')')
		if closing < 2 {
			// No closing parenthesis, or an empty scope.
			goto end
		}
		cc.Scope = rest[1:closing]
		rest = rest[closing+1:]
	}
	if strings.HasPrefix(rest, "!") {
		cc.Breaking = true
		rest = rest[1:]
	}
	if !strings.HasPrefix(rest, ": ") {
		cc = ConventionalCommit{Description: subject}
		goto end
	}
	cc.Type = subject[:i]
	cc.Description = rest[2:]
	ok = true
end:
	return cc, ok
}

// isConventionalTypeByte reports whether b may appear in a conventional
// commit type.
func isConventionalTypeByte(b byte) bool {
	return b >= 'a' && b <= 'z' ||
		b >= 'A' && b <= 'Z' ||
		b >= '0' && b <= '9' ||
		b == '-'
}

// String returns the subject cc describes: Description alone when Type is
// empty, otherwise type(scope)!: description with the scope and ! present
// only when set. For a subject ParseConventionalCommit accepted, it returns
// that subject unchanged.
func (cc ConventionalCommit) String() string {
	return cc.prefix() + cc.Description
}

// PrefixLen returns the length of everything before Description.
func (cc ConventionalCommit) PrefixLen() int {
	return len(cc.prefix())
}

// prefix returns everything String puts before Description.
func (cc ConventionalCommit) prefix() string {
	var sb strings.Builder

	if cc.Type == "" {
		goto end
	}
	sb.WriteString(cc.Type)
	if cc.Scope != "" {
		sb.WriteString("(" + cc.Scope + ")")
	}
	if cc.Breaking {
		sb.WriteString("!")
	}
	sb.WriteString(": ")
end:
	return sb.String()
}
