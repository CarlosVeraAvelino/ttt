package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBranchesMarksCurrentAndSortsByName(t *testing.T) {
	dir := setupTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "a.txt")
	gitRun(t, dir, "commit", "-m", "first")
	gitRun(t, dir, "branch", "zeta")
	gitRun(t, dir, "branch", "alpha")

	branches, err := LocalBranches(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	current := ""
	for _, b := range branches {
		names = append(names, b.Name)
		if b.Current {
			current = b.Name
		}
	}
	if len(names) != 3 || names[0] != "alpha" || names[2] != "zeta" {
		t.Fatalf("branches = %v, want alpha first and zeta last of three", names)
	}
	if current == "" {
		t.Fatal("no branch marked current")
	}
}

func TestSwitchBranchChecksOutExistingBranch(t *testing.T) {
	dir := setupTestRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "a.txt")
	gitRun(t, dir, "commit", "-m", "first")
	gitRun(t, dir, "branch", "feature/x")

	if err := SwitchBranch(dir, "feature/x"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}
	if got := BranchName(dir); got != "feature/x" {
		t.Fatalf("current branch = %q, want feature/x", got)
	}
	if err := SwitchBranch(dir, "does-not-exist"); err == nil {
		t.Fatal("SwitchBranch to a missing branch succeeded")
	}
}

func TestRepoWebURLBuildsTreeLinkPerBranch(t *testing.T) {
	dir := setupTestRepo(t)
	gitRun(t, dir, "remote", "add", "origin", "git@github.com:someone/project.git")

	if got, want := RepoWebURL(dir, ""), "https://github.com/someone/project"; got != want {
		t.Fatalf("RepoWebURL(root) = %q, want %q", got, want)
	}
	if got, want := RepoWebURL(dir, "feature/a b"), "https://github.com/someone/project/tree/feature/a%20b"; got != want {
		t.Fatalf("RepoWebURL(branch) = %q, want %q", got, want)
	}
}

func TestRepoWebURLWithoutOriginIsEmpty(t *testing.T) {
	dir := setupTestRepo(t)
	if got := RepoWebURL(dir, "main"); got != "" {
		t.Fatalf("RepoWebURL without origin = %q, want empty", got)
	}
}
