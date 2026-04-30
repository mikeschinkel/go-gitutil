package gitutils

import "errors"

var (
	ErrNotGitRepo            = errors.New("not a git repository")
	ErrNoSemverTags          = errors.New("no semver tags found")
	ErrNoReachableSemverTags = errors.New("no reachable semver tags found")
	ErrInvalidGitStatusCode  = errors.New("invalid git status code")
)

var (
	ErrGitInfoPathNotExist         = errors.New(".git/info path does not exist")
	ErrFailedInspectingGitInfoPath = errors.New("failed inspecting .git/info path")
	ErrFailedToGetFileSystem       = errors.New("failed to get file system")
	ErrFailedToGetGitInfoFilepath  = errors.New("failed to get .git/info filepath")
	ErrFailedToLoadGitInfoFile     = errors.New("failed to load .git/info file")
	ErrFailedToSaveGitInfoFile     = errors.New("failed to save .git/info file")
	ErrFailedToLoadJSONFile        = errors.New("failed to load JSON file")
	ErrFailedToSaveJSONFile        = errors.New("failed to save JSON file")
	ErrAppendToGitignore           = errors.New("failed to append to .gitignore")
	ErrAppendToGitExclude          = errors.New("failed to append to .git/info/exclude")
	ErrUnstageFile                 = errors.New("failed to unstage file")
	ErrStageFileContent            = errors.New("failed to stage file content")
)

var (
	ErrDiff              = errors.New("diff error")
	ErrParseDiff         = errors.New("parse diff error")
	ErrInvalidStartPos   = errors.New("invalid start position")
	ErrInvalidLineCount  = errors.New("invalid line count")
	ErrInvalidHunkHeader = errors.New("invalid hunk header error")
	ErrEmptyDiff         = errors.New("empty diff error")
)

var (
	ErrPush            = errors.New("failed to push")
	ErrCreateTag       = errors.New("failed to create tag")
	ErrPushTag         = errors.New("failed to push tag")
	ErrDeleteTag       = errors.New("failed to delete tag")
	ErrDeleteRemoteTag = errors.New("failed to delete remote tag")
	ErrCommitLog       = errors.New("failed to get commit log")
	ErrFilesChanged    = errors.New("failed to get files changed")
	ErrBranches        = errors.New("failed to list branches")
	ErrCommitDiff      = errors.New("failed to get commit diff")
	ErrCommitFileDiff  = errors.New("failed to get commit file diff")
	ErrDiffStatRange   = errors.New("failed to get diff stat range")
)
