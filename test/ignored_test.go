package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-gitutil"
)

func TestRepoIgnored(t *testing.T) {
	repo, dir := initRepo(t, nil)
	writeFile(t, dir, ".gitignore", "build/\nsub/tmp\n*.cache/\n")
	for _, d := range []string{"build", "sub/tmp", "sub/keep", "src", "x.cache", "deep/build"} {
		writeFile(t, dir, filepath.Join(d, "f.txt"), "x")
	}

	paths := []dt.RelDirPath{"src", "build", "sub/keep", "sub/tmp", "x.cache/", "deep/build", "missing"}
	got, err := repo.Ignored(context.Background(), paths)
	if err != nil {
		t.Fatalf("Ignored: %v", err)
	}
	want := []dt.RelDirPath{"build", "sub/tmp", "x.cache/", "deep/build"}
	if !slices.Equal(got, want) {
		t.Errorf("Ignored = %v, want %v", got, want)
	}
}

func TestRepoIgnoredNone(t *testing.T) {
	repo, dir := initRepo(t, nil)
	writeFile(t, dir, ".gitignore", "build/\n")
	writeFile(t, dir, "src/f.txt", "x")

	got, err := repo.Ignored(context.Background(), []dt.RelDirPath{"src", "other"})
	if err != nil {
		t.Fatalf("Ignored: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Ignored = %v, want none", got)
	}

	got, err = repo.Ignored(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("Ignored(nil) = %v, %v; want none, nil", got, err)
	}
}

// TestRepoIgnoredOneProcess checks that Ignored starts one git process for
// many paths, by running git through a wrapper that logs each run.
func TestRepoIgnoredOneProcess(t *testing.T) {
	_, dir := initRepo(t, nil)
	writeFile(t, dir, ".gitignore", "build*/\n")

	paths := make([]dt.RelDirPath, 0, 50)
	for i := range 50 {
		name := "keep"
		if i%2 == 0 {
			name = "build"
		}
		paths = append(paths, dt.RelDirPath(fmt.Sprintf("%s%02d", name, i)))
	}

	bin := tempDir(t)
	log := filepath.Join(bin, "runs.log")
	wrapper := "#!/bin/sh\necho run >> '" + log + "'\nexec '" + string(gitPath(t)) + "' \"$@\"\n"
	err := os.WriteFile(filepath.Join(bin, "git-logged"), []byte(wrapper), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	logged := gitutil.NewRepo(dt.DirPath(dir), &gitutil.RepoArgs{
		GitPath: dt.Filepath(filepath.Join(bin, "git-logged")),
	})
	got, err := logged.Ignored(context.Background(), paths)
	if err != nil {
		t.Fatalf("Ignored: %v", err)
	}
	if len(got) != 25 {
		t.Errorf("Ignored returned %d paths, want 25: %v", len(got), got)
	}
	runs, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(runs), "run"); n != 1 {
		t.Errorf("git ran %d times, want 1", n)
	}
}
