package gitutils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// DiffLineType represents the type of a line in a diff hunk
type DiffLineType int

const (
	DiffContext DiffLineType = iota // " " prefix - unchanged
	DiffAdded                       // "+" prefix - new line
	DiffRemoved                     // "-" prefix - deleted line
)

// DiffLine represents a single line in a hunk
type DiffLine struct {
	Type    DiffLineType
	Content string // Line content (without +/- prefix)
	OldLine int    // Line number in old file (0 if added)
	NewLine int    // Line number in new file (0 if removed)
}

// FileDiff represents all hunks for a single file
type FileDiff struct {
	OldPath  dt.RelFilepath // Path in the committed version (may differ for renames)
	NewPath  dt.RelFilepath // Path in actual version, 'actual' being content on disk.
	Hunks    []*DiffHunk
	OldLines []string
	NewLines []string
}

func NewFileDiff(path dt.RelFilepath) *FileDiff {
	return &FileDiff{
		OldPath: path,
		NewPath: path,
		Hunks:   make([]*DiffHunk, 0),
		// OldLines and NewLines intentionally nil so GetDiff()
		// can detect when they need to be populated from file content
	}
}

// ParseUnifiedDiff parses `git diff` output into structured FileDiff objects
func ParseUnifiedDiff(diffOutput string) (fileDiffs []*FileDiff, err error) {
	var lines []string
	var i int
	var line string
	var currentFile *FileDiff

	if diffOutput == "" {
		err = NewErr(ErrParseDiff, ErrEmptyDiff)
		goto end
	}

	lines = strings.Split(diffOutput, "\n")

	for i = 0; i < len(lines); i++ {
		line = lines[i]

		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != nil {
				fileDiffs = append(fileDiffs, currentFile)
			}
			currentFile = &FileDiff{}
			continue
		}

		if currentFile == nil {
			continue
		}

		path, found := currentFile.parseUnifiedDiffFile(line, "---")
		if found {
			currentFile.OldPath = path
			continue
		}
		path, found = currentFile.parseUnifiedDiffFile(line, "+++")
		if found {
			currentFile.NewPath = path
			continue
		}

		if strings.HasPrefix(line, "@@") {
			var hunk *DiffHunk
			hunk, err = parseHunk(lines, &i)
			if err != nil {
				goto end
			}
			currentFile.Hunks = append(currentFile.Hunks, hunk)
		}
	}

	if currentFile != nil {
		fileDiffs = append(fileDiffs, currentFile)
	}

end:
	if err != nil {
		err = WithErr(err, "diff_length", len(diffOutput))
	}
	return fileDiffs, err
}

func (fd *FileDiff) parseUnifiedDiffFile(line, marker string) (path dt.RelFilepath, found bool) {
	if strings.HasPrefix(line, marker+" ") {
		path = parsePath(line[4:])
		found = true
	}
	return path, found
}

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type HeaderRange struct {
	Start int
	Count int
}
type HunkHeader struct {
	Text     string
	OldRange HeaderRange
	NewRange HeaderRange
}

// ParseHunkHeader extracts values from @@ header
// Input: "@@ -10,5 +10,7 @@" or "@@ -10,5 +10,7 @@ func name"
// Returns: oldStart=10, oldCount=5, newStart=10, newCount=7
func ParseHunkHeader(header string) (hunkHeader HunkHeader, err error) {
	var matches []string
	var oldStart, oldCount, newStart, newCount int

	matches = hunkHeaderRegex.FindStringSubmatch(header)
	if matches == nil {
		err = NewErr(ErrParseDiff)
		goto end
	}

	oldStart, err = strconv.Atoi(matches[1])
	if err != nil {
		err = NewErr(ErrInvalidStartPos, "field", "old_start", err)
		goto end
	}

	oldCount = 1
	if matches[2] != "" {
		oldCount, err = strconv.Atoi(matches[2])
		if err != nil {
			err = NewErr(ErrInvalidLineCount, "field", "old_count", err)
			goto end
		}
	}

	newStart, err = strconv.Atoi(matches[3])
	if err != nil {
		err = NewErr(ErrInvalidStartPos, "field", "new_start", err)
		goto end
	}

	newCount = 1
	if matches[4] != "" {
		newCount, err = strconv.Atoi(matches[4])
		if err != nil {
			err = NewErr(ErrInvalidLineCount, "field", "new_count", err)
			goto end
		}
	}

	hunkHeader = HunkHeader{
		Text: header,
		OldRange: HeaderRange{
			Start: oldStart,
			Count: oldCount,
		},
		NewRange: HeaderRange{
			Start: newStart,
			Count: newCount,
		},
	}
end:
	if err != nil {
		err = WithErr(err, ErrInvalidHunkHeader, "header", header)
	}
	return hunkHeader, err
}

// ComputeHunkChecksum returns SHA256 of the hunk's changed content (+/- lines only)
func ComputeHunkChecksum(hunk *DiffHunk) string {
	var sb strings.Builder
	for _, line := range hunk.Lines {
		if line.Type == DiffAdded || line.Type == DiffRemoved {
			sb.WriteString(line.Content)
			sb.WriteString("\n")
		}
	}
	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// ComputeFileChecksum returns SHA256 of file content
func ComputeFileChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// parseHunk parses a single hunk starting at the @@ line
// i is passed as a pointer so we can advance it past the hunk
func parseHunk(lines []string, i *int) (hunk *DiffHunk, err error) {
	var header string
	var hh HunkHeader
	var oldStart, newStart, oldCount, newCount int
	var oldProcessed, newProcessed int
	var line string
	var diffLine DiffLine

	header = lines[*i]
	hh, err = ParseHunkHeader(header)
	if err != nil {
		goto end
	}

	hunk = &DiffHunk{
		Header: hh,
		Lines:  make([]DiffLine, 0),
	}

	oldStart = hh.OldRange.Start
	newStart = hh.NewRange.Start
	oldCount = hh.OldRange.Count
	newCount = hh.NewRange.Count
	*i++

	for *i < len(lines) {
		line = lines[*i]

		if strings.HasPrefix(line, "@@") {
			break
		}

		if strings.HasPrefix(line, "diff --git ") {
			break
		}

		if oldProcessed >= oldCount && newProcessed >= newCount {
			break
		}
		switch {
		case len(line) == 0:
			diffLine = DiffLine{
				Type:    DiffContext,
				Content: "",
				OldLine: oldStart,
				NewLine: newStart,
			}
			oldStart++
			newStart++
			oldProcessed++
			newProcessed++
		default:
			switch line[0] {
			case ' ':
				diffLine = DiffLine{
					Type:    DiffContext,
					Content: line[1:],
					OldLine: oldStart,
					NewLine: newStart,
				}
				oldStart++
				newStart++
				oldProcessed++
				newProcessed++
			case '+':
				diffLine = DiffLine{
					Type:    DiffAdded,
					Content: line[1:],
					OldLine: 0,
					NewLine: newStart,
				}
				newStart++
				newProcessed++
			case '-':
				diffLine = DiffLine{
					Type:    DiffRemoved,
					Content: line[1:],
					OldLine: oldStart,
					NewLine: 0,
				}
				oldStart++
				oldProcessed++
			case '\\':
				*i++
				continue
			default:
				err = NewErr(ErrParseDiff, "unexpected_line_prefix", string(line[0]))
				goto end
			}
		}

		hunk.Lines = append(hunk.Lines, diffLine)
		*i++
	}

	*i--

end:
	if err != nil {
		err = WithErr(err, "header", header)
	}
	return hunk, err
}

// parsePath extracts the file path from a --- or +++ line
// Handles both "a/path" and "/dev/null" formats
func parsePath(pathLine string) (relPath dt.RelFilepath) {
	path := strings.TrimSpace(pathLine)
	if path == "/dev/null" {
		goto end
	}
	if !strings.HasPrefix(path, "a/") && !strings.HasPrefix(path, "b/") {
		goto end
	}
	relPath = dt.RelFilepath(path[2:])
end:
	return relPath
}

// String returns a string representation of DiffLineType
func (t DiffLineType) String() string {
	switch t {
	case DiffContext:
		return "context"
	case DiffAdded:
		return "added"
	case DiffRemoved:
		return "removed"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}
