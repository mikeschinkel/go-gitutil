package gitutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mikeschinkel/go-dt"
)

// AddArgs holds the arguments of Add.
type AddArgs struct {
	// Paths limits the add to these paths. Empty with All set adds the
	// whole working tree.
	Paths []dt.RelFilepath
	// Exclude leaves these directories out, each passed as a
	// :(exclude)<dir>/ pathspec.
	Exclude []dt.RelDirPath
	// All passes --all, so deletions are staged too.
	All bool
	// Force passes --force, so files that git ignores are added too.
	Force bool
}

// Add stages files with git add. It does not need Open. args may be nil.
func (r *Repo) Add(ctx context.Context, args *AddArgs) (err error) {
	var cmd []string

	if args == nil {
		args = &AddArgs{}
	}
	cmd = []string{"add"}
	if args.All {
		cmd = append(cmd, "--all")
	}
	if args.Force {
		cmd = append(cmd, "--force")
	}
	cmd = append(cmd, "--")
	for _, p := range args.Paths {
		cmd = append(cmd, string(p))
	}
	for _, dir := range args.Exclude {
		cmd = append(cmd, ":(exclude)"+strings.TrimSuffix(string(dir), "/")+"/")
	}
	_, err = r.runGit(ctx, cmd...)
	if err != nil {
		err = NewErr(ErrAdd, err)
	}
	return err
}

// CommitArgs holds the arguments of Commit. The identity and date fields
// are pointers: nil means "not given".
type CommitArgs struct {
	// Message is the commit message.
	Message string
	// Author sets GIT_AUTHOR_NAME and GIT_AUTHOR_EMAIL. Nil lets git
	// resolve the author.
	Author *dt.Mailbox
	// Committer sets GIT_COMMITTER_NAME and GIT_COMMITTER_EMAIL. Nil uses
	// Author.
	Committer *dt.Mailbox
	// AuthorDate sets GIT_AUTHOR_DATE. Nil lets git use the current time.
	AuthorDate *time.Time
	// CommitterDate sets GIT_COMMITTER_DATE. Nil uses AuthorDate.
	CommitterDate *time.Time
	// AllowEmpty passes --allow-empty.
	AllowEmpty bool
}

// Commit commits the staged changes. Fixed Author and AuthorDate give the
// same commit ID for the same tree and parents on every run. It does not
// need Open.
func (r *Repo) Commit(ctx context.Context, args *CommitArgs) (err error) {
	var cmd []string
	var env []string
	var committer *dt.Mailbox
	var committerDate *time.Time

	if args == nil {
		args = &CommitArgs{}
	}
	err = checkIdentity("Author", args.Author)
	if err != nil {
		goto end
	}
	err = checkIdentity("Committer", args.Committer)
	if err != nil {
		goto end
	}

	committer = args.Committer
	if committer == nil {
		committer = args.Author
	}
	committerDate = args.CommitterDate
	if committerDate == nil {
		committerDate = args.AuthorDate
	}
	env = appendIdentityEnv(env, "AUTHOR", args.Author, args.AuthorDate)
	env = appendIdentityEnv(env, "COMMITTER", committer, committerDate)

	cmd = []string{"commit", "--message=" + args.Message}
	if args.AllowEmpty {
		cmd = append(cmd, "--allow-empty")
	}
	_, err = r.run(ctx, nil, runArgs{
		Env:     env,
		Command: cmd,
		Stream:  true,
	})
end:
	if err != nil {
		err = NewErr(ErrCommit, err)
	}
	return err
}

// appendIdentityEnv appends GIT_<role>_NAME, GIT_<role>_EMAIL and
// GIT_<role>_DATE for whichever of identity and date are set.
func appendIdentityEnv(env []string, role string, identity *dt.Mailbox, date *time.Time) []string {
	if identity != nil {
		env = append(env,
			"GIT_"+role+"_NAME="+string(identity.DisplayName),
			"GIT_"+role+"_EMAIL="+string(identity.Email),
		)
	}
	if date != nil {
		env = append(env, "GIT_"+role+"_DATE="+gitDate(*date))
	}
	return env
}

// gitDate formats t in git's internal date format, "@<unix seconds> <zone>".
func gitDate(t time.Time) string {
	return fmt.Sprintf("@%d %s", t.Unix(), t.Format("-0700"))
}

// UnstageAll unstages every staged change under r.Root with
// git restore --staged. It does not need Open.
func (r *Repo) UnstageAll(ctx context.Context) (err error) {
	_, err = r.runGit(ctx, "restore", "--staged", ".")
	return err
}

// StageFilesArgs holds the arguments of StageFiles.
type StageFilesArgs struct {
	RepoArgs
	// Paths are the files or directories to stage. Empty stages nothing.
	Paths []dt.RelFilepath
}

// StageFiles stages args.Paths in dir with git add.
func StageFiles(ctx context.Context, dir dt.DirPath, args *StageFilesArgs) error {
	return NewRepo(dir, nil).StageFiles(ctx, args)
}

// StageFiles stages args.Paths with git add. It does not need Open.
func (r *Repo) StageFiles(ctx context.Context, args *StageFilesArgs) (err error) {
	var cmd []string

	if args == nil || len(args.Paths) == 0 {
		goto end
	}
	cmd = []string{"add", "--"}
	for _, p := range args.Paths {
		cmd = append(cmd, string(p))
	}
	_, err = r.run(ctx, &args.RepoArgs, runArgs{Command: cmd})
end:
	return err
}

// StageFileContentArgs holds the arguments of StageFileContent.
type StageFileContentArgs struct {
	RepoArgs
	// Path is the file whose index entry is replaced.
	Path dt.RelFilepath
	// Content is what the index entry holds.
	Content []byte
}

// StageFileContent stages args.Content for args.Path in dir.
func StageFileContent(ctx context.Context, dir dt.DirPath, args *StageFileContentArgs) error {
	return NewRepo(dir, nil).StageFileContent(ctx, args)
}

// StageFileContent stages specific content for a file using git plumbing
// commands, so only selected change blocks are committed. It writes the
// content as a blob with git hash-object, then stages the blob with git
// update-index. It does not need Open.
func (r *Repo) StageFileContent(ctx context.Context, args *StageFileContentArgs) (err error) {
	var blob string

	if args == nil {
		args = &StageFileContentArgs{}
	}
	blob, err = r.run(ctx, &args.RepoArgs, runArgs{
		Stdin:   append([]byte{}, args.Content...),
		Command: []string{"hash-object", "-w", "--stdin"},
	})
	if err != nil {
		err = NewErr(ErrStageFileContent, "step", "hash-object", "path", args.Path, err)
		goto end
	}

	_, err = r.run(ctx, &args.RepoArgs, runArgs{
		Command: []string{"update-index", "--add", "--cacheinfo",
			"100644," + strings.TrimSpace(blob) + "," + string(args.Path),
		},
	})
	if err != nil {
		err = NewErr(ErrStageFileContent, "step", "update-index", "path", args.Path, err)
		goto end
	}

end:
	return err
}

// UnstageFilesArgs holds the arguments of UnstageFiles.
type UnstageFilesArgs struct {
	RepoArgs
	// Files are the files to unstage. Empty unstages nothing.
	Files []dt.RelFilepath
}

// UnstageFiles unstages args.Files in dir.
func UnstageFiles(ctx context.Context, dir dt.DirPath, args *UnstageFilesArgs) error {
	return NewRepo(dir, nil).UnstageFiles(ctx, args)
}

// UnstageFiles removes files from the index without deleting them from the
// working tree, with `git reset HEAD <file>` for each file. A staged file
// added to .gitignore is not ignored until it is unstaged. It does not need
// Open.
func (r *Repo) UnstageFiles(ctx context.Context, args *UnstageFilesArgs) (err error) {
	if args == nil {
		goto end
	}
	for _, file := range args.Files {
		_, err = r.run(ctx, &args.RepoArgs, runArgs{
			Command: []string{"reset", "HEAD", "--", string(file)},
		})
		if err != nil {
			err = NewErr(ErrUnstageFile, err,
				"file", file,
				"repo_root", r.Root,
			)
			goto end
		}
	}
end:
	return err
}
