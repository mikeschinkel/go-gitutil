package gitutils

import (
	"context"
	"fmt"
	"strings"

	"github.com/mikeschinkel/go-diffutils"
	"github.com/mikeschinkel/go-dt"
)

// GetFileDiff returns unified diff for a single file vs HEAD
// Shows all uncommitted changes (staged + unstaged) for the specified file
func (r *Repo) GetDiffRaw(ctx context.Context, path dt.RelFilepath) (diff string, err error) {
	diff, err = r.runGit(ctx, r.Root, "diff", "HEAD", "--", string(path))
	if err != nil {
		err = WithErr(err, "path", path)
		goto end
	}

end:
	return diff, err
}

// GetFilesDiffs returns unified diff for multiple files vs HEAD
// Shows all uncommitted changes (staged + unstaged) for the specified files
// This is used when the user has already selected which files to include in a commit
func (r *Repo) GetFilesDiffs(ctx context.Context, paths []dt.RelFilepath) (diff string, err error) {
	var args []string

	if len(paths) == 0 {
		goto end
	}

	args = []string{"diff", "HEAD", "--"}
	for _, p := range paths {
		args = append(args, string(p))
	}

	diff, err = r.runGit(ctx, r.Root, args...)
	if err != nil {
		err = WithErr(err, "file_count", len(paths))
		goto end
	}

end:
	return diff, err
}

// GetFileAtHEAD retrieves the committed version of a file from HEAD This is used
// for the left side of a diff view (the "before" state) Returns nil content with
// no error if the file doesn't exist in HEAD (e.g., new files)
//
// TODO: This should be reanamed LoadFile() and accept args for things like
//
//	"HEAD" but I need to better understand the options first.
func (r *Repo) GetFileAtHEAD(ctx context.Context, relPath dt.RelFilepath) (content []byte, err error) {
	var cmd string
	var out string

	cmd = "HEAD:" + string(relPath)
	out, err = r.runGit(ctx, r.Root, "show", cmd)
	if err != nil {
		// File may not exist in HEAD (new file) - return empty content
		// Check if error indicates file doesn't exist
		err = WithErr(err, "path", relPath)
		goto end
	}

	content = []byte(out)

end:
	return content, err
}

// GetDiff returns the data needed to build split pane diff view
// Returns FileDiff, committed file lines, and current file lines
func (r *Repo) GetDiff(ctx context.Context, relPath dt.RelFilepath) (fileDiff *FileDiff, err error) {
	var diffOutput string
	var fileDiffs []*FileDiff
	var commitContent []byte
	var actualContent []byte
	var commitLines []string
	var actualLines []string

	// Get both file versions
	commitContent, err = r.GetFileAtHEAD(ctx, relPath)
	if err != nil {
		commitContent = nil
		err = nil // Ignore error for new files
	}

	actualContent, err = dt.FilepathJoin(r.Root, relPath).ReadFile()
	if err != nil {
		actualContent = nil
		err = nil // File may not exist on disk (deleted file)
	}

	commitLines = []string{}
	if len(commitContent) > 0 {
		commitLines = strings.Split(string(commitContent), "\n")
	}
	actualLines = []string{}
	if len(actualContent) > 0 {
		actualLines = strings.Split(string(actualContent), "\n")
	}

	// Get the diff
	diffOutput, err = r.GetDiffRaw(ctx, relPath)
	if err != nil {
		goto end
	}

	if diffOutput == "" {
		goto end
	}

	// Parse the diff
	fileDiffs, err = ParseUnifiedDiff(diffOutput)
	if err != nil {
		goto end
	}
	if len(fileDiffs) == 0 {
		goto end
	}

	fileDiff = fileDiffs[0]
	fileDiff.OldLines = commitLines // Old = from HEAD (before)
	fileDiff.NewLines = actualLines // New = from disk (after)

end:
	if fileDiff == nil && err == nil {
		fileDiff = NewFileDiff(relPath)
	}
	// Ensure lines are set even when there are no hunks (e.g., new/untracked files)
	if fileDiff != nil && fileDiff.OldLines == nil {
		fileDiff.OldLines = commitLines
		fileDiff.NewLines = actualLines
	}
	return fileDiff, err
}

// safeGetLine returns a line from a slice or empty string if out of bounds
func safeGetLine(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	return lines[index]
}

// FileChanges represents file differences optimized for split pane rendering
type FileChanges struct {
	Path         dt.RelFilepath // File path
	CommitLines  []string       // Full old file content (from HEAD)
	ActualLines  []string       // Full new file content (from disk)
	ChangeBlocks []ChangeBlock  // Blocks of additions or deletions
}

// ChangeBlock represents a contiguous block of added or deleted lines
type ChangeBlock struct {
	Type        BlockChangeType
	CommitRange LineRange // Range in commit version
	ActualRange LineRange // Range in actual version
}

// BlockChangeType indicates whether lines were added or deleted in a diff
type BlockChangeType int

const (
	LinesAdded   BlockChangeType = iota // Lines exist in actual but not commit
	LinesDeleted                        // Lines exist in commit but not actual
)

// LineRange specifies a range of lines in a file
type LineRange struct {
	Start int // 1-based line number where range starts
	Count int // Number of lines in the range
}

// ToDiffContent converts FileChanges to diffutils.DiffContent for use with
// teadiff.SplitDiffModel and other go-diffutils consumers.
func (fc *FileChanges) ToDiffContent() *diffutils.DiffContent {
	if fc == nil {
		return nil
	}
	changes := make([]diffutils.DiffChange, len(fc.ChangeBlocks))
	for i, block := range fc.ChangeBlocks {
		changes[i] = diffutils.DiffChange{
			Type:     diffutils.ChangeType(block.Type),
			OldRange: diffutils.LineRange{Start: block.CommitRange.Start, Count: block.CommitRange.Count},
			NewRange: diffutils.LineRange{Start: block.ActualRange.Start, Count: block.ActualRange.Count},
		}
	}
	return &diffutils.DiffContent{
		OldLines: fc.CommitLines,
		NewLines: fc.ActualLines,
		Changes:  changes,
		Label:    string(fc.Path),
	}
}

func (ct BlockChangeType) String() string {
	switch ct {
	case LinesAdded:
		return "added"
	case LinesDeleted:
		return "deleted"
	default:
		return fmt.Sprintf("unknown(%d)", ct)
	}
}

// GetFileChanges returns file changes optimized for TUI rendering
func (r *Repo) GetFileChanges(ctx context.Context, relPath dt.RelFilepath) (changes *FileChanges, err error) {
	var fileDiff *FileDiff

	// Use existing GetDiff which provides:
	// - Parsed FileDiff with hunks
	// - CommitLines and ActualLines
	fileDiff, err = r.GetDiff(ctx, relPath)
	if err != nil {
		goto end
	}

	// Extract change blocks from hunks
	changes = extractChangeBlocks(fileDiff)

	// For new/untracked files with no hunks, synthesize all-added block
	if len(changes.ChangeBlocks) == 0 && len(changes.ActualLines) > 0 {
		changes.ChangeBlocks = append(changes.ChangeBlocks, ChangeBlock{
			Type:        LinesAdded,
			CommitRange: LineRange{Start: 1, Count: 0},
			ActualRange: LineRange{Start: 1, Count: len(changes.ActualLines)},
		})
	}

end:
	return changes, err
}

// extractChangeBlocks converts FileDiff hunks into simplified ChangeBlocks
func extractChangeBlocks(fd *FileDiff) *FileChanges {
	fc := &FileChanges{
		Path:         fd.NewPath, // Use NewPath (actual path on disk)
		CommitLines:  fd.OldLines,
		ActualLines:  fd.NewLines,
		ChangeBlocks: make([]ChangeBlock, 0),
	}

	for _, hunk := range fd.Hunks {
		commitLineNo := hunk.Header.OldRange.Start
		actualLineNo := hunk.Header.NewRange.Start

		var addedBlock []string
		var addedStart int
		var deletedBlock []string
		var deletedStart int

		for _, line := range hunk.Lines {
			switch line.Type {
			case DiffContext:
				// Context line - flush any pending blocks
				// IMPORTANT: Flush deletions BEFORE additions so blocks are in processing order
				if len(deletedBlock) > 0 {
					fc.ChangeBlocks = append(fc.ChangeBlocks, ChangeBlock{
						Type:        LinesDeleted,
						CommitRange: LineRange{Start: deletedStart, Count: len(deletedBlock)},
						ActualRange: LineRange{Start: actualLineNo, Count: 0},
					})
					deletedBlock = nil
				}
				if len(addedBlock) > 0 {
					fc.ChangeBlocks = append(fc.ChangeBlocks, ChangeBlock{
						Type:        LinesAdded,
						CommitRange: LineRange{Start: commitLineNo, Count: 0},
						ActualRange: LineRange{Start: addedStart, Count: len(addedBlock)},
					})
					addedBlock = nil
				}
				commitLineNo++
				actualLineNo++

			case DiffAdded:
				if len(addedBlock) == 0 {
					addedStart = actualLineNo
				}
				addedBlock = append(addedBlock, line.Content)
				actualLineNo++

			case DiffRemoved:
				if len(deletedBlock) == 0 {
					deletedStart = commitLineNo
				}
				deletedBlock = append(deletedBlock, line.Content)
				commitLineNo++
			}
		}

		// Flush any remaining blocks at end of hunk
		// IMPORTANT: Flush deletions BEFORE additions so blocks are in processing order
		if len(deletedBlock) > 0 {
			fc.ChangeBlocks = append(fc.ChangeBlocks, ChangeBlock{
				Type:        LinesDeleted,
				CommitRange: LineRange{Start: deletedStart, Count: len(deletedBlock)},
				ActualRange: LineRange{Start: actualLineNo, Count: 0},
			})
		}
		if len(addedBlock) > 0 {
			fc.ChangeBlocks = append(fc.ChangeBlocks, ChangeBlock{
				Type:        LinesAdded,
				CommitRange: LineRange{Start: commitLineNo, Count: 0},
				ActualRange: LineRange{Start: addedStart, Count: len(addedBlock)},
			})
		}
	}

	return fc
}
