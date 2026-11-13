package gitutil

// DiffHunk represents a single @@ hunk
type DiffHunk struct {
	Header HunkHeader // Raw @@ line, e.g. "@@ -10,5 +10,7 @@"
	Lines  []DiffLine // The actual diff lines
}
