package test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

func TestNewRepo_RunsNoGit(t *testing.T) {
	// A GitPath that does not exist would fail any git process.
	repo := gitutil.NewRepo("/nonexistent/dir", &gitutil.RepoArgs{GitPath: "/nonexistent/git"})
	if repo.Root != "/nonexistent/dir" {
		t.Errorf("Root = %q, want /nonexistent/dir", repo.Root)
	}
	if repo.Branch != "" {
		t.Errorf("Branch = %q, want empty", repo.Branch)
	}
}

func TestRun_UsesRepoSettings(t *testing.T) {
	repo, _ := initRepo(t, nil)
	repo = gitutil.NewRepo(repo.Root, &gitutil.RepoArgs{Env: configEnv("x.y", "from-repo")})
	if got := run(t, repo, "config", "x.y"); got != "from-repo" {
		t.Errorf("Run(config x.y) = %q, want from-repo", got)
	}
}

// untrackedStatus returns the porcelain status of a repo holding one
// untracked file, under the Repo's and the call's settings.
func untrackedStatus(t *testing.T, repoArgs, call gitutil.RepoArgs) (string, error) {
	t.Helper()
	repo, dir := initRepo(t, nil)
	writeFile(t, dir, "untracked.txt", "x")
	repo = gitutil.NewRepo(repo.Root, &repoArgs)
	return repo.Status(context.Background(), &gitutil.StatusArgs{RepoArgs: call})
}

func TestRepoArgs_CombineRules(t *testing.T) {
	git := gitPath(t)

	t.Run("StdIO: the call's replaces the Repo's", func(t *testing.T) {
		repo, dir := initRepo(t, nil)
		writeFile(t, dir, "untracked.txt", "x")
		repoIO, repoOut, _ := newStdIO()
		callIO, callOut, _ := newStdIO()
		repo = gitutil.NewRepo(repo.Root, &gitutil.RepoArgs{StdIO: repoIO})
		_, err := repo.Status(context.Background(), &gitutil.StatusArgs{
			HumanReadable: true,
			RepoArgs:      gitutil.RepoArgs{StdIO: callIO},
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(callOut.String(), "untracked.txt") {
			t.Errorf("call StdIO got %q, want the status", callOut.String())
		}
		if repoOut.Len() != 0 {
			t.Errorf("Repo StdIO got %q, want nothing", repoOut.String())
		}
	})

	t.Run("StdIO: a nil call keeps the Repo's", func(t *testing.T) {
		repo, dir := initRepo(t, nil)
		writeFile(t, dir, "untracked.txt", "x")
		repoIO, repoOut, _ := newStdIO()
		repo = gitutil.NewRepo(repo.Root, &gitutil.RepoArgs{StdIO: repoIO})
		_, err := repo.Status(context.Background(), &gitutil.StatusArgs{HumanReadable: true})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(repoOut.String(), "untracked.txt") {
			t.Errorf("Repo StdIO got %q, want the status", repoOut.String())
		}
	})

	t.Run("Env: the call's entries follow the Repo's", func(t *testing.T) {
		out, err := untrackedStatus(t,
			gitutil.RepoArgs{Env: configEnv("status.showUntrackedFiles", "no")},
			gitutil.RepoArgs{Env: configEnv("status.showUntrackedFiles", "normal")},
		)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "untracked.txt") {
			t.Errorf("status = %q, want the call's showUntrackedFiles=normal to win", out)
		}
		out, err = untrackedStatus(t,
			gitutil.RepoArgs{Env: configEnv("status.showUntrackedFiles", "no")},
			gitutil.RepoArgs{},
		)
		if err != nil {
			t.Fatal(err)
		}
		if out != "" {
			t.Errorf("status = %q, want the Repo's showUntrackedFiles=no to apply", out)
		}
	})

	t.Run("GitPath: the call's replaces the Repo's", func(t *testing.T) {
		_, err := untrackedStatus(t,
			gitutil.RepoArgs{GitPath: "/nonexistent/git"},
			gitutil.RepoArgs{GitPath: git},
		)
		if err != nil {
			t.Errorf("with the call's GitPath: %v", err)
		}
		_, err = untrackedStatus(t,
			gitutil.RepoArgs{GitPath: "/nonexistent/git"},
			gitutil.RepoArgs{},
		)
		if err == nil {
			t.Errorf("with only the Repo's bad GitPath: got no error")
		}
	})

	t.Run("Strict: a call can turn it on", func(t *testing.T) {
		_, err := untrackedStatus(t, gitutil.RepoArgs{}, gitutil.RepoArgs{Strict: true})
		if !errors.Is(err, gitutil.ErrStrictNeedsGitPath) {
			t.Errorf("err = %v, want ErrStrictNeedsGitPath", err)
		}
	})

	t.Run("Strict: a call cannot turn it off", func(t *testing.T) {
		_, err := untrackedStatus(t,
			gitutil.RepoArgs{Strict: true},
			gitutil.RepoArgs{GitPath: git, Env: []string{"X=1"}},
		)
		if err != nil {
			t.Errorf("strict Repo with the call's GitPath: %v", err)
		}
		_, err = untrackedStatus(t,
			gitutil.RepoArgs{Strict: true},
			gitutil.RepoArgs{Env: []string{"X=1"}},
		)
		if !errors.Is(err, gitutil.ErrStrictNeedsGitPath) {
			t.Errorf("err = %v, want ErrStrictNeedsGitPath: the call turned strict off", err)
		}
	})
}

func TestStrict_NeedsGitPath(t *testing.T) {
	dir := filepath.Join(tempDir(t), "repo")
	_, err := gitutil.Init(context.Background(), dt.DirPath(dir), &gitutil.InitArgs{
		RepoArgs: gitutil.RepoArgs{Strict: true},
	})
	if !errors.Is(err, gitutil.ErrStrictNeedsGitPath) {
		t.Fatalf("err = %v, want ErrStrictNeedsGitPath", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("Init created %s; strict without GitPath must fail before running git", dir)
	}
}

// hostileEnv sets a HOME whose .gitconfig hides untracked files and names a
// hostile user, a GIT_DIR pointing nowhere, and a GIT_AUTHOR_NAME, all of
// which would change what git does if they reached it.
func hostileEnv(t *testing.T) {
	t.Helper()
	home := tempDir(t)
	writeFile(t, home, ".gitconfig",
		"[status]\n\tshowUntrackedFiles = no\n[user]\n\tname = Hostile Home\n\temail = home@hostile.example\n")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_DIR", filepath.Join(home, "no-such-git-dir"))
	t.Setenv("GIT_AUTHOR_NAME", "Hostile Env")
	// TestMain points these away from HOME; unset them so HOME is read.
	for _, key := range []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM"} {
		value, ok := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		if ok {
			t.Cleanup(func() { _ = os.Setenv(key, value) })
		}
	}
}

func TestStrict_IsolatesFromEnvironment(t *testing.T) {
	ctx := context.Background()
	hostileEnv(t)
	strict := gitutil.RepoArgs{Strict: true, GitPath: gitPath(t)}
	base := tempDir(t)
	// Strict's HOME directories are made under TMPDIR; each must be removed.
	tmp := tempDir(t)
	t.Setenv("TMPDIR", tmp)
	t.Cleanup(func() {
		entries, err := os.ReadDir(tmp)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			t.Errorf("strict HOME directory left behind: %s", e.Name())
		}
	})

	// The control: without Strict, the hostile GIT_DIR reaches git, which
	// puts the repository there instead of in control/.git.
	control := filepath.Join(base, "control")
	_, err := gitutil.Init(ctx, dt.DirPath(control), &gitutil.InitArgs{Branch: "main"})
	if err != nil {
		t.Fatalf("non-strict Init: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(control, ".git")); statErr == nil {
		t.Fatalf("control: GIT_DIR did not reach non-strict git, so this test proves nothing")
	}

	dir := filepath.Join(base, "repo")
	repo, err := gitutil.Init(ctx, dt.DirPath(dir), &gitutil.InitArgs{RepoArgs: strict, Branch: "main"})
	if err != nil {
		t.Fatalf("strict Init with a hostile GIT_DIR: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr != nil {
		t.Fatalf("strict Init did not create %s/.git: GIT_DIR reached git", dir)
	}

	// HOME is an empty directory of go-gitutil's under TMPDIR, not the
	// hostile one; git expands ~ from it.
	home := run(t, repo, "-c", "x.p=~/f", "config", "--path", "--get", "x.p")
	if !strings.HasPrefix(home, filepath.Join(tmp, "gitutil-home-")) {
		t.Errorf("strict HOME expands ~/f to %q, want a directory under %s", home, tmp)
	}

	// HOME's .gitconfig does not reach git.
	writeFile(t, dir, "untracked.txt", "x")
	out, err := repo.Status(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "untracked.txt") {
		t.Errorf("status = %q: HOME's showUntrackedFiles=no reached git", out)
	}
	_, err = repo.Run(ctx, "config", "--get", "user.name")
	if err == nil {
		t.Errorf("user.name resolved under Strict; HOME's .gitconfig reached git")
	}

	// A commit with no identity fails, rather than guessing one or taking
	// HOME's or GIT_AUTHOR_NAME.
	err = repo.Add(ctx, &gitutil.AddArgs{All: true})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Commit(ctx, &gitutil.CommitArgs{Message: "no identity"})
	if err == nil {
		t.Fatalf("strict Commit with no identity succeeded")
	}

	// GIT_AUTHOR_NAME does not reach git.
	err = repo.Commit(ctx, &gitutil.CommitArgs{Message: "with identity", Author: &testIdentity})
	if err != nil {
		t.Fatalf("strict Commit with Author: %v", err)
	}
	if got := run(t, repo, "log", "-1", "--format=%an"); got != string(testIdentity.DisplayName) {
		t.Errorf("author = %q, want %q: GIT_AUTHOR_NAME reached git", got, testIdentity.DisplayName)
	}

	// Only file remotes are allowed.
	_, err = gitutil.Clone(ctx, "https://example.invalid/repo.git", dt.DirPath(filepath.Join(base, "https")),
		&gitutil.CloneArgs{RepoArgs: strict})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("strict https clone: err = %v, want transport not allowed", err)
	}
	_, err = gitutil.Clone(ctx, "file://"+dir, dt.DirPath(filepath.Join(base, "file")),
		&gitutil.CloneArgs{RepoArgs: strict})
	if err != nil {
		t.Errorf("strict file:// clone: %v", err)
	}
}

func TestStrict_ControlHostileHomeReachesNonStrictGit(t *testing.T) {
	// Shows TestStrict_IsolatesFromEnvironment's hostile HOME is real: a
	// non-strict process reads it.
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	hostileEnv(t)
	t.Setenv("GIT_DIR", filepath.Join(dir, ".git"))
	writeFile(t, dir, "untracked.txt", "x")
	out, err := repo.Status(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("non-strict status = %q, want HOME's showUntrackedFiles=no to hide the file", out)
	}
}
