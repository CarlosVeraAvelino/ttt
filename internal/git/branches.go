package git

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Branch is a local branch and whether it is the checked-out one.
type Branch struct {
	Name    string
	Current bool
}

// LocalBranches lists the local branches in name order.
func LocalBranches(dir string) ([]Branch, error) {
	out, err := exec.Command("git", "-C", dir, "for-each-ref",
		"--format=%(HEAD)%09%(refname:short)", "refs/heads").Output()
	if err != nil {
		return nil, err
	}
	// Only newlines are trimmed: a leading space is the HEAD marker of a
	// branch that is not checked out.
	var branches []Branch
	for _, line := range strings.Split(strings.Trim(string(out), "\n"), "\n") {
		mark, name, ok := strings.Cut(line, "\t")
		if !ok || name == "" {
			continue
		}
		branches = append(branches, Branch{Name: name, Current: mark == "*"})
	}
	return branches, nil
}

// RemoteBranches lists the branches on origin, without the remote prefix.
func RemoteBranches(dir string) ([]string, error) {
	out, err := exec.Command("git", "-C", dir, "for-each-ref",
		"--format=%(refname:short)", "refs/remotes/origin").Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimPrefix(line, "origin/")
		if name == "" || name == "HEAD" || name == "origin" {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// SwitchBranch checks out an existing local branch.
func SwitchBranch(dir, name string) error {
	cmd := exec.Command("git", "-C", dir, "switch", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// RepoWebURL is the GitHub page for the repository at branch, or for the
// repository itself when branch is empty. It is empty without an origin.
func RepoWebURL(dir, branch string) string {
	remote := RemoteURL(dir)
	if remote == "" {
		return ""
	}
	base := remoteToHTTPS(remote)
	if branch == "" {
		return base
	}
	segments := strings.Split(branch, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return base + "/tree/" + strings.Join(segments, "/")
}

// CreatePullRequest opens a pull request from head into base with the gh CLI.
// A head branch that origin does not track yet is pushed first, because gh
// would otherwise stop to ask. It returns the new pull request's URL.
func CreatePullRequest(dir, base, head, title, body string) (string, error) {
	if !branchHasUpstream(dir, head) {
		if err := runGit(dir, "push", "--set-upstream", "origin", head); err != nil {
			return "", err
		}
	}
	cmd := exec.Command("gh", "pr", "create",
		"--base", base, "--head", head, "--title", title, "--body", body)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("gh pr create: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("gh pr create: %w", err)
	}
	lines := strings.Fields(string(out))
	if len(lines) == 0 {
		return "", errors.New("gh pr create printed no URL")
	}
	return lines[len(lines)-1], nil
}

func branchHasUpstream(dir, branch string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", branch+"@{u}")
	return cmd.Run() == nil
}

func runGit(dir string, args ...string) error {
	full := append([]string{"-C", dir}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CreateBranchFrom makes a branch called name starting at base and checks it
// out. base may be a local branch or a remote-tracking one such as origin/main.
func CreateBranchFrom(dir, name, base string) error {
	cmd := exec.Command("git", "-C", dir, "switch", "-c", name, base)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
