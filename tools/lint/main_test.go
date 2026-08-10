package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLinterArgsPinsTheVersionAndForwardsExtras guards the two facts the
// argument vector carries: the linter is fetched by a versioned `go run` (an
// unpinned or PATH-resolved linter is what #426 filed), and the caller's own
// arguments arrive after the `run` subcommand so `go tool lint --fix` works.
func TestLinterArgsPinsTheVersionAndForwardsExtras(t *testing.T) {
	got := linterArgs([]string{"--fix", "./xsd/..."})
	want := []string{"run", linterPin, "run", "--fix", "./xsd/..."}

	if len(got) != len(want) {
		t.Fatalf("linterArgs = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("linterArgs = %q, want %q", got, want)
		}
	}
	if !strings.Contains(linterPin, "@v") {
		t.Errorf("linterPin = %q, want a `pkg@version` pin", linterPin)
	}
}

// TestCacheEnvIsUnderTheModuleRoot is the #450/#650 guard: the child's cache
// must be inside the checkout being linted, so two checkouts of one commit
// cannot share a cache and replay each other's absolute paths.
func TestCacheEnvIsUnderTheModuleRoot(t *testing.T) {
	root := t.TempDir()

	assign := cacheEnv(root)
	name, value, ok := strings.Cut(assign, "=")
	if !ok {
		t.Fatalf("cacheEnv(%q) = %q, want a NAME=VALUE assignment", root, assign)
	}
	if name != "GOLANGCI_LINT_CACHE" {
		t.Errorf("cacheEnv sets %q, want GOLANGCI_LINT_CACHE", name)
	}
	if want := filepath.Join(root, ".lintcache"); value != want {
		t.Errorf("cacheEnv(%q) points at %q, want %q", root, value, want)
	}
}

// TestModuleRootIgnoresTheWorkingDirectory is the other half of the same
// guard: gate part 2 must lint the same tree whether it is invoked from the
// module root or from a package directory beneath it.
func TestModuleRootIgnoresTheWorkingDirectory(t *testing.T) {
	fromHere, err := moduleRoot()
	if err != nil {
		t.Fatalf("moduleRoot from the package directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fromHere, "go.mod")); err != nil {
		t.Fatalf("moduleRoot returned %q, which holds no go.mod: %v", fromHere, err)
	}

	t.Chdir(fromHere)
	fromRoot, err := moduleRoot()
	if err != nil {
		t.Fatalf("moduleRoot from the module root: %v", err)
	}
	if fromRoot != fromHere {
		t.Errorf("moduleRoot = %q from the module root but %q from a subdirectory", fromRoot, fromHere)
	}
}
