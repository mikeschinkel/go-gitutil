package gitutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// RepoArgs holds the settings every git process of a Repo runs with.
//
// A Repo keeps the RepoArgs it was built with for its whole life. The Args
// type of an operation that exists both as a Repo method and as a
// package-level function embeds RepoArgs; when a method gets a non-zero one,
// it combines with the Repo's for that call only:
//
//   - StdIO: the call's value replaces the Repo's when it is non-nil.
//   - Env: the call's entries come after the Repo's, so a repeated key takes
//     the call's value.
//   - GitPath: the call's value replaces the Repo's when it is non-empty.
//   - Strict: true turns isolated mode on for the call; a call cannot turn a
//     strict Repo's isolation off.
type RepoArgs struct {
	// StdIO receives the output of commands that print for a person, such
	// as commit, clone and push. Nil means that output is captured and not
	// shown. Output go-gitutil parses is always captured.
	StdIO dt.StdIO

	// Env holds KEY=VALUE entries applied after the base environment: the
	// current process's environment, or in Strict mode the isolated one.
	Env []string

	// GitPath is the git binary to run. Empty means git is found on PATH.
	GitPath dt.Filepath

	// Strict runs git isolated from the machine. Each git process gets an
	// environment built from nothing: HOME set to an empty directory
	// go-gitutil creates and removes, GIT_CONFIG_GLOBAL=/dev/null,
	// GIT_CONFIG_NOSYSTEM=1, GIT_ALLOW_PROTOCOL=file, GIT_TERMINAL_PROMPT=0,
	// then Env. Each also gets -c user.useConfigOnly=true, so git never
	// guesses an identity. Only local remotes work, and PATH is not passed,
	// so Strict requires GitPath and fails before running git without it.
	Strict bool
}

// combine returns ra with call's non-zero settings applied, under the rules
// in RepoArgs' doc comment. A nil call returns ra unchanged.
func (ra RepoArgs) combine(call *RepoArgs) (out RepoArgs) {
	out = ra
	if call == nil {
		goto end
	}
	if call.StdIO != nil {
		out.StdIO = call.StdIO
	}
	if len(call.Env) > 0 {
		out.Env = append(append([]string{}, ra.Env...), call.Env...)
	}
	if call.GitPath != "" {
		out.GitPath = call.GitPath
	}
	if call.Strict {
		out.Strict = true
	}
end:
	return out
}

// strictEnv is the isolated environment of a Strict process, before
// RepoArgs.Env and HOME are added.
var strictEnv = []string{
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_CONFIG_NOSYSTEM=1",
	"GIT_ALLOW_PROTOCOL=file",
	"GIT_TERMINAL_PROMPT=0",
}

// runArgs describes one git process.
type runArgs struct {
	// Dir is the working directory. Empty means the current process's.
	Dir dt.DirPath
	// Env holds KEY=VALUE entries applied after RepoArgs.Env.
	Env []string
	// Stdin is the process's input. Nil means no input.
	Stdin []byte
	// Stream sends output to RepoArgs.StdIO, when it is set, as well as
	// capturing it. Set it for commands that print for a person.
	Stream bool
	// NULSeparated keeps trailing line endings, for -z output.
	NULSeparated bool
	// Command holds git's arguments.
	Command []string
}

// run runs one git process with settings ra. Every git process go-gitutil
// starts goes through it. It returns stdout with trailing line endings
// trimmed, unless args.NULSeparated is set.
func run(ctx context.Context, ra RepoArgs, args runArgs) (out string, err error) {
	var cmd *exec.Cmd
	var stdout, stderr bytes.Buffer
	var gitPath string
	var env []string
	var gitArgs []string
	var home string

	gitPath = string(ra.GitPath)
	if gitPath == "" {
		if ra.Strict {
			err = NewErr(ErrStrictNeedsGitPath)
			goto end
		}
		gitPath = "git"
	}

	gitArgs = args.Command
	switch ra.Strict {
	case true:
		home, err = os.MkdirTemp("", "gitutil-home-")
		if err != nil {
			err = NewErr(ErrStrictHome, err)
			goto end
		}
		defer func() {
			err = CombineErrs([]error{err, os.RemoveAll(home)})
		}()
		env = append([]string{"HOME=" + home}, strictEnv...)
		gitArgs = append([]string{"-c", "user.useConfigOnly=true"}, gitArgs...)
	default:
		env = os.Environ()
	}
	env = append(env, ra.Env...)
	env = append(env, args.Env...)

	cmd = exec.CommandContext(ctx, gitPath, gitArgs...)
	cmd.Dir = string(args.Dir)
	cmd.Env = env
	if args.Stdin != nil {
		cmd.Stdin = bytes.NewReader(args.Stdin)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if args.Stream && ra.StdIO != nil {
		cmd.Stdout = teeWriter(&stdout, ra.StdIO.Output())
		cmd.Stderr = teeWriter(&stderr, ra.StdIO.ErrOut())
	}

	err = cmd.Run()
	if err != nil {
		err = NewErr(
			ErrGitCommandFailed,
			"command", "git "+strings.Join(gitArgs, " "),
			"dir", args.Dir,
			err,
		)
		s := strings.TrimSpace(stderr.String())
		if s != "" {
			err = WithErr(err, "stderr", s)
		}
		goto end
	}

	out = stdout.String()
	if !args.NULSeparated {
		out = strings.TrimRight(out, "\r\n")
	}
end:
	return out, err
}

// teeWriter writes to buf and to w, or to buf alone when w is nil.
func teeWriter(buf *bytes.Buffer, w io.Writer) io.Writer {
	if w == nil {
		return buf
	}
	return io.MultiWriter(buf, w)
}

// exitCode returns the exit code of the git process err came from, or -1
// when err did not come from a process that exited.
func exitCode(err error) (code int) {
	var ee *exec.ExitError
	code = -1
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	}
	return code
}
