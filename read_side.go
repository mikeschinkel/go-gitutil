package gitutil

import (
	"context"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// GitlinkMode is the tree entry mode of a gitlink: a nested repository or
// submodule recorded as a commit.
const GitlinkMode = "160000"

// CommitCount returns the number of commits reachable from rev. It does not
// need Open.
func (r *Repo) CommitCount(ctx context.Context, rev dt.GitRevision) (count int, err error) {
	var out string

	out, err = r.runGit(ctx, "rev-list", "--count", "--end-of-options", string(rev))
	if err != nil {
		goto end
	}
	count, err = strconv.Atoi(strings.TrimSpace(out))
end:
	if err != nil {
		err = NewErr(ErrCommitCount, "revision", rev, err)
	}
	return count, err
}

// TreeEntry is one entry of a git tree, as git ls-tree lists it.
type TreeEntry struct {
	// Mode is the entry's octal mode, such as 100644, 040000, or
	// GitlinkMode (160000).
	Mode string
	// Type is the object type: blob, tree or commit.
	Type string
	// ObjectID names the entry's object.
	ObjectID dt.GitObjectID
	// Path is the entry's path from the repository's top level.
	Path dt.RelFilepath
}

// TreeEntry returns the entry for path in rev's tree, or an error wrapping
// ErrTreeEntryNotFound when the tree has none. path is relative to the
// repository's top level. It does not need Open.
func (r *Repo) TreeEntry(ctx context.Context, rev dt.GitRevision, path dt.RelFilepath) (entry TreeEntry, err error) {
	var out string
	var meta, name string
	var fields []string
	var found bool

	out, err = r.run(ctx, nil, runArgs{
		NULSeparated: true,
		Command: []string{"ls-tree", "-z", "--full-tree", "--end-of-options",
			string(rev), "--", string(path),
		},
	})
	if err != nil {
		goto end
	}
	// Output is "<mode> SP <type> SP <object> TAB <path> NUL".
	out = strings.TrimSuffix(out, "\x00")
	if out == "" {
		err = NewErr(ErrTreeEntryNotFound)
		goto end
	}
	meta, name, found = strings.Cut(out, "\t")
	fields = strings.Fields(meta)
	if !found || len(fields) != 3 {
		err = NewErr(ErrTreeEntry, "output", out)
		goto end
	}
	entry = TreeEntry{
		Mode:     fields[0],
		Type:     fields[1],
		ObjectID: dt.GitObjectID(fields[2]),
		Path:     dt.RelFilepath(name),
	}
end:
	if err != nil {
		err = WithErr(err, ErrTreeEntry, "revision", rev, "path", path)
	}
	return entry, err
}
