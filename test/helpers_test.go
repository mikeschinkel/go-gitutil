package test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

// TestMain keeps the machine's git config out of every test, so a global
// setting such as commit.gpgsign cannot change a result.
func TestMain(m *testing.M) {
	_ = dt.EnsureUserHomeDir()
	_ = os.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	os.Exit(m.Run())
}

var (
	testIdentity  = dt.Mailbox{DisplayName: "Test User", Email: "test@example.com"}
	otherIdentity = dt.Mailbox{DisplayName: "Other User", Email: "other@example.com"}
	fixedDate     = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
)

// gitPath returns the absolute path of the git on PATH, which Strict needs.
func gitPath(t *testing.T) dt.Filepath {
	t.Helper()
	p, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git not found: %v", err)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return dt.Filepath(abs)
}

// tempDir returns a new temporary directory with symlinks resolved, so it
// compares equal to the paths git reports.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeFile writes content to dir/rel, creating parent directories.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	err := os.MkdirAll(filepath.Dir(p), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(p, []byte(content), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

// initRepo returns a new repo on branch main with testIdentity in its config.
func initRepo(t *testing.T, args *gitutil.InitArgs) (*gitutil.Repo, string) {
	t.Helper()
	dir := filepath.Join(tempDir(t), "repo")
	if args == nil {
		args = &gitutil.InitArgs{}
	}
	if args.Branch == "" {
		args.Branch = "main"
	}
	if args.Identity == nil {
		args.Identity = &testIdentity
	}
	repo, err := gitutil.Init(context.Background(), dt.DirPath(dir), args)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return repo, dir
}

// commitFile writes rel, stages it and commits it with fixed identity and
// date.
func commitFile(t *testing.T, repo *gitutil.Repo, dir, rel, content string) {
	t.Helper()
	ctx := context.Background()
	writeFile(t, dir, rel, content)
	err := repo.Add(ctx, &gitutil.AddArgs{Paths: []dt.RelFilepath{dt.RelFilepath(rel)}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	err = repo.Commit(ctx, &gitutil.CommitArgs{
		Message:    "add " + rel,
		Author:     &testIdentity,
		AuthorDate: &fixedDate,
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// run runs repo.Run and fails the test on error.
func run(t *testing.T, repo *gitutil.Repo, args ...string) string {
	t.Helper()
	out, err := repo.Run(context.Background(), args...)
	if err != nil {
		t.Fatalf("Run(%v): %v", args, err)
	}
	return strings.TrimSpace(out)
}

// newStdIO returns a StdIO writing to its own buffers.
func newStdIO() (dt.StdIO, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return dt.NewStdIO(&out, &errOut), &out, &errOut
}

// configEnv returns environment entries that set one config key through
// GIT_CONFIG_COUNT, which every git process reads.
func configEnv(key, value string) []string {
	return []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=" + key,
		"GIT_CONFIG_VALUE_0=" + value,
	}
}
