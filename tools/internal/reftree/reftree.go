// Package reftree finds the module a tool runs in, reads a tree's module path,
// and materialises another git ref's tree beside it, for the tools that
// compare the working tree against a base (`go tool surface -base`, `go tool
// restatements -base`).
//
// The ref's tree is checked out in a temporary detached git worktree (`git
// worktree add --detach`), read, and removed again (`git worktree remove
// --force`), so the caller's checkout and branch are never touched —
// docs/WORKFLOW.md's "one writer per checkout" forbids stashing or checking
// out in place. Only the ref itself is resolved, never a merge base: sessions
// run in shallow clones, where a merge base can be missing.
package reftree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ModuleRoot walks up from the current working directory until it finds the
// directory containing go.mod, so a tool works the same way whether it is run
// from the module root or from a package directory beneath it.
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from working directory")
		}
		dir = parent
	}
}

// ModulePath extracts the `module` directive from the go.mod file at root,
// without a full go.mod parser: a hand-rolled tool has no need for the
// replace/require directives a module graph resolver would.
func ModulePath(root string) (string, error) {
	path := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		mod, ok := strings.CutPrefix(strings.TrimSpace(line), "module ")
		if !ok {
			continue
		}
		return strings.TrimSpace(mod), nil
	}
	return "", fmt.Errorf("no module directive found in %s", path)
}

// Checkout materialises ref beside the repository rooted at repoRoot, as the
// package doc states, returning the tree's path and a remove func the caller
// must defer, which reports a failed removal as its error.
func Checkout(repoRoot, ref string) (string, func() error, error) {
	dir, err := os.MkdirTemp("", "goxsd8-reftree-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir for %s worktree: %w", ref, err)
	}

	add := exec.Command("git", "worktree", "add", "--detach", dir, ref)
	add.Dir = repoRoot
	if out, err := add.CombinedOutput(); err != nil {
		_ = os.RemoveAll(dir) // best effort: add failed, nothing was checked out into dir
		return "", nil, fmt.Errorf("git worktree add --detach %s %s: %w\n%s", dir, ref, err, out)
	}

	remove := func() error {
		cmd := exec.Command("git", "worktree", "remove", "--force", dir)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("removing temp worktree %s: %w\n%s", dir, err, out)
		}
		return nil
	}
	return dir, remove, nil
}
