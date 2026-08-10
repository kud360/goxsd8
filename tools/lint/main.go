// Command lint runs gate part 2 — the machine-checkable subset of
// docs/STYLE.md configured in .golangci.yml — so that one command gives the
// same answer from any working directory, in any checkout, on any machine.
//
// Usage:
//
//	go tool lint                 # gate part 2
//	go tool lint --fix           # extra args are forwarded to the linter
//
// It resolves the module root from `go env GOMOD` and runs the linter there,
// so the caller's working directory cannot change which files are linted or
// how .golangci.yml's path-anchored exclusions match.
//
// It gives each checkout its own results cache, <module root>/.lintcache.
// golangci-lint keys that cache by package content rather than by path, and
// every cached issue carries an absolute filename, so two checkouts of one
// commit hash identically and a shared cache replays the other checkout's
// issues under the other checkout's absolute paths. Rendered relative to this
// checkout those paths escape the tree and no anchored exclusion pattern can
// match them (#450); once the other checkout is deleted the generated-file
// filter cannot open them at all, and a filter that errors stops excluding, so
// generated files leak through as formatting hits (#650). A per-checkout cache
// makes the replay impossible by construction, and a removed worktree takes
// its cache with it.
//
// The linter is fetched by a version-pinned `go run`, which ignores the
// enclosing module: go.mod and go.sum stay free of the linter's dependency
// tree, so none of it reaches a consumer of the library, and no separately
// installed binary has to be on PATH (#426). A cold module cache therefore
// needs network access.
//
// It exits 0 for a clean run and non-zero otherwise, and 2 when the linter
// never ran at all (no module root, or a `go` that could not be started). Test
// zero against non-zero, not a particular code: `go run` reports its child's
// failure as "exit status N" on stderr and then exits 1 itself, so the
// linter's own codes (1 for findings, 3 for a bad flag or config) all arrive
// here as 1.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// linterPin is the golangci-lint the gate runs, package and version. It is a
// pin: the version decides what the gate rejects, so bumping it is a
// deliberate act with its own review, never a side effect of another change.
const linterPin = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2"

// cacheDirName is the linter's results cache, relative to the module root. It
// lives inside the checkout — and is gitignored there — because being
// per-checkout is the whole point (#450, #650).
const cacheDirName = ".lintcache"

// cacheEnvVar is the environment variable golangci-lint reads for its cache
// location.
const cacheEnvVar = "GOLANGCI_LINT_CACHE"

func main() {
	code, err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "lint: %v\n", err)
		os.Exit(2)
	}
	os.Exit(code)
}

// run executes the pinned linter at the module root with a per-checkout cache,
// streaming its output through unchanged, and returns the child's exit code.
// An error return means `go` itself could not be started; a non-zero code with
// a nil error means the child ran and failed, which for a well-formed
// invocation means the linter reported findings.
func run(extra []string) (int, error) {
	root, err := moduleRoot()
	if err != nil {
		return 0, err
	}

	cmd := exec.Command("go", linterArgs(extra)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), cacheEnv(root))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, fmt.Errorf("running %s in %s: %w", linterPin, root, err)
}

// linterArgs builds the `go` argument vector: `go run <pin> run`, with the
// caller's own arguments appended so `go tool lint --fix` reaches the linter.
func linterArgs(extra []string) []string {
	args := []string{"run", linterPin, "run"}
	return append(args, extra...)
}

// cacheEnv renders the cache assignment added to the child's environment. It
// is appended after os.Environ, so it overrides an inherited shared cache
// rather than deferring to it.
func cacheEnv(root string) string {
	return cacheEnvVar + "=" + filepath.Join(root, cacheDirName)
}

// moduleRoot reports the directory holding the go.mod that governs the current
// working directory, so the gate lints the same tree wherever it is invoked
// from. It asks the go command rather than walking upward itself: `go env
// GOMOD` already answers exactly this, and a second walker would be a second
// encoding of module resolution.
func moduleRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("running go env GOMOD: %w", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		return "", fmt.Errorf("no module found from the working directory (go env GOMOD reported %q)", gomod)
	}
	return filepath.Dir(gomod), nil
}
