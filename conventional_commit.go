package gitutil

import "strings"

// conventionalCommitType maps a type name to its prefix string.
type conventionalCommitType struct {
	Name   string // "none", "feat", "fix", etc.
	Prefix string // "", "feat: ", "fix: ", etc.
}

// conventionalCommitTypes defines the known conventional commit types.
var conventionalCommitTypes = []conventionalCommitType{
	{Name: "none", Prefix: ""},
	{Name: "feat", Prefix: "feat: "},
	{Name: "fix", Prefix: "fix: "},
	{Name: "docs", Prefix: "docs: "},
	{Name: "style", Prefix: "style: "},
	{Name: "refactor", Prefix: "refactor: "},
	{Name: "perf", Prefix: "perf: "},
	{Name: "test", Prefix: "test: "},
	{Name: "build", Prefix: "build: "},
	{Name: "ci", Prefix: "ci: "},
	{Name: "chore", Prefix: "chore: "},
	{Name: "revert", Prefix: "revert: "},
}

// ConventionalCommitTypeNames returns the list of known conventional commit type names.
func ConventionalCommitTypeNames() []string {
	names := make([]string, len(conventionalCommitTypes))
	for i, ct := range conventionalCommitTypes {
		names[i] = ct.Name
	}
	return names
}

// ConventionalCommitPrefixLen returns the prefix length for the given type name.
// Returns 0 for unknown types, "none", or empty string.
func ConventionalCommitPrefixLen(typeName string) int {
	for _, ct := range conventionalCommitTypes {
		if ct.Name == typeName {
			return len(ct.Prefix)
		}
	}
	return 0
}

// ParseConventionalCommitType extracts the conventional commit type from a subject line.
// Returns (typeName, remainder). If no known type prefix is found, returns ("", subject).
func ParseConventionalCommitType(subject string) (typeName string, remainder string) {
	remainder = subject
	// Skip "none" (index 0) since it has no prefix to match
	for _, ct := range conventionalCommitTypes[1:] {
		if !strings.HasPrefix(subject, ct.Prefix) {
			continue
		}
		typeName = ct.Name
		remainder = subject[len(ct.Prefix):]
		goto end
	}
end:
	return typeName, remainder
}

// ComposeConventionalCommitSubject combines type and subject text.
// If typeName is "" or "none", returns subject as-is.
func ComposeConventionalCommitSubject(typeName string, subject string) (result string) {
	result = subject
	if typeName == "" || typeName == "none" {
		goto end
	}
	for _, ct := range conventionalCommitTypes {
		if ct.Name != typeName {
			continue
		}
		result = ct.Prefix + subject
		goto end
	}
end:
	return result
}
