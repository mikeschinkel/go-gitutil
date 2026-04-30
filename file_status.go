package gitutils

import (
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

// FileStatus represents the status of a file from git status --porcelain.
// The two-character status format is: XY filename
// where X is the index status (position 0) and Y is the working tree status (position 1).
type FileStatus struct {
	StagedChange   StatusChangeType // Status in index (position 0)
	UnstagedChange StatusChangeType // Status in working tree (position 1)
	Staging        Staging    // Derived staging state
}

func NewFileStatus() FileStatus {
	return FileStatus{}
}

type FilesMap map[dt.RelFilepath]File

func (fm FilesMap) Files() (files []File) {
	return dtx.MapValues(fm)
}

func (fm FilesMap) EnsureFile(fp dt.RelFilepath) {
	// Look up git status for this file
	_, found := fm[fp]
	if found {
		goto end
	}
	fm[fp] = NewFile(fp)
end:
	return
}

// ParseFilesMap parses git status --porcelain output into a map of file with filepath and status.
// Returns a map from relative filepath to FileStatus.
// Format of git status --porcelain:
//
//	XY filename
//	where X = index status (position 0), Y = worktree status (position 1)
//
// Examples:
//
//	"MM file.txt"     -> staged M, unstaged M
//	" M file.txt"     -> no staged change, unstaged M
//	"A  newfile.txt"  -> staged A, no unstaged change
//	"?? untracked.txt"-> untracked (both positions are '?')
func ParseFilesMap(output string) (statuses FilesMap, err error) {
	var line string
	var lines []string
	var relPath dt.RelFilepath

	statuses = make(FilesMap)

	// Split output into lines
	lines = strings.Split(output, "\n")

	for _, line = range lines {
		var status FileStatus
		// Git status --porcelain format: "XY filename"
		// where X and Y are status codes
		if len(line) < 4 {
			// Invalid line format - skip
			continue
		}

		relPath, status, err = ParseStatusLine(line)
		if err != nil {
			goto end
		}
		statuses[relPath] = File{
			Filepath: relPath,
			Status:   status,
		}
	}

end:
	return statuses, err
}

func ParseStatusFiles(output string) (files []dt.RelFilepath, err error) {
	var errs []error
	lines := strings.Split(output, "\n")
	files = make([]dt.RelFilepath, len(lines))

	for i, line := range lines {
		// Git status --porcelain format: "XY filename"
		// where X and Y are status codes
		if len(line) < 4 {
			// Invalid line format - skip
			continue
		}
		files[i], err = ParseStatusFile(line)
		errs = AppendErr(errs, err)
	}

	return files, CombineErrs(errs)
}

func ParseStatusFile(line string) (fp dt.RelFilepath, err error) {
	if len(line) < 4 {
		err = NewErr(dt.ErrTooLong, "line", line, "min_length", 4, "length", len(line))
		goto end
	}
	fp = dt.RelFilepath(strings.TrimSpace(line[3:]))
end:
	return fp, err
}

func ParseStatusLine(line string) (fp dt.RelFilepath, status FileStatus, err error) {
	var stagedChange, unstagedChange StatusChangeType
	var staging Staging

	fp, err = ParseStatusFile(line)
	if err != nil {
		goto end
	}

	// Parse position 0 (index/staged)
	stagedChange, err = ParseChangeType(line[0])
	if err != nil {
		goto end
	}

	// Parse position 1 (worktree/unstaged)
	unstagedChange, err = ParseChangeType(line[1])
	if err != nil {
		goto end
	}

	// Determine staging state
	staging, err = ParseStaging(stagedChange, unstagedChange)
	if err != nil {
		goto end
	}

	status = FileStatus{
		StagedChange:   stagedChange,
		UnstagedChange: unstagedChange,
		Staging:        staging,
	}

end:
	return fp, status, err
}
