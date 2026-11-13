package test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

func TestInit_BranchIdentityConfig(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, &gitutil.InitArgs{
		Branch: "trunk",
		Config: map[string]string{
			"commit.gpgsign": "false",
			"core.autocrlf":  "false",
			"user.name":      "Overridden By Identity",
		},
	})
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("no .git: %v", err)
	}
	branch, err := repo.CurrentBranch(ctx)
	if err != nil {
		t.Fatalf("CurrentBranch on an unborn branch: %v", err)
	}
	if branch != "trunk" {
		t.Errorf("CurrentBranch = %q, want trunk", branch)
	}
	want := map[string]string{
		"user.name":      string(testIdentity.DisplayName),
		"user.email":     string(testIdentity.Email),
		"commit.gpgsign": "false",
		"core.autocrlf":  "false",
	}
	for key, value := range want {
		got, err := repo.GetConfig(ctx, key)
		if err != nil {
			t.Errorf("GetConfig(%s): %v", key, err)
			continue
		}
		if got != value {
			t.Errorf("GetConfig(%s) = %q, want %q", key, got, value)
		}
	}
}

func TestInit_ConfigWrittenInSortedOrder(t *testing.T) {
	_, dir := initRepo(t, &gitutil.InitArgs{
		Config: map[string]string{"zz.key": "1", "aa.key": "2", "mm.key": "3"},
	})
	b, err := os.ReadFile(filepath.Join(dir, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	order := []string{"[aa]", "[mm]", "[user]", "[zz]"}
	last := -1
	for _, section := range order {
		i := strings.Index(text, section)
		if i < 0 {
			t.Fatalf("config has no %s section:\n%s", section, text)
		}
		if i < last {
			t.Errorf("%s comes before the previous key; want sorted order:\n%s", section, text)
		}
		last = i
	}
}

func TestInit_RefusesIdentityWithoutDisplayName(t *testing.T) {
	dir := filepath.Join(tempDir(t), "repo")
	_, err := gitutil.Init(context.Background(), dt.DirPath(dir), &gitutil.InitArgs{
		Identity: &dt.Mailbox{Email: "nobody@example.com"},
	})
	if !errors.Is(err, gitutil.ErrIdentityNoDisplayName) {
		t.Fatalf("err = %v, want ErrIdentityNoDisplayName", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("Init ran git before refusing the identity")
	}
}

func TestInit_MethodOnNewRepo(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(tempDir(t), "repo")
	repo := gitutil.NewRepo(dt.DirPath(dir), nil)
	err := repo.Init(ctx, &gitutil.InitArgs{Branch: "main"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := run(t, repo, "symbolic-ref", "--short", "HEAD"); got != "main" {
		t.Errorf("HEAD = %q, want main", got)
	}
}

func TestPackageLevel_KeepsRepoArgs(t *testing.T) {
	ctx := context.Background()
	env := gitutil.RepoArgs{Env: configEnv("x.kept", "yes")}

	initRepo, dir := initRepo(t, &gitutil.InitArgs{RepoArgs: env})
	commitFile(t, initRepo, dir, "a.txt", "a")

	cloned, err := gitutil.Clone(ctx, dir, dt.DirPath(filepath.Join(tempDir(t), "clone")),
		&gitutil.CloneArgs{RepoArgs: env})
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	opened, err := gitutil.Open(ctx, dt.DirPath(dir), &gitutil.OpenArgs{RepoArgs: env})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	for name, repo := range map[string]*gitutil.Repo{"Init": initRepo, "Clone": cloned, "Open": opened} {
		if got := run(t, repo, "config", "x.kept"); got != "yes" {
			t.Errorf("%s's Repo: config x.kept = %q, want yes from its RepoArgs.Env", name, got)
		}
	}
}

func TestClone_Args(t *testing.T) {
	ctx := context.Background()
	src, srcDir := initRepo(t, nil)
	commitFile(t, src, srcDir, "a.txt", "a")
	run(t, src, "branch", "feature")
	base := tempDir(t)

	t.Run("Branch, Identity and Config", func(t *testing.T) {
		repo, err := gitutil.Clone(ctx, srcDir, dt.DirPath(filepath.Join(base, "branch")), &gitutil.CloneArgs{
			Branch:   "feature",
			Identity: &otherIdentity,
			Config:   map[string]string{"x.during": "clone"},
		})
		if err != nil {
			t.Fatal(err)
		}
		branch, err := repo.CurrentBranch(ctx)
		if err != nil || branch != "feature" {
			t.Errorf("CurrentBranch = %q, %v; want feature", branch, err)
		}
		for key, want := range map[string]string{
			"x.during":   "clone",
			"user.name":  string(otherIdentity.DisplayName),
			"user.email": string(otherIdentity.Email),
		} {
			got, err := repo.GetConfig(ctx, key)
			if err != nil || got != want {
				t.Errorf("GetConfig(%s) = %q, %v; want %q", key, got, err, want)
			}
		}
	})

	t.Run("Config applies during the clone", func(t *testing.T) {
		// A required filter that always fails breaks the clone's checkout
		// only if the -c config reaches it.
		_, err := gitutil.Clone(ctx, srcDir, dt.DirPath(filepath.Join(base, "during")), &gitutil.CloneArgs{
			Config: map[string]string{
				"filter.fail.smudge":   "false",
				"filter.fail.required": "true",
				"core.attributesFile":  writeAttributes(t),
			},
		})
		if err == nil {
			t.Errorf("Clone succeeded; the -c filter config did not apply during the checkout")
		}
	})

	t.Run("NoCheckout and Local", func(t *testing.T) {
		dir := filepath.Join(base, "nocheckout")
		_, err := gitutil.Clone(ctx, srcDir, dt.DirPath(dir), &gitutil.CloneArgs{Local: true, NoCheckout: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
			t.Errorf("a.txt is checked out; want no checkout")
		}
	})

	t.Run("Bare", func(t *testing.T) {
		dir := filepath.Join(base, "bare.git")
		repo, err := gitutil.Clone(ctx, srcDir, dt.DirPath(dir), &gitutil.CloneArgs{Bare: true})
		if err != nil {
			t.Fatal(err)
		}
		if got := run(t, repo, "rev-parse", "--is-bare-repository"); got != "true" {
			t.Errorf("is-bare-repository = %q, want true", got)
		}
	})

	t.Run("Method on NewRepo", func(t *testing.T) {
		repo := gitutil.NewRepo(dt.DirPath(filepath.Join(base, "method")), nil)
		err := repo.Clone(ctx, srcDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := run(t, repo, "log", "-1", "--format=%s"); got != "add a.txt" {
			t.Errorf("log = %q, want add a.txt", got)
		}
	})
}

// writeAttributes writes an attributes file routing every file through the
// "fail" filter and returns its path.
func writeAttributes(t *testing.T) string {
	t.Helper()
	dir := tempDir(t)
	writeFile(t, dir, "attributes", "* filter=fail\n")
	return filepath.Join(dir, "attributes")
}

func TestSetConfigGetConfig(t *testing.T) {
	ctx := context.Background()
	repo := gitutil.NewRepo(initRepoRoot(t), nil)
	err := repo.SetConfig(ctx, "core.hooksPath", "")
	if err != nil {
		t.Fatal(err)
	}
	err = repo.SetConfig(ctx, "x.y", "z")
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetConfig(ctx, "x.y")
	if err != nil || got != "z" {
		t.Errorf("GetConfig(x.y) = %q, %v; want z", got, err)
	}
	got, err = repo.GetConfig(ctx, "core.hooksPath")
	if err != nil || got != "" {
		t.Errorf("GetConfig(core.hooksPath) = %q, %v; want empty", got, err)
	}
	_, err = repo.GetConfig(ctx, "x.missing")
	if !errors.Is(err, gitutil.ErrConfigNotSet) {
		t.Errorf("GetConfig(x.missing) err = %v, want ErrConfigNotSet", err)
	}
}

// initRepoRoot returns the root of a new repo.
func initRepoRoot(t *testing.T) dt.DirPath {
	t.Helper()
	repo, _ := initRepo(t, nil)
	return repo.Root
}

func TestAdd(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	writeFile(t, dir, ".gitignore", "ignored.txt\n")
	writeFile(t, dir, "ignored.txt", "i")
	writeFile(t, dir, "keep/a.txt", "a")
	writeFile(t, dir, "skip/b.txt", "b")
	writeFile(t, dir, "skip2/c.txt", "c")

	err := repo.Add(ctx, &gitutil.AddArgs{All: true, Exclude: []dt.RelDirPath{"skip", "skip2/"}})
	if err != nil {
		t.Fatal(err)
	}
	staged := run(t, repo, "diff", "--cached", "--name-only")
	if staged != ".gitignore\nkeep/a.txt" {
		t.Errorf("staged = %q, want .gitignore and keep/a.txt", staged)
	}

	err = repo.Add(ctx, &gitutil.AddArgs{Paths: []dt.RelFilepath{"ignored.txt"}})
	if err == nil {
		t.Errorf("Add of an ignored file without Force succeeded")
	}
	err = repo.Add(ctx, &gitutil.AddArgs{Paths: []dt.RelFilepath{"ignored.txt"}, Force: true})
	if err != nil {
		t.Errorf("Add with Force: %v", err)
	}
	if !strings.Contains(run(t, repo, "diff", "--cached", "--name-only"), "ignored.txt") {
		t.Errorf("ignored.txt not staged with Force")
	}
}

func TestCommit_DeterministicWithFixedAuthorAndDate(t *testing.T) {
	ctx := context.Background()
	var ids []dt.GitObjectID
	for range 2 {
		repo, dir := initRepo(t, nil)
		writeFile(t, dir, "a.txt", "same")
		err := repo.Add(ctx, &gitutil.AddArgs{All: true})
		if err != nil {
			t.Fatal(err)
		}
		err = repo.Commit(ctx, &gitutil.CommitArgs{Message: "m", Author: &testIdentity, AuthorDate: &fixedDate})
		if err != nil {
			t.Fatal(err)
		}
		id, err := repo.RevParse(ctx, "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) == 0 {
			// The two commits must not share a clock second.
			time.Sleep(1100 * time.Millisecond)
		}
		ids = append(ids, id)
	}
	if ids[0] != ids[1] {
		t.Errorf("commit IDs differ: %s, %s", ids[0], ids[1])
	}
}

func TestCommit_IdentityAndDateDefaults(t *testing.T) {
	ctx := context.Background()
	committerDate := fixedDate.Add(time.Hour)

	tests := []struct {
		name string
		args gitutil.CommitArgs
		want string // %an|%ae|%cn|%ce|%at|%ct
	}{
		{
			name: "nil Committer uses Author, nil CommitterDate uses AuthorDate",
			args: gitutil.CommitArgs{Author: &otherIdentity, AuthorDate: &fixedDate},
			want: "Other User|other@example.com|Other User|other@example.com|1767323045|1767323045",
		},
		{
			name: "separate Committer and CommitterDate",
			args: gitutil.CommitArgs{
				Author: &otherIdentity, Committer: &testIdentity,
				AuthorDate: &fixedDate, CommitterDate: &committerDate,
			},
			want: "Other User|other@example.com|Test User|test@example.com|1767323045|1767326645",
		},
		{
			name: "nil Author lets git resolve it from config",
			args: gitutil.CommitArgs{AuthorDate: &fixedDate},
			want: "Test User|test@example.com|Test User|test@example.com|1767323045|1767323045",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := initRepo(t, nil)
			tt.args.Message = "m"
			tt.args.AllowEmpty = true
			err := repo.Commit(ctx, &tt.args)
			if err != nil {
				t.Fatal(err)
			}
			got := run(t, repo, "log", "-1", "--format=%an|%ae|%cn|%ce|%at|%ct")
			if got != tt.want {
				t.Errorf("got  %s\nwant %s", got, tt.want)
			}
		})
	}

	t.Run("nil AuthorDate lets git use the current time", func(t *testing.T) {
		repo, _ := initRepo(t, nil)
		before := time.Now().Add(-time.Minute).Unix()
		err := repo.Commit(ctx, &gitutil.CommitArgs{Message: "m", AllowEmpty: true, Author: &testIdentity})
		if err != nil {
			t.Fatal(err)
		}
		got, err := strconv.ParseInt(run(t, repo, "log", "-1", "--format=%at"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if got < before {
			t.Errorf("author time = %d, want about now (after %d)", got, before)
		}
	})
}

func TestCommit_RefusesIdentityWithoutDisplayName(t *testing.T) {
	ctx := context.Background()
	repo, _ := initRepo(t, nil)
	noName := &dt.Mailbox{Email: "nobody@example.com"}
	for name, args := range map[string]*gitutil.CommitArgs{
		"Author":    {Message: "m", AllowEmpty: true, Author: noName},
		"Committer": {Message: "m", AllowEmpty: true, Author: &testIdentity, Committer: noName},
	} {
		err := repo.Commit(ctx, args)
		if !errors.Is(err, gitutil.ErrIdentityNoDisplayName) {
			t.Errorf("%s: err = %v, want ErrIdentityNoDisplayName", name, err)
		}
	}
	if _, err := repo.CommitCount(ctx, "HEAD"); err == nil {
		t.Errorf("a commit was made; the identity must be refused before git runs")
	}
}

func TestCommit_StreamsToStdIO(t *testing.T) {
	ctx := context.Background()
	stdio, out, _ := newStdIO()
	repo, _ := initRepo(t, &gitutil.InitArgs{RepoArgs: gitutil.RepoArgs{StdIO: stdio}})
	out.Reset()
	err := repo.Commit(ctx, &gitutil.CommitArgs{Message: "streamed", AllowEmpty: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "streamed") {
		t.Errorf("StdIO output = %q, want the commit summary", out.String())
	}
}
