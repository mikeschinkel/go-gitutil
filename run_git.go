package gitutils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

var ErrGitCommandFailed = errors.New("git command failed")

type GitRunner struct {
	RepoRoot dt.DirPath
}

func newGitRunner(repoRoot dt.DirPath) GitRunner {
	return GitRunner{RepoRoot: repoRoot}
}

type runGitArgs struct {
	NULSeparated bool
	Stdin        []byte // optional, nil means no stdin
	Command      []string
}

func (r GitRunner) runGit(ctx context.Context, args runGitArgs) (out string, err error) {
	cmd := exec.CommandContext(ctx, "git", args.Command...)
	if r.RepoRoot != "" {
		cmd.Dir = string(r.RepoRoot)
	}

	if len(args.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(args.Stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		err = NewErr(
			ErrGitCommandFailed,
			"command", fmt.Sprintf("git %s", strings.Join(args.Command, " ")),
			err,
		)
		s := strings.TrimSpace(stderr.String())
		if s != "" {
			err = WithErr(err, "stderr", s)
		}
		goto end
	}

	out = stdout.String()

	// Default behavior: line-oriented output
	if !args.NULSeparated {
		out = strings.TrimRight(out, "\r\n")
	}
end:
	return out, err
}
