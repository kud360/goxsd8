package loader

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
)

// fsResolver serves locations as entries of an fs.FS; see FS.
type fsResolver struct {
	fsys fs.FS
}

// FS returns a Resolver that serves each location as a path within fsys
// (an embed.FS, os.DirFS, testing fstest.MapFS, etc.).
//
// location is passed straight to fsys.Open, so fs.ValidPath governs it:
// rooted ("/…"), "." / ".." segment, and Windows-volume paths are rejected
// by the filesystem itself, giving traversal safety for free (engineering
// property, not a spec rule). A missing entry maps to ErrNotFound.
//
// Unlike Dir, FS does NOT canonicalize the resolved location it returns: it
// hands back location as given. fs.FS guarantees no directory-listing
// capability in general, so there is nothing to canonicalize against; when
// fsys is case-insensitive (an os.DirFS over a case-insensitive filesystem,
// say), keeping Resolver's one-resource-one-identity contract by spelling
// locations consistently is the caller's responsibility.
func FS(fsys fs.FS) Resolver {
	return fsResolver{fsys: fsys}
}

// Resolve implements Resolver.
func (r fsResolver) Resolve(namespace, location string) (io.ReadCloser, string, error) {
	f, err := r.fsys.Open(location)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", fmt.Errorf("loader: %q not found in fs: %w", location, ErrNotFound)
	}
	if err != nil {
		return nil, "", fmt.Errorf("loader: opening %q in fs: %w", location, err)
	}
	// f streams from fsys; the caller closes it (P4 — no buffering here).
	return f, location, nil
}
