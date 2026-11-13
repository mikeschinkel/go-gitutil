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

func TestOpen_RemoteAndBranchNamesWithSlash(t *testing.T) {
	ctx := context.Background()
	origin, originDir := initRepo(t, nil)
	commitFile(t, origin, originDir, "a.txt", "a")
	run(t, origin, "branch", "release/v1")

	clone, err := gitutil.Clone(ctx, originDir, dt.DirPath(filepath.Join(tempDir(t), "clone")), nil)
	if err != nil {
		t.Fatal(err)
	}
	run(t, clone, "remote", "add", "team/upstream", originDir)
	run(t, clone, "fetch", "-q", "team/upstream")
	run(t, clone, "checkout", "-q", "-b", "feature/x", "--track", "team/upstream/release/v1")

	err = clone.Open(ctx, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if clone.Branch != "feature/x" {
		t.Errorf("Branch = %q, want feature/x", clone.Branch)
	}
	want := gitutil.GitRemote{Name: "team/upstream", Branch: "release/v1"}
	if clone.Remote != want {
		t.Errorf("Remote = %+v, want %+v", clone.Remote, want)
	}
}

func TestOpen_UnbornAndDetached(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)

	err := repo.Open(ctx, nil)
	if err != nil {
		t.Fatalf("Open on an unborn branch: %v", err)
	}
	if repo.Branch != "main" {
		t.Errorf("Branch = %q, want main", repo.Branch)
	}

	commitFile(t, repo, dir, "a.txt", "a")
	run(t, repo, "checkout", "-q", "--detach")
	err = repo.Open(ctx, nil)
	if err != nil {
		t.Fatalf("Open on a detached HEAD: %v", err)
	}
	if repo.Branch != "" || repo.Remote != (gitutil.GitRemote{}) {
		t.Errorf("Branch, Remote = %q, %+v; want empty for a detached HEAD", repo.Branch, repo.Remote)
	}
}

func TestOpen_SetsRootToTopLevel(t *testing.T) {
	ctx := context.Background()
	_, dir := initRepo(t, nil)
	sub := filepath.Join(dir, "sub")
	err := os.MkdirAll(sub, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := gitutil.Open(ctx, dt.DirPath(sub), nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(repo.Root) != dir {
		t.Errorf("Root = %s, want %s", repo.Root, dir)
	}
	_, err = gitutil.Open(ctx, dt.DirPath(tempDir(t)), nil)
	if !errors.Is(err, gitutil.ErrNotGitRepo) {
		t.Errorf("Open outside a repo: err = %v, want ErrNotGitRepo", err)
	}
}

func TestUnopenedRepo(t *testing.T) {
	ctx := context.Background()
	opened, dir := initRepo(t, nil)
	commitFile(t, opened, dir, "a.txt", "a")
	repo := gitutil.NewRepo(dt.DirPath(dir), nil)

	needOpen := map[string]func() error{
		"Push":            func() error { _, err := repo.Push(ctx); return err },
		"PushAll":         func() error { _, err := repo.PushAll(ctx); return err },
		"PushTags":        func() error { _, err := repo.PushTags(ctx); return err },
		"PushTag":         func() error { return repo.PushTag(ctx, "v1.0.0") },
		"DeleteRemoteTag": func() error { return repo.DeleteRemoteTag(ctx, "v1.0.0") },
		"FetchTags":       func() error { return repo.FetchTags(ctx) },
		"RemoteTags":      func() error { _, err := repo.RemoteTags(ctx, ""); return err },
		"CompareRemoteTags": func() error {
			_, err := repo.CompareRemoteTags(ctx, "")
			return err
		},
		"CurrentRemoteURL":   func() error { _, err := repo.CurrentRemoteURL(ctx); return err },
		"OpenCachedWorktree": func() error { _, err := repo.OpenCachedWorktree(ctx); return err },
	}
	for name, fn := range needOpen {
		if err := fn(); !errors.Is(err, gitutil.ErrRepoNotOpen) {
			t.Errorf("%s on an unopened Repo: err = %v, want ErrRepoNotOpen", name, err)
		}
	}

	noOpen := map[string]func() error{
		"Status":        func() error { _, err := repo.Status(ctx, nil); return err },
		"CurrentBranch": func() error { _, err := repo.CurrentBranch(ctx); return err },
		"RevParse":      func() error { _, err := repo.RevParse(ctx, "HEAD"); return err },
		"CommitCount":   func() error { _, err := repo.CommitCount(ctx, "HEAD"); return err },
		"Tags":          func() error { _, err := repo.Tags(ctx, ""); return err },
		"Branches":      func() error { _, err := repo.Branches(ctx); return err },
		"ListWorktrees": func() error { _, err := repo.ListWorktrees(ctx); return err },
		"CreateTag": func() error {
			return repo.CreateTag(ctx, gitutil.CreateTagArgs{TagName: "v1.0.0", Message: "v1"})
		},
	}
	for name, fn := range noOpen {
		if err := fn(); err != nil {
			t.Errorf("%s on an unopened Repo: %v", name, err)
		}
	}
}

func TestPackageLevelWrappers(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	commitFile(t, repo, dir, "a.txt", "a")
	root := dt.DirPath(dir)
	stdio, out, _ := newStdIO()

	writeFile(t, dir, "b.txt", "b")
	status, err := gitutil.Status(ctx, root, &gitutil.StatusArgs{
		HumanReadable: true,
		RepoArgs:      gitutil.RepoArgs{StdIO: stdio},
	})
	if err != nil || !strings.Contains(status, "b.txt") || !strings.Contains(out.String(), "b.txt") {
		t.Errorf("Status = %q, %v; StdIO got %q; want b.txt in both", status, err, out.String())
	}

	err = gitutil.StageFiles(ctx, root, &gitutil.StageFilesArgs{Paths: []dt.RelFilepath{"b.txt"}})
	if err != nil {
		t.Fatalf("StageFiles: %v", err)
	}
	err = gitutil.UnstageFiles(ctx, root, &gitutil.UnstageFilesArgs{Files: []dt.RelFilepath{"b.txt"}})
	if err != nil {
		t.Fatalf("UnstageFiles: %v", err)
	}
	if got := run(t, repo, "diff", "--cached", "--name-only"); got != "" {
		t.Errorf("staged after UnstageFiles = %q, want nothing", got)
	}

	err = gitutil.StageFileContent(ctx, root, &gitutil.StageFileContentArgs{Path: "a.txt", Content: []byte("staged")})
	if err != nil {
		t.Fatalf("StageFileContent: %v", err)
	}
	if got := run(t, repo, "show", ":a.txt"); got != "staged" {
		t.Errorf("index a.txt = %q, want staged", got)
	}

	dest := tempDir(t)
	err = gitutil.ExportStagedFiles(ctx, root, &gitutil.ExportStagedArgs{DestDir: dt.DirPath(dest)})
	if err != nil {
		t.Fatalf("ExportStagedFiles: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dest, "a.txt")); err != nil || string(b) != "staged" {
		t.Errorf("exported a.txt = %q, %v; want staged", b, err)
	}

	err = repo.UnstageAll(ctx)
	if err != nil {
		t.Fatalf("UnstageAll: %v", err)
	}
	if got := run(t, repo, "diff", "--cached", "--name-only"); got != "" {
		t.Errorf("staged after UnstageAll = %q, want nothing", got)
	}

	writeFile(t, dir, ".gitignore", "*.log\n")
	writeFile(t, dir, "x.log", "x")
	ignored, err := gitutil.IgnoredFiles(ctx, root, nil)
	if err != nil || len(ignored) != 1 || ignored[0] != "x.log" {
		t.Errorf("IgnoredFiles = %v, %v; want [x.log]", ignored, err)
	}
	files, err := gitutil.LsFilesRaw(ctx, root, &gitutil.LsFilesRawArgs{Args: []string{"-c"}})
	if err != nil || len(files) != 1 || files[0] != "a.txt" {
		t.Errorf("LsFilesRaw(-c) = %v, %v; want [a.txt]", files, err)
	}

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	top, err := gitutil.FindRepoRoot(ctx, dt.DirPath(sub), nil)
	if err != nil || string(top) != dir {
		t.Errorf("FindRepoRoot = %s, %v; want %s", top, err, dir)
	}
	info, err := gitutil.InfoDir(ctx, root, &gitutil.InfoDirArgs{})
	if err != nil || string(info) != filepath.Join(dir, ".git", "info") {
		t.Errorf("InfoDir = %s, %v; want %s", info, err, filepath.Join(dir, ".git", "info"))
	}
}

func TestAddWorktree(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	commitFile(t, repo, dir, "a.txt", "a")
	run(t, repo, "branch", "existing")
	base := tempDir(t)

	tests := []struct {
		name    string
		args    gitutil.AddWorktreeArgs
		wantErr bool
	}{
		{name: "new branch", args: gitutil.AddWorktreeArgs{Branch: "fresh", Base: "main"}},
		{name: "existing branch without ReuseBranch fails", args: gitutil.AddWorktreeArgs{Branch: "existing"}, wantErr: true},
		{name: "existing branch with ReuseBranch", args: gitutil.AddWorktreeArgs{Branch: "existing", ReuseBranch: true}},
		{name: "new branch with ReuseBranch", args: gitutil.AddWorktreeArgs{Branch: "fresh2", ReuseBranch: true}},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(base, "wt", string(rune('a'+i)))
			tt.args.Path = dt.DirPath(path)
			err := repo.AddWorktree(ctx, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("AddWorktree succeeded, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("AddWorktree: %v", err)
			}
			branch, err := gitutil.NewRepo(dt.DirPath(path), nil).CurrentBranch(ctx)
			if err != nil || branch != tt.args.Branch {
				t.Errorf("worktree branch = %q, %v; want %q", branch, err, tt.args.Branch)
			}
		})
	}

	worktrees, err := repo.ListWorktrees(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var branches []string
	for _, wt := range worktrees {
		branches = append(branches, string(wt.Branch))
		if len(wt.HEAD) != 40 {
			t.Errorf("worktree %s HEAD = %q, want a full object ID", wt.Path, wt.HEAD)
		}
	}
	if got := strings.Join(branches, ","); got != "main,fresh,existing,fresh2" {
		t.Errorf("ListWorktrees branches = %s, want main,fresh,existing,fresh2", got)
	}
}

func TestCommitCountAndTreeEntry(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	commitFile(t, repo, dir, "a.txt", "a")
	commitFile(t, repo, dir, "dir/b.txt", "b")

	count, err := repo.CommitCount(ctx, "HEAD")
	if err != nil || count != 2 {
		t.Errorf("CommitCount(HEAD) = %d, %v; want 2", count, err)
	}
	count, err = repo.CommitCount(ctx, "HEAD~1")
	if err != nil || count != 1 {
		t.Errorf("CommitCount(HEAD~1) = %d, %v; want 1", count, err)
	}

	// A nested repo added with Force becomes a gitlink.
	nested, nestedDir := initRepo(t, nil)
	commitFile(t, nested, nestedDir, "n.txt", "n")
	nestedPath := filepath.Join(dir, "nested")
	err = os.Rename(nestedDir, nestedPath)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Add(ctx, &gitutil.AddArgs{All: true, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Commit(ctx, &gitutil.CommitArgs{Message: "nested", Author: &testIdentity, AuthorDate: &fixedDate})
	if err != nil {
		t.Fatal(err)
	}
	nestedHead, err := gitutil.NewRepo(dt.DirPath(nestedPath), nil).RevParse(ctx, "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path dt.RelFilepath
		want gitutil.TreeEntry
	}{
		{path: "nested", want: gitutil.TreeEntry{Mode: gitutil.GitlinkMode, Type: "commit", ObjectID: nestedHead, Path: "nested"}},
		{path: "dir", want: gitutil.TreeEntry{Mode: "040000", Type: "tree", Path: "dir"}},
		{path: "dir/b.txt", want: gitutil.TreeEntry{Mode: "100644", Type: "blob", Path: "dir/b.txt"}},
	}
	for _, tt := range tests {
		got, err := repo.TreeEntry(ctx, "HEAD", tt.path)
		if err != nil {
			t.Errorf("TreeEntry(%s): %v", tt.path, err)
			continue
		}
		if tt.want.ObjectID == "" {
			tt.want.ObjectID = got.ObjectID
			if len(got.ObjectID) != 40 {
				t.Errorf("TreeEntry(%s).ObjectID = %q, want a full object ID", tt.path, got.ObjectID)
			}
		}
		if got != tt.want {
			t.Errorf("TreeEntry(%s) = %+v, want %+v", tt.path, got, tt.want)
		}
	}
	_, err = repo.TreeEntry(ctx, "HEAD", "missing.txt")
	if !errors.Is(err, gitutil.ErrTreeEntryNotFound) {
		t.Errorf("TreeEntry(missing.txt) err = %v, want ErrTreeEntryNotFound", err)
	}
}

func TestOpenCachedWorktree(t *testing.T) {
	ctx := context.Background()
	repo, dir := initRepo(t, nil)
	commitFile(t, repo, dir, "a.txt", "a")
	t.Setenv("NEXTVER_CACHE_DIR", filepath.Join(tempDir(t), "cache"))

	err := repo.Open(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 { // the second run reuses the cached clone
		wt, err := repo.OpenCachedWorktree(ctx)
		if err != nil {
			t.Fatalf("OpenCachedWorktree: %v", err)
		}
		err = wt.Checkout(ctx, "refs/remotes/source/main")
		if err != nil {
			t.Fatalf("Checkout: %v", err)
		}
		b, err := os.ReadFile(filepath.Join(string(wt.Dir), "a.txt"))
		if err != nil || string(b) != "a" {
			t.Errorf("cached a.txt = %q, %v; want a", b, err)
		}
		err = wt.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
