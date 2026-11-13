package gitutil

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"golang.org/x/mod/semver"
)

// FilesMap returns the porcelain status of the working tree parsed into a
// FilesMap.
// It does not need Open.
func (r *Repo) FilesMap(ctx context.Context, args *StatusArgs) (m FilesMap, err error) {
	var output string
	if args == nil {
		args = &StatusArgs{}
	}
	args.HumanReadable = false
	output, err = r.Status(ctx, args)
	if err != nil {
		goto end
	}

	// Parse git status
	m, err = ParseFilesMap(output)
	if err != nil {
		goto end
	}

end:
	return m, err
}

// IsDirty reports whether the working tree has uncommitted changes.
// It does not need Open.
func (r *Repo) IsDirty(ctx context.Context) (bool, error) {
	out, err := r.Status(ctx, nil)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// IsDirtyInPath checks if there are uncommitted changes within a specific path
// It does not need Open.
func (r *Repo) IsDirtyInPath(ctx context.Context, relPath dt.PathSegments) (bool, error) {
	out, err := r.Status(ctx, &StatusArgs{Path: relPath})
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// UpstreamState returns how far the current branch is ahead of and behind
// its upstream. It reads the upstream from git (@{u}).
// It does not need Open.
func (r *Repo) UpstreamState(ctx context.Context) (us UpstreamState, err error) {
	var out string
	var fields []string

	_, err = r.runGit(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		goto end
	}
	out, err = r.runGit(ctx, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		goto end
	}
	fields = strings.Fields(strings.TrimSpace(out))
	if len(fields) != 2 {
		err = fmt.Errorf("unexpected rev-list output: %q", out)
		goto end
	}
	us = UpstreamState{
		ahead:  r.toIntPtr(fields[0]),
		behind: r.toIntPtr(fields[1]),
	}
end:
	return us, err
}

// CurrentRemoteURL returns the push URL of the current remote
// Tries get-url first, falls back to parsing remote -v output.
// It needs Open.
func (r *Repo) CurrentRemoteURL(ctx context.Context) (url dt.URL, err error) {
	var remote dt.GitRemoteName
	var out string
	var lines []string
	var fields []string
	var u string

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}

	// Try get-url first (simpler, but may not work on all git versions)
	u, err = r.runGit(ctx, "remote", "get-url", string(remote))
	if err == nil && strings.TrimSpace(u) != "" {
		url, err = dt.ParseURL(strings.TrimSpace(u))
		goto end
	}

	// Fall back to parsing remote -v output
	out, err = r.runGit(ctx, "remote", "-v")
	if err != nil {
		goto end
	}

	// Parse output: "remotename\turl (push)"
	// Find first line with "(push)"
	lines = strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "(push)") {
			// Split on whitespace
			fields = strings.Fields(line)
			if len(fields) >= 2 {
				url, err = dt.ParseURL(fields[1])
				goto end
			}
		}
	}

	err = fmt.Errorf("no push remote found")

end:
	return url, err
}

// StatusCounts holds counts of staged, unstaged, and untracked files.
type StatusCounts struct {
	Staged    int
	Unstaged  int
	Untracked int
}

// StatusCounts returns counts of staged, unstaged, and untracked files,
// scoped and filtered by args.
// It does not need Open.
func (r *Repo) StatusCounts(ctx context.Context, args *StatusArgs) (counts StatusCounts, err error) {
	var out string
	var lines []string
	var pathPrefix string

	if args == nil {
		args = &StatusArgs{}
	}

	out, err = r.Status(ctx, args)
	if err != nil {
		goto end
	}

	// Build path prefix for filtering (with trailing slash if not root)
	// Note: "." means current directory (same as empty string for our purposes)
	if !args.Path.IsRoot() {
		pathPrefix = string(args.Path) + "/"
	}

	// TrimRight (not TrimSpace): leading characters are meaningful git status codes.
	// E.g., " M file" starts with space meaning "not staged" — TrimSpace would strip it.
	lines = strings.Split(strings.TrimRight(out, "\n\r"), "\n")
	for _, line := range lines {
		var filePath string
		var shouldExclude bool

		if line == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}

		// Apply exclude paths if any were specified
		if len(args.ExcludePaths) > 0 {
			// Extract file path (format: "XY path" where XY is 2-char status)
			filePath = strings.TrimSpace(line[3:])

			// Remove the path prefix to get relative path within module
			if pathPrefix != "" {
				if !strings.HasPrefix(filePath, pathPrefix) {
					continue
				}
				filePath = strings.TrimPrefix(filePath, pathPrefix)
			}

			// Check if this file is in an excluded subpath
			for _, excludePath := range args.ExcludePaths {
				excludePrefix := string(excludePath) + "/"
				if strings.HasPrefix(filePath, excludePrefix) || filePath == string(excludePath) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}
		}

		x := line[0] // Index (staged) status
		y := line[1] // Worktree (unstaged) status

		// Untracked files
		if x == '?' && y == '?' {
			counts.Untracked++
			continue
		}

		// Staged changes
		if x != ' ' && x != '?' {
			counts.Staged++
		}

		// Unstaged changes
		if y != ' ' && y != '?' {
			counts.Unstaged++
		}
	}

end:
	return counts, err
}

// FetchTags fetches all tags from the current branch's remote. It needs Open.
func (r *Repo) FetchTags(ctx context.Context) (err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	_, err = r.streamGit(ctx, "fetch", "--tags", string(remote))
end:
	return err
}

func (r *Repo) toIntPtr(s string) (n *int) {
	n = new(int)
	*n, _ = strconv.Atoi(s)
	return n
}

// Tags returns local tags under prefix: tags with no "/" when prefix is
// the root, or tags starting with prefix + "/" otherwise.
// It does not need Open.
func (r *Repo) Tags(ctx context.Context, prefix dt.PathSegments) (tags []dt.GitTagName, err error) {
	out, err := r.runGit(ctx, "tag", "--list")
	if err != nil {
		goto end
	}

	// Normalize prefix: "." means root (same as empty)
	if prefix.IsRoot() {
		prefix = ""
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		tag := strings.TrimSpace(line)
		switch {
		case tag == "":
			continue
		case prefix.IsRoot() && strings.Contains(tag, "/"):
			continue
		case !prefix.IsRoot() && !strings.HasPrefix(tag, string(prefix)+"/"):
			continue
		}
		tags = append(tags, dt.GitTagName(tag))
	}
end:
	return tags, err
}

// RemoteTags returns all tags from the current branch's remote. It needs Open.
func (r *Repo) RemoteTags(ctx context.Context, prefix dt.PathSegments) (tags []dt.GitTagName, err error) {
	var out string
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}

	out, err = r.runGit(ctx, "ls-remote", "--tags", string(remote))
	if err != nil {
		goto end
	}

	// Normalize prefix: "." means root (same as empty)
	if prefix.IsRoot() {
		prefix = ""
	}

	// Parse ls-remote output: "hash refs/tags/tagname" or "hash refs/tags/tagname^{}"
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ref := fields[1]
		// Skip dereferenced tags (those ending with ^{})
		if strings.HasSuffix(ref, "^{}") {
			continue
		}

		// Extract tag name from refs/tags/tagname
		if !strings.HasPrefix(ref, "refs/tags/") {
			continue
		}
		tag := strings.TrimPrefix(ref, "refs/tags/")

		// Apply prefix filter
		switch {
		case prefix.IsRoot() && strings.Contains(tag, "/"):
			continue
		case !prefix.IsRoot() && !strings.HasPrefix(tag, string(prefix)+"/"):
			continue
		}

		tags = append(tags, dt.GitTagName(tag))
	}

end:
	return tags, err
}

// CompareRemoteTags checks if remote has newer tags than local
// Returns: missingTags (tags on remote but not local), error. It needs Open.
func (r *Repo) CompareRemoteTags(ctx context.Context, prefix dt.PathSegments) (missingTags []dt.GitTagName, err error) {
	var localTags []dt.GitTagName
	var remoteTags []dt.GitTagName
	var localTagSet map[dt.GitTagName]bool

	localTags, err = r.Tags(ctx, prefix)
	if err != nil {
		goto end
	}

	remoteTags, err = r.RemoteTags(ctx, prefix)
	if err != nil {
		goto end
	}

	// Build set of local tags for fast lookup
	localTagSet = make(map[dt.GitTagName]bool, len(localTags))
	for _, tag := range localTags {
		localTagSet[tag] = true
	}

	// Find tags that exist on remote but not locally
	for _, tag := range remoteTags {
		if !localTagSet[tag] {
			missingTags = append(missingTags, tag)
		}
	}

	// Sort missing tags (newest first if they're semver)
	if len(missingTags) > 0 {
		sort.Slice(missingTags, func(i, j int) bool {
			return semver.Compare(string(missingTags[i]), string(missingTags[j])) > 0
		})
	}

end:
	return missingTags, err
}

type LatestTagArgs struct {
	IncludeUnreachable bool // default: false
	AllowNonSemver     bool // default: false
	ModuleRelPath      dt.RelDirPath
}

// LatestTag returns the latest reachable semver tag by default.
// Passing non-default options is currently unsupported.
// It does not need Open.
func (r *Repo) LatestTag(ctx context.Context, headCommit dt.GitRevision, args *LatestTagArgs) (latest dt.GitTagName, err error) {
	var semverTags []dt.GitTagName
	var reachable []dt.GitTagName
	var tags []dt.GitTagName

	switch {
	case args == nil:
		args = &LatestTagArgs{}
	case args.IncludeUnreachable:
		panic("IncludeUnreachable option for Repo.LatestTag() is not yet implemented")
	case args.AllowNonSemver:
		panic("AllowNonSemver option for Repo.LatestTag() is not yet implemented")
	}

	tags, err = r.Tags(ctx, args.ModuleRelPath)

	if err != nil {
		goto end
	}

	for _, tag := range tags {
		if !semver.IsValid(string(tag)) {
			continue
		}
		semverTags = append(semverTags, tag)
	}

	if len(semverTags) == 0 {
		err = ErrNoSemverTags
		goto end
	}

	for _, tag := range semverTags {
		var ok bool
		ok, err = r.isAncestor(ctx, dt.GitRevision(tag), headCommit)
		if err != nil {
			goto end
		}
		if ok {
			reachable = append(reachable, tag)
		}
	}
	if len(reachable) == 0 {
		err = ErrNoReachableSemverTags
		goto end
	}

	sort.Slice(reachable, func(i, j int) bool {
		return semver.Compare(string(reachable[i]), string(reachable[j])) > 0
	})
	latest = reachable[0]
end:
	return latest, err
}

// isAncestor reports whether older is an ancestor of newer.
func (r *Repo) isAncestor(ctx context.Context, older, newer dt.GitRevision) (isAncestor bool, err error) {
	_, err = r.runGit(ctx, "merge-base", "--is-ancestor", string(older), string(newer))
	switch {
	case err == nil:
		isAncestor = true
	case exitCode(err) == 1:
		err = nil
	}
	return isAncestor, err
}

// DiffNameStatus returns git diff --name-status lines for fromRef..toRef.
// It does not need Open.
func (r *Repo) DiffNameStatus(ctx context.Context, fromRef, toRef dt.GitRevision, patterns ...string) (newLines []string, err error) {
	var lines []string
	var out string
	var n int
	args := []string{"diff", "--name-status", fmt.Sprintf("%s..%s", fromRef, toRef), "--"}
	args = append(args, patterns...)
	out, err = r.runGit(ctx, args...)
	if err != nil {
		goto end
	}
	lines = strings.Split(strings.TrimSpace(out), "\n")
	newLines = make([]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		newLines[n] = line
		n++
	}
	lines = newLines[:n]
end:
	return lines, err
}

// DiffStatEntry represents a single file's change statistics from git diff --numstat
type DiffStatEntry struct {
	Path       string
	Insertions int
	Deletions  int
}

// DiffStatRange returns parsed file stats for the range fromRef..toRef, optionally
// filtered by paths. Results are sorted by total changes (insertions+deletions) descending.
// Binary files (shown as "-\t-\t<path>" by git) get Insertions=0, Deletions=0.
// It does not need Open.
func (r *Repo) DiffStatRange(ctx context.Context, fromRef, toRef dt.GitRevision, paths []string) (entries []DiffStatEntry, err error) {
	var out string
	var lines []string

	args := []string{"diff", "--numstat", fmt.Sprintf("%s..%s", fromRef, toRef)}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	out, err = r.runGit(ctx, args...)
	if err != nil {
		err = NewErr(ErrDiffStatRange, err)
		goto end
	}

	lines = strings.Split(strings.TrimSpace(out), "\n")
	entries = make([]DiffStatEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		entry := DiffStatEntry{Path: fields[2]}
		// Binary files show "-" for both insertions and deletions
		if fields[0] != "-" {
			entry.Insertions, _ = strconv.Atoi(fields[0])
		}
		if fields[1] != "-" {
			entry.Deletions, _ = strconv.Atoi(fields[1])
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		totalI := entries[i].Insertions + entries[i].Deletions
		totalJ := entries[j].Insertions + entries[j].Deletions
		return totalI > totalJ
	})

end:
	return entries, err
}

// DiffNumStat returns git diff --numstat lines for fromRef..toRef.
// It does not need Open.
func (r *Repo) DiffNumStat(ctx context.Context, fromRef, toRef dt.GitRevision, patterns ...string) ([]string, error) {
	args := []string{"diff", "--numstat", fmt.Sprintf("%s..%s", fromRef, toRef), "--"}
	args = append(args, patterns...)
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// OpenCachedWorktree returns a locked clone of the repository in the user's
// cache directory, refreshed from it. Close releases the lock. It needs Open,
// because the cache clones the repository's top level.
func (r *Repo) OpenCachedWorktree(ctx context.Context) (wt *CachedWorktree, err error) {
	err = r.requireOpen()
	if err != nil {
		goto end
	}
	wt, err = openCachedWorktree(ctx, r)
end:
	return wt, err
}

// GetChangedFiles returns all changed files in the working directory
// This includes both staged and unstaged changes
// It does not need Open.
func (r *Repo) GetChangedFiles(ctx context.Context) (files []dt.RelFilepath, err error) {
	var out string
	var lines []string

	// Use git status --porcelain to get all changed files
	// Format: XY filename
	// X = staged status, Y = unstaged status
	out, err = r.Status(ctx, nil)
	if err != nil {
		goto end
	}

	lines = strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Split on first space only to separate status from filename
		// Handles filenames with spaces correctly
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		fp := dt.RelFilepath(parts[1])
		if len(fp) == 0 {
			continue
		}
		if fp[0] == ' ' {
			fp = fp[1:]
		}
		files = append(files, fp)
	}

end:
	return files, err
}

// GetChangedFilesFiltered returns changed files matching the filter function
// This includes both staged and unstaged changes, filtered by the provided function
// It does not need Open.
func (r *Repo) GetChangedFilesFiltered(ctx context.Context, filter FileFilter, _ *StatusArgs) (filtered []dt.RelFilepath, err error) {
	var allFiles []dt.RelFilepath

	// Get all changed files
	allFiles, err = r.GetChangedFiles(ctx)
	if err != nil {
		goto end
	}

	// Apply filter if provided
	if filter != nil {
		filtered = make([]dt.RelFilepath, 0, len(allFiles))
		for _, file := range allFiles {
			if filter(file) {
				filtered = append(filtered, file)
			}
		}
	} else {
		filtered = allFiles
	}

end:
	return filtered, err
}

// GetWorkingDiff returns the full diff of all changes in the working directory
// This includes both staged and unstaged changes
// It does not need Open.
func (r *Repo) GetWorkingDiff(ctx context.Context) (diff string, err error) {
	var stagedDiff string
	var unstagedDiff string

	// Get staged diff (what's in the index)
	stagedDiff, err = r.GetStagedDiff(ctx)
	if err != nil {
		goto end
	}

	// Get unstaged diff (working directory changes not in index)
	unstagedDiff, err = r.runGit(ctx, "diff")
	if err != nil {
		goto end
	}

	// Combine both diffs
	if stagedDiff != "" && unstagedDiff != "" {
		diff = stagedDiff + "\n" + unstagedDiff
	} else if stagedDiff != "" {
		diff = stagedDiff
	} else {
		diff = unstagedDiff
	}

end:
	return diff, err
}

// CreateTagArgs provides arguments for CreateTag
type CreateTagArgs struct {
	TagName dt.GitTagName  // Required: tag name (e.g. "v1.2.3" or "gommod/v1.0.0")
	Message string         // Required: annotated tag message
	Commit  dt.GitRevision // Optional: commit to tag (defaults to HEAD)
}

// CreateTag creates an annotated git tag
// It does not need Open.
func (r *Repo) CreateTag(ctx context.Context, args CreateTagArgs) (err error) {
	var commit dt.GitRevision

	commit = args.Commit
	if commit == "" {
		commit = "HEAD"
	}

	_, err = r.runGit(ctx, "tag", "-a", string(args.TagName), "-m", args.Message, string(commit))

	return err
}

// remoteName returns the remote the current branch tracks, or DefaultRemote
// when it tracks none. It returns ErrRepoNotOpen on an unopened Repo.
func (r *Repo) remoteName() (remote dt.GitRemoteName, err error) {
	err = r.requireOpen()
	if err != nil {
		goto end
	}
	remote = r.Remote.Name
	if remote == "" {
		remote = DefaultRemote
	}
end:
	return remote, err
}

// Push pushes the current branch to its remote. It needs Open.
func (r *Repo) Push(ctx context.Context) (output string, err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	output, err = r.streamGit(ctx, "push", string(remote), string(r.Branch))
end:
	return output, err
}

// PushAll pushes the current branch and its tags to its remote. It needs
// Open.
func (r *Repo) PushAll(ctx context.Context) (output string, err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	output, err = r.streamGit(ctx, "push", string(remote), string(r.Branch), "--follow-tags")
end:
	return output, err
}

// PushTags pushes all tags to the current branch's remote. It needs Open.
func (r *Repo) PushTags(ctx context.Context) (output string, err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	output, err = r.streamGit(ctx, "push", "--tags", string(remote))
end:
	return output, err
}

// PushTag pushes a tag to the current branch's remote. It needs Open.
func (r *Repo) PushTag(ctx context.Context, tag dt.GitTagName) (err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	_, err = r.streamGit(ctx, "push", string(remote), string(tag.ToRef()))
end:
	return err
}

// DeleteTag deletes a local tag. It does not need Open.
func (r *Repo) DeleteTag(ctx context.Context, tag dt.GitTagName) (err error) {
	_, err = r.runGit(ctx, "tag", "-d", string(tag))
	return err
}

// DeleteRemoteTag deletes a tag on the current branch's remote. It needs
// Open.
func (r *Repo) DeleteRemoteTag(ctx context.Context, tag dt.GitTagName) (err error) {
	var remote dt.GitRemoteName

	remote, err = r.remoteName()
	if err != nil {
		goto end
	}
	_, err = r.streamGit(ctx, "push", string(remote), ":"+string(tag.ToRef()))
end:
	return err
}

// commitRecordSep is the record separator for git log output with multi-line bodies.
// Must not contain null bytes — OS rejects them in exec arguments.
const commitRecordSep = "---COMMIT_RECORD_SEP---"

// CommitLogEntry represents a single entry from git log
type CommitLogEntry struct {
	Hash       dt.GitObjectID // Full commit ID
	Subject    string         // First line of commit message
	Body       string         // Remaining commit message body (may be empty)
	AuthorDate string         // ISO 8601 date
}

// CommitsSinceTag returns commits between a tag and HEAD, optionally filtered by paths.
// Returns commits in reverse chronological order (newest first). It does
// not need Open.
func (r *Repo) CommitsSinceTag(ctx context.Context, fromTag dt.GitTagName, paths []string) ([]CommitLogEntry, error) {
	return r.commitLog(ctx, string(fromTag)+"..HEAD", paths)
}

// CommitsBetweenTags returns commits between two tags (exclusive of fromTag, inclusive of toTag).
// Returns commits in reverse chronological order (newest first). It does
// not need Open.
func (r *Repo) CommitsBetweenTags(ctx context.Context, fromTag, toTag dt.GitTagName, paths []string) ([]CommitLogEntry, error) {
	return r.commitLog(ctx, string(fromTag)+".."+string(toTag), paths)
}

// CommitsOnBranch returns commits unique to a branch (not on the current branch).
// Returns commits in reverse chronological order (newest first). It reads
// the current branch from git, so it does not need Open.
func (r *Repo) CommitsOnBranch(ctx context.Context, branch dt.GitBranchName, paths []string) (entries []CommitLogEntry, err error) {
	var current dt.GitBranchName

	current, err = r.currentBranch(ctx, nil)
	if err != nil {
		err = NewErr(ErrCommitLog, err)
		goto end
	}
	if current == "" {
		// Detached HEAD: compare against the commit HEAD is at.
		current = "HEAD"
	}
	entries, err = r.commitLog(ctx, string(current)+".."+string(branch), paths)
end:
	return entries, err
}

// commitLog returns the commits in revRange, optionally filtered by paths,
// newest first.
func (r *Repo) commitLog(ctx context.Context, revRange string, paths []string) (entries []CommitLogEntry, err error) {
	var out string
	var records []string
	var args []string

	args = []string{
		"log",
		revRange,
		"--format=%H\t%s\t%aI\t%b" + commitRecordSep,
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	out, err = r.runGit(ctx, args...)
	if err != nil {
		err = NewErr(ErrCommitLog, err)
		goto end
	}

	out = strings.TrimSpace(out)
	if out == "" {
		goto end
	}

	records = strings.Split(out, commitRecordSep)
	entries = make([]CommitLogEntry, 0, len(records))
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		entry := CommitLogEntry{
			Hash:       dt.GitObjectID(parts[0]),
			Subject:    parts[1],
			AuthorDate: parts[2],
		}
		if len(parts) >= 4 {
			entry.Body = strings.TrimSpace(parts[3])
		}
		entries = append(entries, entry)
	}

end:
	return entries, err
}

// FilesChangedArgs provides arguments for FilesChanged.
type FilesChangedArgs struct {
	CommitHash dt.GitRevision // Required: the commit to inspect
	Paths      []string       // Optional: filter to these paths
}

// FilesChanged returns the list of files changed in a specific commit. It
// does not need Open.
func (r *Repo) FilesChanged(ctx context.Context, args FilesChangedArgs) (files []string, err error) {
	var out string

	gitArgs := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", string(args.CommitHash)}
	if len(args.Paths) > 0 {
		gitArgs = append(gitArgs, "--")
		gitArgs = append(gitArgs, args.Paths...)
	}

	out, err = r.runGit(ctx, gitArgs...)
	if err != nil {
		err = NewErr(ErrFilesChanged, err)
		goto end
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}

end:
	return files, err
}

// CommitDiff returns the full unified diff for a commit.
// Runs: git show --format="" <rev>. It does not need Open.
func (r *Repo) CommitDiff(ctx context.Context, rev dt.GitRevision) (diff string, err error) {
	diff, err = r.runGit(ctx, "show", "--format=", string(rev))
	if err != nil {
		err = NewErr(ErrCommitDiff, err)
		goto end
	}
	diff = strings.TrimSpace(diff)
end:
	return diff, err
}

// CommitFileDiff returns the diff for a single file in a commit.
// Runs: git diff <rev>^..<rev> -- <path>. It does not need Open.
func (r *Repo) CommitFileDiff(ctx context.Context, rev dt.GitRevision, path string) (diff string, err error) {
	diff, err = r.runGit(ctx, "diff", string(rev)+"^.."+string(rev), "--", path)
	if err != nil {
		err = NewErr(ErrCommitFileDiff, err)
		goto end
	}
	diff = strings.TrimSpace(diff)
end:
	return diff, err
}

// Branches returns all local branch names. It does not need Open.
func (r *Repo) Branches(ctx context.Context) (branches []dt.GitBranchName, err error) {
	var out string

	out, err = r.runGit(ctx, "branch", "--list", "--format=%(refname:short)")
	if err != nil {
		err = NewErr(ErrBranches, err)
		goto end
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, dt.GitBranchName(line))
	}

end:
	return branches, err
}
