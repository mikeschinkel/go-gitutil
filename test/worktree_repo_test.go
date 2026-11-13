package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

// gitIn runs git in dir with a fixed identity, failing the test on error.
func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// newRepoWithWorktree returns a clone with one commit, which tracks an
// upstream, and a linked worktree of it on a new branch with no upstream.
func newRepoWithWorktree(t *testing.T) (clone, worktree string) {
	t.Helper()
	base := t.TempDir()
	origin := filepath.Join(base, "origin")
	clone = filepath.Join(base, "clone")
	worktree = filepath.Join(base, "wt")

	gitIn(t, base, "init", "-q", "-b", "main", origin)
	gitIn(t, origin, "commit", "-q", "--allow-empty", "-m", "init")
	gitIn(t, base, "clone", "-q", origin, clone)
	gitIn(t, clone, "worktree", "add", "-q", "-b", "feature", worktree)
	return clone, worktree
}

func TestInfoDir_ResolvesInWorktree(t *testing.T) {
	clone, worktree := newRepoWithWorktree(t)
	ctx := context.Background()

	want, err := filepath.EvalSymlinks(filepath.Join(clone, ".git", "info"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{clone, worktree} {
		got, err := gitutil.InfoDir(ctx, dt.DirPath(dir), nil)
		if err != nil {
			t.Fatalf("gitutil.InfoDir(%s): %v", dir, err)
		}
		resolved, err := filepath.EvalSymlinks(string(got))
		if err != nil {
			t.Fatalf("gitutil.InfoDir(%s) = %s, which does not exist: %v", dir, got, err)
		}
		if resolved != want {
			t.Errorf("gitutil.InfoDir(%s) = %s, want %s", dir, resolved, want)
		}
	}
}

func TestInfoStore_SaveLoadInWorktree(t *testing.T) {
	_, worktree := newRepoWithWorktree(t)

	store := gitutil.NewInfoStore(dt.DirPath(worktree), "gomion-test.json")
	err := store.Save([]byte("hello"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("Load = %q, want %q", got, "hello")
	}
}

func TestOpen_Remote(t *testing.T) {
	clone, worktree := newRepoWithWorktree(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		dir        string
		wantRemote gitutil.GitRemote
	}{
		{name: "tracks origin/main", dir: clone, wantRemote: gitutil.GitRemote{Name: "origin", Branch: "main"}},
		{name: "no upstream", dir: worktree, wantRemote: gitutil.GitRemote{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := gitutil.Open(ctx, dt.DirPath(tt.dir), nil)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			if repo.Remote != tt.wantRemote {
				t.Errorf("Remote = %+v, want %+v", repo.Remote, tt.wantRemote)
			}
		})
	}
}
