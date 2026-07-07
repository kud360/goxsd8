package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// dirResolver serves locations as paths under a filesystem root; see Dir.
type dirResolver struct {
	root string
}

// Dir returns a Resolver that serves each location as a path relative to
// the root directory path.
//
// location is treated as an untrusted, possibly attacker-supplied hint
// (xsi:schemaLocation flows through the same seam), so Dir refuses any
// location that would escape root via ".." segments or an absolute
// override, mapping such attempts to ErrNotFound. This traversal defense is
// an engineering decision, NOT a spec rule — no XSD text governs it.
func Dir(path string) Resolver {
	return dirResolver{root: path}
}

// Resolve implements Resolver.
func (d dirResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	root := filepath.Clean(d.root)
	full := filepath.Join(root, location)

	// Confinement check (engineering, not spec-derived): the cleaned join
	// must remain inside root. filepath.Join has already collapsed ".."
	// segments, so a location that climbs out yields a rel starting "..".
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, "", fmt.Errorf("loader: location %q escapes root %q: %w", location, d.root, ErrNotFound)
	}

	f, err := os.Open(full)
	if os.IsNotExist(err) {
		return nil, "", fmt.Errorf("loader: %q not found under %q: %w", location, d.root, ErrNotFound)
	}
	if err != nil {
		return nil, "", fmt.Errorf("loader: opening %q under %q: %w", location, d.root, err)
	}
	// f streams from disk; the caller closes it (P4 — no buffering here).
	return f, full, nil
}
